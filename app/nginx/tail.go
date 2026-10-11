package nginx

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"syscall"
	"time"
)

// Tailer streams new lines from a growing file (nginx access log). It handles rotation
// (inode change, same path) and truncation (file shrinks) by reopening transparently.
// Polling-based so the implementation stays dependency-free across platforms; the poll
// interval is tight enough for operational log following and loose enough to idle.
type Tailer struct {
	Path         string        // file to follow
	FromStart    bool          // read from offset 0 on first open; default seeks to EOF
	PollInterval time.Duration // default 500 ms
}

const (
	defaultPollInterval = 500 * time.Millisecond
	// maxLineBytes caps a single logical line so a pathological UA header or malicious
	// writer cannot force unbounded growth. Realistic nginx access-log lines run ~1-2 KB.
	maxLineBytes = 1 << 20 // 1 MiB
)

// tailState holds the per-open file state the poll loop mutates. It lives here instead
// of as free variables inside Run so Run can delegate to small methods and stay below
// cyclomatic complexity 30.
type tailState struct {
	file             *os.File
	reader           *bufio.Reader
	inode            uint64
	partial          []byte
	skipUntilNewline bool // dropping chunks of an over-length line until its newline arrives
}

// Run follows the file until ctx is done. Every full line (newline-terminated) is sent
// on out exactly once; a partial trailing fragment is held until the newline arrives.
// Run closes out on return so a downstream `for line := range out` exits cleanly without
// the caller having to arrange a separate done signal.
func (t Tailer) Run(ctx context.Context, out chan<- string) {
	defer close(out)

	interval := t.PollInterval
	if interval <= 0 {
		interval = defaultPollInterval
	}

	st := &tailState{}
	// first open may fail (file absent at startup); retry on the poll loop until ctx done.
	if err := t.open(st); err != nil {
		if !waitOrClosed(ctx, interval) {
			return
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if st.file != nil {
			if !t.drain(ctx, st, out) {
				return
			}
		}
		if st.file == nil {
			if err := t.open(st); err != nil {
				if !waitOrClosed(ctx, interval) {
					return
				}
				continue
			}
		} else if reopened, cont := t.checkRotation(ctx, st, interval, out); cont {
			if !reopened {
				return
			}
			continue
		}

		select {
		case <-ctx.Done():
			if st.file != nil {
				_ = st.file.Close()
			}
			return
		case <-ticker.C:
		}
	}
}

// open seeks to the right offset and resets per-file state so a reopen starts clean.
// After the first open any subsequent reopen reads from offset 0, so FromStart is
// flipped on so later opens stream the whole post-rotation file.
func (t *Tailer) open(st *tailState) error {
	if st.file != nil {
		_ = st.file.Close()
	}
	nf, err := os.Open(t.Path)
	if err != nil {
		return err
	}
	if !t.FromStart {
		if _, err := nf.Seek(0, io.SeekEnd); err != nil {
			_ = nf.Close()
			return err
		}
	}
	fi, err := nf.Stat()
	if err != nil {
		_ = nf.Close()
		return err
	}
	st.inode = fileInode(fi)
	st.file = nf
	st.reader = bufio.NewReader(nf)
	// release the buffer rather than slicing to zero: append grows capacity and never
	// shrinks it, so slicing-to-zero after an oversized line would keep the giant backing
	// array and silently fail the maxLineBytes check for everything after.
	st.partial = nil
	st.skipUntilNewline = false
	t.FromStart = true
	return nil
}

// drain reads all currently available lines; returns false if ctx cancelled mid-send.
func (t *Tailer) drain(ctx context.Context, st *tailState, out chan<- string) bool {
	for {
		chunk, err := st.reader.ReadBytes('\n')
		if len(chunk) > 0 {
			if !appendAndMaybeEmit(ctx, st, chunk, out) {
				_ = st.file.Close()
				return false
			}
		}
		if err == nil {
			continue
		}
		if errors.Is(err, io.EOF) {
			return true
		}
		// a read error: close and let the poll loop reopen
		_ = st.file.Close()
		st.file = nil
		return true
	}
}

// appendAndMaybeEmit extends st.partial with chunk; on newline it emits a full line.
// On overflow (one logical line exceeds maxLineBytes) it enters a "skip until newline"
// state, drops every byte of the overlong line including this chunk, and resumes normal
// emission from the first line after it. Returns false only on ctx cancellation during
// the send.
//
// The length-based check and the full slice release matter: capacity grows and never
// shrinks via append, so `cap(st.partial) > N` plus `st.partial = st.partial[:0]`
// permanently poisons the tailer after one oversized line — every subsequent append
// still exceeds N and gets discarded. Using len() plus nil-ing the slice resets both.
func appendAndMaybeEmit(ctx context.Context, st *tailState, chunk []byte, out chan<- string) bool {
	// skip state: the current logical line already overflowed; drain chunks until we
	// see the terminating newline, then drop both the state and the carry buffer.
	if st.skipUntilNewline {
		if chunk[len(chunk)-1] == '\n' {
			st.skipUntilNewline = false
			st.partial = nil
		}
		return true
	}

	// pre-check: if adding this chunk would push the carry buffer past the cap, enter
	// skip state. If the chunk already carries its own newline, settle the state inline.
	if len(st.partial)+len(chunk) > maxLineBytes {
		if chunk[len(chunk)-1] == '\n' {
			st.partial = nil
			return true
		}
		st.skipUntilNewline = true
		st.partial = nil
		return true
	}

	st.partial = append(st.partial, chunk...)
	if chunk[len(chunk)-1] != '\n' {
		return true
	}
	line := string(st.partial[:len(st.partial)-1])
	// trim trailing CR for files written on Windows-like hosts
	if n := len(line); n > 0 && line[n-1] == '\r' {
		line = line[:n-1]
	}
	st.partial = st.partial[:0]
	return sendLine(ctx, out, line)
}

// checkRotation stats the path and reopens on inode change or truncation.
// Returns (reopened, cont): cont=true means skip the ticker wait; reopened=false means
// a reopen failed and the ctx was cancelled during the retry delay.
//
// On inode change we drain the old fd one final time before open() closes it: any lines
// nginx wrote between drain's EOF and this stat land on the old inode and would otherwise
// be lost. The grace drain closes that millisecond-wide window; a bigger "keep the old fd
// open across a few polls" grace covers the window where nginx hasn't yet reopened after
// logrotate's rename+create but is left for a follow-up — the typical loss there is a
// handful of lines on daily rotations.
func (t *Tailer) checkRotation(ctx context.Context, st *tailState, interval time.Duration, out chan<- string) (reopened, cont bool) {
	fi, err := os.Stat(t.Path)
	if err != nil {
		// path disappeared; keep current fd draining until a newer open succeeds
		return true, false
	}
	if fileInode(fi) == st.inode && fi.Size() >= currentOffset(st.file) {
		return true, false
	}
	// rotation (inode changed) or truncation (size < offset): drain the old fd for any
	// trailing lines nginx wrote between the drain-at-the-top and this stat, then reopen
	// from the start of the new file. The out channel is the same one Run drains to.
	if st.file != nil && !t.drain(ctx, st, out) {
		return false, true
	}
	t.FromStart = true
	if err := t.open(st); err != nil {
		if !waitOrClosed(ctx, interval) {
			return false, true
		}
	}
	return true, true
}

// waitOrClosed sleeps for d or returns false if ctx is done first.
func waitOrClosed(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// sendLine delivers a line on out, respecting ctx cancellation so a stuck consumer
// cannot pin the tailer. Returns false if ctx is done.
func sendLine(ctx context.Context, out chan<- string, line string) bool {
	select {
	case out <- line:
		return true
	case <-ctx.Done():
		return false
	}
}

// fileInode returns the inode number from a FileInfo (Linux + Darwin). On platforms
// that don't expose a *syscall.Stat_t it falls back to 0, disabling rotation detection
// by inode (the size-shrink path still catches truncation).
func fileInode(fi os.FileInfo) uint64 {
	if sys, ok := fi.Sys().(*syscall.Stat_t); ok {
		return uint64(sys.Ino) //nolint:gosec,unconvert // Darwin returns uint64, Linux returns uint64 on amd64; the convert keeps 32-bit arches happy
	}
	return 0
}

// currentOffset reports where the next read will come from; used to detect truncation
// (file shrank below our reader position). Errors treated as "unknown, pretend offset 0"
// so a transient Seek failure doesn't trigger a spurious reopen loop.
func currentOffset(f *os.File) int64 {
	off, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0
	}
	return off
}

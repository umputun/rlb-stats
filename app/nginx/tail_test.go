package nginx

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collect runs t.Run in a goroutine and returns a function that stops it and returns
// everything received so far. Tailer is started with a small poll interval so tests
// finish in well under the default 500 ms.
func startTailer(t *testing.T, path string, fromStart bool) (ctx context.Context, cancel context.CancelFunc, lines chan string, wait func()) {
	t.Helper()
	ctx, cancel = context.WithCancel(context.Background())
	lines = make(chan string, 64)
	tl := Tailer{Path: path, FromStart: fromStart, PollInterval: 20 * time.Millisecond}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		tl.Run(ctx, lines)
	}()
	wait = wg.Wait
	return ctx, cancel, lines, wait
}

func collectUntil(t *testing.T, lines chan string, want int, deadline time.Duration) []string {
	t.Helper()
	got := make([]string, 0, want)
	timeout := time.After(deadline)
	for len(got) < want {
		select {
		case line, ok := <-lines:
			if !ok {
				return got
			}
			got = append(got, line)
		case <-timeout:
			return got
		}
	}
	return got
}

func TestTailerReadsFromStart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	require.NoError(t, os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o600))

	_, cancel, lines, wait := startTailer(t, path, true)
	defer wait()
	defer cancel()

	got := collectUntil(t, lines, 3, time.Second)
	assert.Equal(t, []string{"one", "two", "three"}, got)
}

func TestTailerAppendsAfterOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o600))

	_, cancel, lines, wait := startTailer(t, path, true)
	defer wait()
	defer cancel()

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	require.NoError(t, err)
	defer f.Close()

	_, err = f.WriteString("first\nsecond\n")
	require.NoError(t, err)

	got := collectUntil(t, lines, 2, time.Second)
	assert.Equal(t, []string{"first", "second"}, got)
}

func TestTailerHandlesPartialLine(t *testing.T) {
	// Writer flushes an incomplete line, then completes it: the tailer must hold the
	// fragment until the newline arrives, not emit "hal" and "f".
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o600))

	_, cancel, lines, wait := startTailer(t, path, true)
	defer wait()
	defer cancel()

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	require.NoError(t, err)
	defer f.Close()

	_, err = f.WriteString("hal")
	require.NoError(t, err)
	// a brief wait so the tailer gets a chance to read the partial fragment
	time.Sleep(80 * time.Millisecond)
	_, err = f.WriteString("f\n")
	require.NoError(t, err)

	got := collectUntil(t, lines, 1, time.Second)
	assert.Equal(t, []string{"half"}, got)
}

func TestCheckRotationDrainsOldFdOnInodeChange(t *testing.T) {
	// Deterministic coverage for the drain-before-reopen fix in checkRotation.
	// The race window the fix closes is between drain's EOF and os.Stat: nginx writes a
	// trailing line to the old inode while rlb-stats is between those two calls. Under
	// Run's ticker-driven loop this gap is hard to force in a timing test; the previous
	// try passed with the fix removed. Here we open the file at package level, do an
	// initial drain, write a late line, rename+create, and call checkRotation directly.
	// Mutation check (verified by hand): remove the `t.drain(ctx, st, out)` call inside
	// checkRotation and this test must fail to see "late-on-old-fd".
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	require.NoError(t, os.WriteFile(path, []byte("one\n"), 0o600))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tl := &Tailer{Path: path, FromStart: true, PollInterval: time.Hour /* never auto-ticks */}
	st := &tailState{}
	require.NoError(t, tl.open(st))

	out := make(chan string, 8)

	// 1. baseline drain: reads "one\n" and reaches EOF
	require.True(t, tl.drain(ctx, st, out))
	assert.Equal(t, "one", <-out)

	// 2. the race window: write a late line to the (still open) old inode, THEN rotate
	oldF, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	require.NoError(t, err)
	_, err = oldF.WriteString("late-on-old-fd\n")
	require.NoError(t, err)
	require.NoError(t, oldF.Close())
	require.NoError(t, os.Rename(path, path+".1"))
	require.NoError(t, os.WriteFile(path, []byte("post-rotate\n"), 0o600))

	// 3. checkRotation must detect the inode change, drain the OLD fd (surfacing the
	//    late line), close it, and reopen the new file. We then drain the new file too.
	reopened, cont := tl.checkRotation(ctx, st, 20*time.Millisecond, out)
	assert.True(t, reopened)
	assert.True(t, cont)
	require.True(t, tl.drain(ctx, st, out))

	close(out)
	var got []string
	for line := range out {
		got = append(got, line)
	}
	assert.Equal(t, []string{"late-on-old-fd", "post-rotate"}, got,
		"late write to old inode must drain before reopen; new inode must then stream")
}

func TestTailerHandlesRotation(t *testing.T) {
	// classic logrotate: rename access.log -> access.log.1, create new access.log.
	// The tailer must detect the new inode and read the new file from offset 0.
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	require.NoError(t, os.WriteFile(path, []byte("pre-rotate\n"), 0o600))

	_, cancel, lines, wait := startTailer(t, path, true)
	defer wait()
	defer cancel()

	// confirm the pre-rotate line surfaced before rotating
	pre := collectUntil(t, lines, 1, time.Second)
	assert.Equal(t, []string{"pre-rotate"}, pre)

	require.NoError(t, os.Rename(path, path+".1"))
	require.NoError(t, os.WriteFile(path, []byte("post-rotate\n"), 0o600))

	got := collectUntil(t, lines, 1, 2*time.Second)
	assert.Equal(t, []string{"post-rotate"}, got, "lines after rotation must stream from the new inode")
}

func TestTailerHandlesTruncation(t *testing.T) {
	// logrotate copytruncate: ": > access.log" truncates in place, nginx continues
	// appending afterwards. The explicit truncate step is visible to the tailer's
	// polling, triggering reopen; a subsequent appender writes the new content.
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	require.NoError(t, os.WriteFile(path, []byte("initial\n"), 0o600))

	_, cancel, lines, wait := startTailer(t, path, true)
	defer wait()
	defer cancel()

	pre := collectUntil(t, lines, 1, time.Second)
	assert.Equal(t, []string{"initial"}, pre)

	// explicit truncate — gives the tailer a chance to see size < offset and reopen
	require.NoError(t, os.Truncate(path, 0))
	time.Sleep(100 * time.Millisecond)

	// nginx-style append after truncation
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	require.NoError(t, err)
	_, err = f.WriteString("after-trunc\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	got := collectUntil(t, lines, 1, 2*time.Second)
	assert.Equal(t, []string{"after-trunc"}, got)
}

func TestTailerSeeksToEndByDefault(t *testing.T) {
	// Production use: start at EOF so a restart doesn't re-ingest already-counted history.
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	require.NoError(t, os.WriteFile(path, []byte("history-A\nhistory-B\n"), 0o600))

	_, cancel, lines, wait := startTailer(t, path, false /* fromStart */)
	defer wait()
	defer cancel()

	// nothing should be emitted from the pre-existing content; a short wait proves it
	select {
	case line := <-lines:
		t.Fatalf("tailer emitted pre-existing line %q but should seek to EOF", line)
	case <-time.After(150 * time.Millisecond):
	}

	// a new line after startup must still land
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString("fresh\n")
	require.NoError(t, err)

	got := collectUntil(t, lines, 1, time.Second)
	assert.Equal(t, []string{"fresh"}, got)
}

func TestTailerClosesChannelOnCancel(t *testing.T) {
	// Downstream consumer should see the channel close so a `for range` exits cleanly.
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o600))

	ctx, cancel, lines, wait := startTailer(t, path, true)
	_ = ctx
	cancel()

	// drain whatever's buffered, then expect closure within a short deadline
	closed := false
	timeout := time.After(time.Second)
loop:
	for {
		select {
		case _, ok := <-lines:
			if !ok {
				closed = true
				break loop
			}
		case <-timeout:
			break loop
		}
	}
	assert.True(t, closed, "lines channel must close when ctx is cancelled")
	wait()
}

func TestTailerSurvivesOversizedLine(t *testing.T) {
	// regression trap for the critical finding: an oversized line used to permanently
	// poison the carry buffer. revmux reproduced the bug with a scratch tail.go after a
	// 2 MiB line, two normal lines, by `cap() > N` plus `s = s[:0]` (capacity sticks).
	// This test writes an over-cap line sandwiched between normal lines and asserts the
	// tailer drops the oversized one but keeps emitting the normal ones.
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o600))

	_, cancel, lines, wait := startTailer(t, path, true)
	defer wait()
	defer cancel()

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	require.NoError(t, err)
	defer f.Close()

	_, err = f.WriteString("before\n")
	require.NoError(t, err)
	// one line well above maxLineBytes (1 MiB) — pathologically long UA, malicious writer
	huge := make([]byte, 2*1024*1024)
	for i := range huge {
		huge[i] = 'x'
	}
	huge = append(huge, '\n')
	_, err = f.Write(huge)
	require.NoError(t, err)
	_, err = f.WriteString("after1\nafter2\n")
	require.NoError(t, err)

	got := collectUntil(t, lines, 3, 3*time.Second)
	assert.Equal(t, []string{"before", "after1", "after2"}, got,
		"oversized line must be dropped but subsequent lines must still emit")
}

func TestTailerSurvivesMissingFileAtStartup(t *testing.T) {
	// Operational case: rlb-stats boots before nginx creates its log. The tailer must
	// retry until the file appears, not crash or exit.
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")

	_, cancel, lines, wait := startTailer(t, path, true)
	defer wait()
	defer cancel()

	// no file yet: nothing emitted
	select {
	case line := <-lines:
		t.Fatalf("tailer emitted %q before the file existed", line)
	case <-time.After(80 * time.Millisecond):
	}

	// now create and populate
	require.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o600))

	got := collectUntil(t, lines, 1, 2*time.Second)
	assert.Equal(t, []string{"hello"}, got)
}

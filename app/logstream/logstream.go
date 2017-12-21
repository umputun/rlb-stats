package logstream

import (
	"io"
	"io/ioutil"
	"log"
	"time"

	"bytes"

	"github.com/fsouza/go-dockerclient"
)

// LogStreamer connects and activates container's log stream with io.Writer
type LogStreamer struct {
	DockerClient *docker.Client
	ContainerID  string
	LogWriter    io.Writer
}

// LineExtractor have buffer to store bytes before \n happen and channel to return complete line
type LineExtractor struct {
	ch  chan string
	buf []byte
}

// NewLineExtractor create LineExtractor
func NewLineExtractor() *LineExtractor {
	return &LineExtractor{ch: make(chan string)}
}

// Go activates streamer
func (l *LogStreamer) Go() {
	log.Printf("[INFO] start log streamer for %s", l.ContainerID)
	go func() {
		logOpts := docker.LogsOptions{
			Container:         l.ContainerID,
			OutputStream:      l.LogWriter,    // logs writer for stdout
			ErrorStream:       ioutil.Discard, // err writer for stderr
			Tail:              "10",
			Follow:            true,
			Stdout:            true,
			Stderr:            true,
			InactivityTimeout: time.Hour * 10000,
		}
		err := l.DockerClient.Logs(logOpts) // this is blocking call. Will run until container up and will publish to streams
		log.Printf("[INFO] stream from %s terminated, %v", l.ContainerID, err)
	}()
}

// Write complete strings into channel
func (le *LineExtractor) Write(p []byte) (n int, err error) {
	le.buf = append(le.buf, p...)

	for bytes.Count(le.buf, []byte{'\n'}) > 0 {
		if n := bytes.IndexByte(le.buf, '\n'); n >= 0 {
			line := string(le.buf[:n])
			le.ch <- string(line)
			le.buf = le.buf[n+1:]
		}
	}
	return len(p), nil
}

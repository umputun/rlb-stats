package logservice

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"

	"github.com/umputun/rlb-stats/app/store"
)

func TestLogService(t *testing.T) {
	const regEx = `^(?P<Date>.+) - (?:.+) - (?P<FileName>.+) - (?P<SourceIP>.+) - (?:.+) - (?P<AnswerTime>.+) - https?://(?P<DestinationNode>.+?)/.+$`
	const defaultDateFormat = `2006/01/02 15:04:05`
	s, err := store.NewBolt("/tmp/test.bd")
	assert.Nil(t, err, "engine created")
	l := LogService{
		DockerHost:    "unix:///dev/null",
		ContainerName: "nginx",
		Engine:        s,
		RegEx:         regEx,
		DateFormat:    defaultDateFormat,
		LogTail:       "all",
	}

	if os.Getenv("CRASH_TEST") == "1" {
		l.Go()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestLogService")
	cmd.Env = append(os.Environ(), "CRASH_TEST=1")
	err = cmd.Run()
	e, ok := err.(*exec.ExitError)
	assert.NotEqual(t, e.Success(), ok, "run is not OK")
	assert.EqualValues(t, "exit status 1", fmt.Sprintf("%v", e), "exit status 1 because of broken docker connection")
	logExtractor := newLineExtractor()
	assert.Panics(t, func() { getLogStreamer(&docker.Client{}, l.ContainerName, l.LogTail, logExtractor) }, "memory problem because of pointer to nothing")

	parser := getParser(l.RegEx, l.DateFormat)
	startLogStreamer(logStreamer{}, parser, logExtractor, l.Engine)
}

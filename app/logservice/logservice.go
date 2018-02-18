package logservice

import (
	"log"

	"github.com/fsouza/go-dockerclient"
	"github.com/umputun/rlb-stats/app/store"
)

// LogService connects to specified container, gathers and stores it's logs
type LogService struct {
	DockerHost    string
	ContainerName string
	Engine        store.Engine
	RegEx         string
	DateFormat    string
	LogTail       string
}

// Go starts LogService
func (l *LogService) Go() {
	log.Printf("[INFO] get %v loglines from container %v and listen for new ones", l.LogTail, l.ContainerName)
	parser := getParser(l.RegEx, l.DateFormat)
	dockerClient := getDocker(l.DockerHost)
	logExtractor := newLineExtractor()
	logStreamer := getLogStreamer(dockerClient, l.ContainerName, l.LogTail, logExtractor)
	startLogStreamer(logStreamer, parser, logExtractor, l.Engine)
}

// getParser create and validates parser
func getParser(regEx string, dateFormat string) *Parser {
	parser, err := newParser(regEx, dateFormat)
	if err != nil {
		log.Fatalf("[ERROR] can't validate regex, %v", err)
	}
	return parser
}

// getDocker connects to docker
func getDocker(endpoint string) *docker.Client {
	dockerClient, err := docker.NewClient(endpoint)
	if err != nil {
		log.Fatalf("[ERROR] can't initialise docker client, %v", err)
	}
	return dockerClient
}

// getLogStreamer connects to container and returns logStreamer
func getLogStreamer(d *docker.Client, containerName string, tailOption string, le *lineExtractor) logStreamer {
	imageInfo, err := d.InspectContainer(containerName)
	if err != nil {
		log.Fatalf("[ERROR] can't get container id for %s, %v", containerName, err)
	}
	if imageInfo.State.Status != "running" {
		log.Fatalf("[ERROR] container %s is not running, status %s", containerName, imageInfo.State.Status)
	}

	ls := logStreamer{
		DockerClient:  d,
		ContainerName: containerName,
		ContainerID:   imageInfo.ID,
		LogWriter:     le,
		Tail:          tailOption,
	}
	return ls
}

func startLogStreamer(ls logStreamer, p *Parser, le *lineExtractor, storage store.Engine) {

	ls.Go()     // start listening to container logs
	go func() { // start parser on logs
		for line := range le.Ch() {
			entry, err := p.Do(line)
			if err == nil {
				if candle, ok := p.submit(entry); ok { // Submit returns ok in case candle is ready
					err = storage.Save(candle)
					if err != nil {
						log.Printf("[ERROR] couldn't write candle to storage, %v", err)
					}
				}
			}
		}
	}()
}

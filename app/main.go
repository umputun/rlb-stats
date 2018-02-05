package main

import (
	"log"

	"os"

	"github.com/jessevdk/go-flags"

	"github.com/fsouza/go-dockerclient"
	"github.com/umputun/rlb-stats/app/convert"
	"github.com/umputun/rlb-stats/app/logstream"
	"github.com/umputun/rlb-stats/app/parse"
	"github.com/umputun/rlb-stats/app/rest"
	"github.com/umputun/rlb-stats/app/store"
)

var opts struct {
	BoltDB        string `long:"bolt" env:"BOLT_FILE" default:"/tmp/rlb-stats.bd" description:"boltdb file"`
	Port          int    `long:"port" env:"PORT" default:"8080" description:"REST server port"`
	ContainerName string `long:"container_name" env:"CONTAINER_NAME" default:"" description:"container name"`
	DockerHost    string `long:"docker" env:"DOCKER_HOST" default:"unix:///var/run/docker.sock" description:"docker host"`
	LogTail       string `long:"log_tail" env:"LOG_TAIL" default:"0" description:"How many log entries to load from container, set to 'all' on the first run"`
	RegEx         string `long:"regexp" env:"REGEXP" description:"log line regexp" default:"^(?P<Date>.+) - (?:.+) - (?P<FileName>.+) - (?P<SourceIP>.+) - (?:.+) - (?P<AnswerTime>.+) - https?://(?P<DestinationNode>.+?)/.+$"`
	DateFormat    string `long:"date_format" env:"DATE_FORMAT" description:"format of the date in log line" default:"2006/01/02 15:04:05"`
	Dbg           bool   `long:"dbg" description:"debug mode"`
}

var revision string

func main() {
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}

	log.SetFlags(log.Ldate | log.Ltime)
	if opts.Dbg {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	}
	log.Printf("rlb-stats %s", revision)
	storage := getEngine(opts.BoltDB)
	if opts.ContainerName != "" { // start container log streamer and parse logic only if there is container
		parser := getParser(opts.RegEx, opts.DateFormat)
		dockerClient := getDocker(opts.DockerHost)
		logExtractor := logstream.NewLineExtractor()
		logStreamer := getLogStreamer(dockerClient, opts.ContainerName, opts.LogTail, logExtractor)
		startLogStreamer(logStreamer, parser, logExtractor, storage)
	}
	server := rest.Server{
		Engine: storage,
		Port:   opts.Port,
	}
	server.Run()
}

func getEngine(boltFile string) store.Engine {
	storage, err := store.NewBolt(boltFile)
	if err != nil {
		log.Fatalf("[ERROR] can't open db, %v", err)
	}
	return storage
}

func getParser(regEx string, dateFormat string) *parse.Parser {
	parser, err := parse.New(regEx, dateFormat)
	if err != nil {
		log.Fatalf("[ERROR] can't validate regex, %v", err)
	}
	return parser
}

func getDocker(endpoint string) *docker.Client {
	dockerClient, err := docker.NewClient(endpoint)
	if err != nil {
		log.Fatalf("[ERROR] can't initialise docker client, %v", err)
	}
	return dockerClient
}

func getLogStreamer(d *docker.Client, containerName string, tailOption string, le *logstream.LineExtractor) logstream.LogStreamer {
	imageInfo, err := d.InspectContainer(containerName)
	if err != nil {
		log.Fatalf("[ERROR] can't get container id for %s, %v", containerName, err)
	}
	if imageInfo.State.Status != "running" {
		log.Fatalf("[ERROR] container %s is not running, status %s", containerName, imageInfo.State.Status)
	}

	logStreamer := logstream.LogStreamer{
		DockerClient:  d,
		ContainerName: containerName,
		ContainerID:   imageInfo.ID,
		LogWriter:     le,
		Tail:          tailOption,
	}
	return logStreamer
}

func startLogStreamer(ls logstream.LogStreamer, p *parse.Parser, le *logstream.LineExtractor, storage store.Engine) {

	ls.Go()     // start listening to container logs
	go func() { // start parser on logs
		for line := range le.Ch() {
			entry, err := p.Do(line)
			if err == nil {
				if candle, ok := convert.Submit(entry); ok { // Submit returns ok in case candle is ready
					err = storage.Save(candle)
					if err != nil {
						log.Printf("[ERROR] couldn't write candle to storage, %v", err)
					}
				}
			}
		}
	}()
}

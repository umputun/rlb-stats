package main

import (
	"log"

	"os"

	"github.com/jessevdk/go-flags"

	"github.com/umputun/rlb-stats/app/api"
	"github.com/umputun/rlb-stats/app/logservice"
	"github.com/umputun/rlb-stats/app/store"
	"github.com/umputun/rlb-stats/app/web"
)

var opts struct {
	BoltDB        string `long:"bolt" env:"BOLT_FILE" default:"/tmp/rlb-stats.bd" description:"boltdb file"`
	Port          int    `long:"port" env:"PORT" default:"8080" description:"REST server port"`
	UIPort        int    `long:"ui_port" env:"UI_PORT" default:"8000" description:"UI server port"`
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
	if opts.ContainerName != "" { // start log streamer and parse logic only if there is container
		logServ := logservice.LogService{
			DockerHost:    opts.DockerHost,
			ContainerName: opts.ContainerName,
			Engine:        storage,
			RegEx:         opts.RegEx,
			DateFormat:    opts.DateFormat,
			LogTail:       opts.LogTail,
		}
		logServ.Go()
	}
	restServ := api.Server{
		Engine:  storage,
		Port:    opts.Port,
		Version: revision,
	}
	uiServer := web.Server{
		Port:     opts.UIPort,
		RESTPort: opts.Port,
	}
	go restServ.Run()
	uiServer.Run()
}

func getEngine(boltFile string) store.Engine {
	storage, err := store.NewBolt(boltFile)
	if err != nil {
		log.Fatalf("[ERROR] can't open db, %v", err)
	}
	return storage
}

package main

import (
	"log"

	"os"

	"github.com/jessevdk/go-flags"

	"github.com/umputun/rlb-stats/app/parse"
	"github.com/umputun/rlb-stats/app/store"
)

var opts struct {
	RegEx         string `long:"regexp" env:"REGEXP" description:"log line regexp" default:"^(?P<Date>.+) - (?:.+) - (?P<FileName>.+) - (?P<SourceIP>.+) - (?:.+) - (?P<AnswerTime>.+) - https?://(?P<DestinationNode>.+?)/.+$"`
	ContainerName string `long:"container_name" env:"CONTAINER_NAME" default:"" description:"container name"`
	BoltDB        string `long:"bolt" env:"BOLT_FILE" default:"/tmp/rlb-stats.bd" description:"boltdb file"`
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
	getEngine(opts.BoltDB)
	getParser(opts.RegEx)
}

func getEngine(boltFile string) store.Engine {
	storage, err := store.NewBolt(boltFile)
	if err != nil {
		log.Fatalf("[ERROR] can't open db, %v", err)
	}
	return storage
}

func getParser(regEx string) *parse.Parser {
	parser, err := parse.New(regEx)
	if err != nil {
		log.Fatalf("[ERROR] can't validate regex, %v", err)
	}
	return parser
}

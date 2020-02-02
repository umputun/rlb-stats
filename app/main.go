package main

import (
	"os"

	log "github.com/go-pkgz/lgr"
	"github.com/jessevdk/go-flags"

	"github.com/umputun/rlb-stats/app/store"
	"github.com/umputun/rlb-stats/app/web"
)

type opts struct {
	BoltDB string `long:"bolt" env:"BOLT_FILE" default:"/tmp/rlb-stats.bd" description:"boltdb file path"`
	Port   int    `long:"port" env:"PORT" default:"8080" description:"Web server port"`
	Dbg    bool   `long:"dbg" env:"DEBUG" description:"debug mode"`
}

var revision string

func main() {
	var opts opts
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}

	log.Setup(log.Msec, log.LevelBraces)
	if opts.Dbg {
		log.Setup(log.Debug, log.CallerFile, log.Msec, log.LevelBraces)
	}
	log.Printf("rlb-stats %s", revision)
	storage := getEngine(opts.BoltDB)
	webServer := web.Server{
		Engine:     storage,
		Aggregator: &store.Aggregator{},
		Port:       opts.Port,
		Version:    revision,
	}
	webServer.Run()
}

func getEngine(boltFile string) store.Engine {
	storage, err := store.NewBolt(boltFile)
	if err != nil {
		log.Fatalf("[ERROR] can't open db, %v", err)
	}
	return storage
}

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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

	if revision == "" {
		revision = "unknown"
	}
	log.Printf("rlb-stats %s", revision)

	storage := getEngine(opts.BoltDB)
	aggregator := &store.Aggregator{}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	webServer := web.Server{
		Engine:     storage,
		Aggregator: aggregator,
		Port:       opts.Port,
		Version:    revision,
	}
	webServer.Run(ctx)

	// shutdown sequence: flush aggregator and close storage
	if candle, ok := aggregator.Flush(); ok {
		if err := storage.Save(candle); err != nil {
			log.Printf("[WARN] failed to save flushed candle, %s", err)
		} else {
			log.Printf("[INFO] flushed aggregator candle on shutdown")
		}
	}
	if err := storage.Close(); err != nil {
		log.Printf("[WARN] failed to close bolt, %s", err)
	}
	log.Printf("[INFO] rlb-stats terminated")
}

func getEngine(boltFile string) *store.Bolt {
	storage, err := store.NewBolt(boltFile)
	if err != nil {
		log.Fatalf("[ERROR] can't open db, %v", err)
	}
	return storage
}

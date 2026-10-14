package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/go-pkgz/lgr"
	"github.com/jessevdk/go-flags"

	"github.com/umputun/rlb-stats/app/nginx"
	"github.com/umputun/rlb-stats/app/store"
	"github.com/umputun/rlb-stats/app/web"
)

type opts struct {
	BoltDB    string `long:"bolt" env:"BOLT_FILE" default:"/tmp/rlb-stats.bd" description:"boltdb file path"`
	Port      int    `long:"port" env:"PORT" default:"8080" description:"Web server port"`
	WebappDir string `long:"webapp" env:"WEBAPP" default:"webapp" description:"directory overriding the embedded UI, used if present"`

	// Nginx tail mode. When NginxLog is set, rlb-stats tails that file and feeds parsed
	// records through the same aggregator and store used by /api/insert. See README for
	// the required log_format directive.
	NginxLog    string   `long:"nginx-log" env:"NGINX_LOG" description:"nginx access log to tail; required log_format in README"`
	NginxMethod []string `long:"nginx-method" env:"NGINX_METHOD" env-delim:"," default:"GET" description:"HTTP methods to count; comma-separated or repeated"`
	NginxStatus []int    `long:"nginx-status" env:"NGINX_STATUS" env-delim:"," default:"200" default:"206" description:"HTTP status codes to count; comma-separated or repeated"`
	NginxPath   []string `long:"nginx-path" env:"NGINX_PATH" env-delim:"," default:"*.mp3" description:"path globs to count (matched on basename); comma-separated or repeated"`
	NginxHost   []string `long:"nginx-host" env:"NGINX_HOST" env-delim:"," description:"nginx $http_host allowlist; empty = all hosts; comma-separated or repeated"`

	Dbg bool `long:"dbg" env:"DEBUG" description:"debug mode"`
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

	// validate the filter before opening the DB or signal handler so a bad flag combination
	// fails fast without having to tear anything down. nil filter is only used when
	// opts.NginxLog is set; the branch below guards that.
	var filter nginx.Filter
	if opts.NginxLog != "" {
		var err error
		filter, err = nginx.NewFilter(opts.NginxMethod, opts.NginxStatus, opts.NginxPath, opts.NginxHost)
		if err != nil {
			log.Fatalf("[ERROR] invalid nginx filter: %v", err)
		}
	}

	storage := getEngine(opts.BoltDB)
	aggregator := &store.Aggregator{}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var tailWG sync.WaitGroup
	if opts.NginxLog != "" {
		log.Printf("[INFO] tailing nginx access log %s (methods=%v status=%v path=%v host=%v)",
			opts.NginxLog, opts.NginxMethod, opts.NginxStatus, opts.NginxPath, opts.NginxHost)

		lines := make(chan string, 256)
		tailWG.Add(2)
		go func() {
			defer tailWG.Done()
			nginx.Tailer{Path: opts.NginxLog}.Run(ctx, lines)
		}()
		go func() {
			defer tailWG.Done()
			nginx.Consume(lines, filter, aggregator, storage)
		}()
	}

	webServer := web.Server{
		Engine:     storage,
		Aggregator: aggregator,
		Port:       opts.Port,
		Version:    revision,
		WebappDir:  opts.WebappDir,
	}
	webServer.Run(ctx)

	// shutdown ordering closes the race the tailer would otherwise leak: wait for the
	// tailer to close its lines channel and for the consumer to drain it, THEN flush
	// the aggregator so no in-flight record is lost on SIGTERM.
	tailWG.Wait()

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

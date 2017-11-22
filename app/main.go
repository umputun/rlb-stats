package main

import (
	"log"
	"time"

	"github.com/umputun/rlb-stats/app/store"
)

const defaultRegEx = "^(?P<Date>.+) - (?:.+) - (?P<FileName>.+) - (?P<SourceIP>.+) - (?:.+) - (?P<AnswerTime>.+) - (?P<DestinationNode>.+)$"
const db_filename = "/tmp/rlb-stats.bd"

// TODO: command-line parameters \ env

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	// store := getEngine(db_filename)
	// regex := parse.InitRegex(defaultRegEx)
}

func getEngine(boltFile string) store.Engine {
	store, err := store.NewBolt(boltFile, time.Minute*1)
	if err != nil {
		log.Fatalf("[ERROR] can't open db, %v", err)
	}
	return store
}

package main

import (
	"log"
)

//const defaultRegEx = "^(?P<Date>.+) - (?:.+) - (?P<FileName>.+) - (?P<SourceIP>.+) - (?:.+) - (?P<AnswerTime>.+) - (?P<DestinationNode>.+)$"
//const dbFilename = "/tmp/rlb-stats.bd"

// TODO: command-line parameters \ env

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	//store := getEngine(dbFilename)
	//parser := parse.New(defaultRegEx)
}

//func getEngine(boltFile string) store.Engine {
//	store, err := store.NewBolt(boltFile)
//	if err != nil {
//		log.Fatalf("[ERROR] can't open db, %v", err)
//	}
//	return store
//}

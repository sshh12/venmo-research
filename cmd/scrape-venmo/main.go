package main

import (
	"flag"
	"log"
	"os"

	"github.com/sshh12/venmo-research/storage"
	"github.com/sshh12/venmo-research/venmo"
)

func main() {
	token := flag.String("token", "", "venmo token")
	shardIdx := flag.Int("shard_idx", 0, "shard index")
	shardCnt := flag.Int("shard_cnt", 1, "total shards")
	workerCnt := flag.Int("workers", 5, "parallel workers")
	startID := flag.Int("start_id", 0, "venmo id to start from")
	endID := flag.Int("end_id", 93000000, "venmo id to end at")
	interval := flag.Int("interval_size", 10000, "number of ids per file")
	randomMode := flag.Bool("random", false, "random mode, just fetch accounts at random (ignores interval and sharding)")
	scrapeMode := flag.String("mode", "transactions", "What to scrape {transactions, geoprofiles}")
	flag.Parse()

	store, err := storage.NewPostgresStore()
	if err != nil {
		log.Fatal(err)
		return
	}

	if *token == "" {
		*token = os.Getenv("VENMO_TOKEN")
		if *token == "" {
			log.Fatal("Token is required")
			return
		}
	}
	client := venmo.NewClient(*token)

	if *scrapeMode == "transactions" {
		RunTransactionScraper(client, store, *randomMode, *workerCnt, *startID, *endID, *interval, *shardIdx, *shardCnt)
	} else if *scrapeMode == "geoprofiles" {
		RunGeoProfilesScraper(store)
	} else {
		log.Fatal("Unknown scrape mode")
	}

}

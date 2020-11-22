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
	selPath := flag.String("sel_path", "selenium-server-standalone-3.141.59.jar", "Path to selenium server")
	selDriver := flag.String("sel_driver", "C:\\Dev\\bin\\chromedriver.exe", "Path to selenium chrome driver")
	selPort := flag.Int("sel_port", 8123, "Port for selenium server")
	selHeadless := flag.Bool("sel_headless", false, "Run selenium with headless option")
	selXvfb := flag.Bool("sel_xvfb", false, "Run selenium with X virtual framebuffer")
	fbUser := flag.String("fb_user", "", "Facebook username")
	fbPass := flag.String("fb_pass", "", "Facebook password")
	scrapeMode := flag.String("mode", "transactions", "What to scrape {transactions, transactions2, namesearch, facebook, facebookpics, peekyoulocs}")
	flag.Parse()

	store, err := storage.NewPostgresStore()
	if err != nil {
		log.Fatal(err)
		return
	}

	if *token == "" {
		*token = os.Getenv("VENMO_TOKEN")
	}
	client := venmo.NewClient(*token)

	if *scrapeMode == "transactions" {
		RunTransactionScraper(client, store, *randomMode, *workerCnt, *startID, *endID, *interval, *shardIdx, *shardCnt)
	} else if *scrapeMode == "transactions2" {
		RunTransaction2Scraper(client, store, *workerCnt)
	} else if *scrapeMode == "namesearch" {
		RunNameSearchScraper(store, *workerCnt)
	} else if *scrapeMode == "facebook" {
		RunFacebookScraper(store, *workerCnt, *selPath, *selDriver, *selPort, *selHeadless, *selXvfb, *fbUser, *fbPass)
	} else if *scrapeMode == "facebookpics" {
		RunFacebookPicsScraper(store, *workerCnt, *selPath, *selDriver, *selPort, *selHeadless, *selXvfb, *fbUser, *fbPass)
	} else if *scrapeMode == "peekyoulocs" {
		RunPeekYouLocScraper(store, *workerCnt, *selPath, *selDriver, *selPort, *selHeadless, *selXvfb, *fbUser, *fbPass)
	} else {
		log.Fatal("Unknown scrape mode")
	}

}

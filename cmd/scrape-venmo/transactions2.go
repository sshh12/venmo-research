package main

import (
	"log"
	"time"

	"github.com/sshh12/venmo-research/storage"
	"github.com/sshh12/venmo-research/venmo"
)

func randomWorker2(client *venmo.Client, store *storage.Store, complete chan<- bool) {
	log.Printf("Random Worker2 Started\n")
	idsFound := make(map[int]bool)
	for {
		feed, err := client.FetchPublic(10000)
		if err != nil {
			log.Println("randomWorker2:", err)
			continue
		}
		overlap := 0
		for _, item := range feed {
			if v := idsFound[item.PaymentID]; v {
				overlap++
			}
			idsFound[item.PaymentID] = true
		}
		if overlap == len(feed) {
			store.Flush()
			log.Println("randomWorker2: all transactions already found...waiting")
			time.Sleep(500 * time.Second)
		} else {
			for _, item := range feed {
				if err := store.AddTransactions(&item); err != nil {
					log.Println(err)
				}
			}
		}
	}
}

// RunTransaction2Scraper scrapes transactions
func RunTransaction2Scraper(client *venmo.Client, store *storage.Store, workerCnt int) {
	if client == nil {
		log.Fatal("Token is required for scraping venmo")
		return
	}
	complete := make(chan bool)
	for j := 0; j < workerCnt; j++ {
		go randomWorker2(client, store, complete)
	}
	for j := 0; j < workerCnt; j++ {
		<-complete
	}
	store.Flush()
}

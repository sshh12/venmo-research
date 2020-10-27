package main

import (
	"log"
	"math/rand"

	"github.com/sshh12/venmo-research/storage"
	"github.com/sshh12/venmo-research/venmo"
)

type workerTask struct {
	Start  int
	End    int
	Shard  int
	Shards int
}

func randomWorker(client *venmo.Client, store *storage.Store, start int, end int, complete chan<- bool) {
	// run (end - start) number of times
	log.Printf("Random Worker Started -- [%d, %d)\n", start, end)
	for i := start; i < end; i++ {
		randID := rand.Intn(end-start) + start
		downloadFeedRange(client, store, randID, randID+1, 0, 1)
	}
	complete <- true
}

func worker(client *venmo.Client, store *storage.Store, tasks <-chan workerTask, complete chan<- bool) {
	for task := range tasks {
		log.Printf("Worker Started -- [%d, %d) -- shard(%d/%d)\n", task.Start, task.End, task.Shard, task.Shards)
		downloadFeedRange(client, store, task.Start, task.End, task.Shard, task.Shards)
		log.Printf("Worker Finished -- [%d, %d) -- shard(%d/%d)\n", task.Start, task.End, task.Shard, task.Shards)
	}
	complete <- true
}

func downloadFeedRange(client *venmo.Client, store *storage.Store, start int, end int, shard int, shards int) {
	for i := start; i < end; i++ {
		if i%shards != shard {
			continue
		}
		feed, err := client.FetchFeed(i)
		if err != nil {
			log.Fatal("failed @", i)
			panic(err)
		}
		for _, item := range feed {
			if err := store.AddTransactions(&item); err != nil {
				log.Println(err)
			}
		}
	}
}

// RunTransactionScraper scrapes transactions
func RunTransactionScraper(client *venmo.Client, store *storage.Store, randomMode bool, workerCnt int, startID int, endID int, interval int, shardIdx int, shardCnt int) {
	if client == nil {
		log.Fatal("Token is required for scraping venmo")
		return
	}
	if (endID-startID)%interval != 0 {
		log.Fatal("The range provided is not divisable by the interval")
		return
	}
	tasks := make(chan workerTask)
	complete := make(chan bool)
	if !randomMode {
		for j := 0; j < workerCnt; j++ {
			go worker(client, store, tasks, complete)
		}
		for i := startID; i < endID; i += interval {
			tasks <- workerTask{Start: i, End: i + interval, Shard: shardIdx, Shards: shardCnt}
		}
		close(tasks)
	} else {
		for j := 0; j < workerCnt; j++ {
			go randomWorker(client, store, startID, endID, complete)
		}
	}
	for j := 0; j < workerCnt; j++ {
		<-complete
	}
	store.Flush()
}

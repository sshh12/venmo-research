package main

import (
	"flag"
	"log"
	"math/rand"
	"os"

	"github.com/sshh12/venmo-research/storage"
	"github.com/sshh12/venmo-research/venmo"
)

type workerTask struct {
	Start  int
	End    int
	Shard  int
	Shards int
}

func main() {
	token := flag.String("token", "", "venmo token")
	shardIdx := flag.Int("shard_idx", 0, "shard index")
	shardCnt := flag.Int("shard_cnt", 1, "total shards")
	workerCnt := flag.Int("workers", 5, "parallel workers")
	startID := flag.Int("start_id", 0, "venmo id to start from")
	endID := flag.Int("end_id", 92000000, "venmo id to end at")
	interval := flag.Int("interval_size", 10000, "number of ids per file")
	randomMode := flag.Bool("random", false, "random mode, just fetch accounts at random (ignores interval and sharding)")
	flag.Parse()
	if *token == "" {
		*token = os.Getenv("VENMO_TOKEN")
		if *token == "" {
			log.Fatal("Token is required")
			return
		}
	}
	if (*endID-*startID)%*interval != 0 {
		log.Fatal("The range provided is not divisable by the interval")
		return
	}

	store, err := storage.NewPostgresStore()
	if err != nil {
		log.Fatal(err)
		return
	}

	client := venmo.NewClient(*token)
	tasks := make(chan workerTask)
	complete := make(chan bool)
	if !*randomMode {
		for j := 0; j < *workerCnt; j++ {
			go worker(client, store, tasks, complete)
		}
		for i := *startID; i < *endID; i += *interval {
			tasks <- workerTask{Start: i, End: i + *interval, Shard: *shardIdx, Shards: *shardCnt}
		}
		close(tasks)
	} else {
		for j := 0; j < *workerCnt; j++ {
			go randomWorker(client, store, *startID, *endID, complete)
		}
	}
	for j := 0; j < *workerCnt; j++ {
		<-complete
	}
	store.Flush()
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

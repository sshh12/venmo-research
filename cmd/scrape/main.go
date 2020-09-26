package main

import (
	"flag"
	"log"
	"os"

	"github.com/sshh12/venmo-research/storage"
	"github.com/sshh12/venmo-research/venmo"
)

type workerTask struct {
	Start  int
	End    int
	Path   string
	Shard  int
	Shards int
	Store  *storage.Store
}

func main() {

	var token string
	var savePath string
	flag.StringVar(&token, "token", "", "venmo token")
	shardIdx := flag.Int("shard_idx", 0, "shard index")
	shardCnt := flag.Int("shard_cnt", 1, "total shards")
	startID := flag.Int("start_id", 0, "venmo id to start from")
	endID := flag.Int("end_id", 90000000, "venmo id to start from")
	interval := flag.Int("interval_size", 10000, "number of ids per file")
	flag.Parse()
	os.Mkdir(savePath, 0755)
	if token == "" {
		log.Fatal("Token is required")
		return
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

	client := venmo.NewClient(token)
	workerCnt := 5
	tasks := make(chan workerTask)
	complete := make(chan bool)
	for j := 0; j < workerCnt; j++ {
		go worker(client, tasks, complete)
	}
	for i := *startID; i < *endID; i += *interval {
		tasks <- workerTask{Start: i, End: i + *interval, Path: savePath, Shard: *shardIdx, Shards: *shardCnt, Store: store}
	}
	close(tasks)
	for j := 0; j < workerCnt; j++ {
		<-complete
	}
	store.Flush()
}

func worker(client *venmo.Client, tasks <-chan workerTask, complete chan<- bool) {
	for task := range tasks {
		log.Printf("Worker Started -- [%d, %d) -- shard(%d/%d)\n", task.Start, task.End, task.Shard, task.Shards)
		downloadFeedRange(client, task.Store, task.Start, task.End, task.Path, task.Shard, task.Shards)
		log.Printf("Worker Finished -- [%d, %d) -- shard(%d/%d)\n", task.Start, task.End, task.Shard, task.Shards)
	}
	complete <- true
}

func downloadFeedRange(client *venmo.Client, store *storage.Store, start int, end int, savePath string, shard int, shards int) {
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
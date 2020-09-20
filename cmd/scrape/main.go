package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/sshh12/venmo-research/venmo"
)

type workerTask struct {
	Start  int
	End    int
	Path   string
	Shard  int
	Shards int
}

func main() {

	var token string
	var savePath string
	flag.StringVar(&token, "token", "", "venmo token")
	flag.StringVar(&savePath, "save_path", ".", "path to download to")
	shardIdx := flag.Int("shard_idx", 0, "shard index")
	shardCnt := flag.Int("shard_cnt", 1, "total shards")
	startID := flag.Int("start_id", 0, "venmo id to start from")
	endID := flag.Int("end_id", 90000000, "venmo id to start from")
	interval := flag.Int("interval_size", 10000, "number of ids per file")
	flag.Parse()
	if token == "" {
		panic("Token is required")
	}
	if (*endID-*startID)%*interval != 0 {
		panic("The range provided is not divisable by the interval")
	}

	client := venmo.NewClient(token)
	workerCnt := 5
	tasks := make(chan workerTask)
	complete := make(chan bool)
	for j := 0; j < workerCnt; j++ {
		go worker(client, tasks, complete)
	}
	for i := *startID; i < *endID; i += *interval {
		tasks <- workerTask{Start: i, End: i + *interval, Path: savePath, Shard: *shardIdx, Shards: *shardCnt}
	}
	close(tasks)
	for j := 0; j < workerCnt; j++ {
		<-complete
	}
}

func worker(client *venmo.Client, tasks <-chan workerTask, complete chan<- bool) {
	for task := range tasks {
		fmt.Printf("Worker Started -- [%d, %d) -- shard(%d/%d)\n", task.Start, task.End, task.Shard, task.Shards)
		downloadFeedRange(client, task.Start, task.End, task.Path, task.Shard, task.Shards)
		fmt.Printf("Worker Finished -- [%d, %d) -- shard(%d/%d)\n", task.Start, task.End, task.Shard, task.Shards)
	}
	complete <- true
}

func downloadFeedRange(client *venmo.Client, start int, end int, savePath string, shard int, shards int) {
	tempFn := path.Join(savePath, fmt.Sprintf("%d-%d.venmo-raw", start, end))
	fp, err := os.Create(tempFn)
	if err != nil {
		panic(err)
	}
	defer fp.Close()
	for i := start; i < end; i++ {
		if i%shards != shard {
			continue
		}
		feed, err := client.FetchFeed(i)
		if err != nil {
			fmt.Println("failed @", i)
			panic(err)
		}
		for _, item := range feed {
			encoded, _ := json.Marshal(item)
			if _, err := fp.WriteString(string(encoded) + "\n"); err != nil {
				panic(err)
			}
		}
	}
}

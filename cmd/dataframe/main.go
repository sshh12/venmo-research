package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/sshh12/venmo-research/venmo"
)

func main() {
	var savePath string
	flag.StringVar(&savePath, "save_path", ".", "path of raw venmo data")
	flag.Parse()
	files, err := filepath.Glob(path.Join(savePath, "*.venmo-raw"))
	if err != nil || len(files) == 0 {
		log.Fatal("No files found")
		return
	}
	users := make(map[int]bool)
	i := 0
	for _, fn := range files {
		file, err := os.Open(fn)
		if err != nil {
			log.Fatal(err)
			continue
		}
		reader := bufio.NewReader(file)
		var item venmo.FeedItem
		for {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				break
			}
			if err := json.Unmarshal([]byte(line), &item); err != nil {
				// log.Print(err)
				continue
			}
			id, _ := strconv.Atoi(item.Actor.ID)
			if !users[id] {
				fmt.Println(item.Actor.ID, item.Actor.Username)
				users[id] = true
			}

			i++
			if i%10000 == 0 {
				fmt.Println(i)
			}
		}
		file.Close()
	}
}

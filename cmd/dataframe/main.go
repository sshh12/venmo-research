package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"

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
	i := 0
	for _, fn := range files {
		fmt.Println(fn)
		file, err := os.Open(fn)
		if err != nil {
			log.Fatal(err)
			continue
		}
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)
		var item venmo.FeedItem
		for scanner.Scan() {
			line := scanner.Text()
			if err := json.Unmarshal([]byte(line), &item); err != nil {
				log.Print(line)
				continue
			}
			// fmt.Println(item.Message)
			// id, _ := strconv.Atoi(item.Actor.ID)

			i++
			if i%10000 == 0 {
				fmt.Println(i)
			}
		}
		file.Close()
	}
}

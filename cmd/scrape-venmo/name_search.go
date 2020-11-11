package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/sshh12/venmo-research/storage"
)

// RunNameSearchScraper scrapes geolocations
func RunNameSearchScraper(store *storage.Store) {
	done := make(chan error, 2)
	go runBing(store, done)
	go runDDG(store, done)
	log.Println(<-done)
	log.Println(<-done)
}

func runBing(store *storage.Store, done chan<- error) {
	for {
		users, err := store.SampleUsersWithoutBingResults(1000)
		if err != nil {
			done <- err
			return
		}
		for _, user := range users {
			log.Print(user.Name + " (Bing)")
			if err := bingLookup(&user); err != nil {
				log.Print(err)
			} else {
				store.UpdateUser(&user)
			}
		}
	}
}

func runDDG(store *storage.Store, done chan<- error) {
	for {
		users, err := store.SampleUsersWithoutDDGResults(1000)
		if err != nil {
			done <- err
			return
		}
		for _, user := range users {
			log.Print(user.Name + " (DuckDuckGo)")
			if err := ddgLookup(&user); err != nil {
				log.Print(err)
			} else {
				store.UpdateUser(&user)
			}
		}
	}
}

func bingLookup(user *storage.User) error {
	client := &http.Client{}
	url := fmt.Sprintf("https://www.bing.com/search?q=%s", strings.ReplaceAll(user.Name, " ", "+"))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.111 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	rg := regexp.MustCompile("<a href=\"([^\"]+?)\"[\\w=\",\\. ]+>([^<]+?)<\\/a><\\/h2>[\\s\\S]+?<p>([\\s\\S]+?)<\\/p>")
	matches := rg.FindAllStringSubmatch(string(body), -1)
	results := make([][]string, 0)
	for _, match := range matches {
		results = append(results, match[1:])
	}
	user.BingResults = map[string]interface{}{
		"results": results,
	}
	return nil
}

func ddgLookup(user *storage.User) error {
	client := &http.Client{}
	url := fmt.Sprintf("https://duckduckgo.com/d.js?q=%s", strings.ReplaceAll(user.Name, " ", "+"))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.111 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	splits := strings.Split(string(body), "DDG.Data.languages.resultLanguages")
	if len(splits) != 2 {
		return fmt.Errorf("Weird DDG resp")
	}
	user.DDGResults = strings.TrimSpace(splits[1])
	return nil
}

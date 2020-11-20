package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/sshh12/venmo-research/images"
	"github.com/sshh12/venmo-research/storage"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.111 Safari/537.36"

// RunNameSearchScraper scrapes geolocations
func RunNameSearchScraper(store *storage.Store, workers int) {
	done := make(chan error)
	for i := 0; i < workers; i++ {
		go runBing(store, done)
		go runDDG(store, done)
		go runPeekYou(store, done)
	}
	for err := range done {
		log.Println(err)
	}
}

func runBing(store *storage.Store, done chan<- error) {
	log.Printf("Bing worker started")
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
				if err := store.UpdateUser(&user); err != nil {
					log.Println(err)
				}
			}
		}
	}
}

func runDDG(store *storage.Store, done chan<- error) {
	log.Printf("DDG worker started")
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
				if err := store.UpdateUser(&user); err != nil {
					log.Println(err)
				}
			}
		}
	}
}

func runPeekYou(store *storage.Store, done chan<- error) {
	log.Printf("PeekYou worker started")
	for {
		users, err := store.SampleUsersWithoutPeekYouResults(1000)
		if err != nil {
			done <- err
			return
		}
		for _, user := range users {
			log.Print(user.Name + " (PeekYou)")
			if err := peekYouLookup(&user); err != nil {
				log.Print(err)
			} else {
				if err := store.UpdateUser(&user); err != nil {
					log.Println(err)
				}
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
	req.Header.Set("User-Agent", userAgent)
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
	if len(results) == 0 {
		return fmt.Errorf("bing results empty")
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
	req.Header.Set("User-Agent", userAgent)
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

func peekYouLookup(user *storage.User) error {
	client := &http.Client{}

	venmoPic, err := images.DownloadJPG(user.PictureURL)
	if err != nil {
		return err
	}

	indexURL := fmt.Sprintf("https://www.peekyou.com/%s", strings.ToLower(strings.ReplaceAll(user.Name, " ", "_")))
	indexReq, err := http.NewRequest("GET", indexURL, nil)
	if err != nil {
		return err
	}
	indexReq.Header.Set("User-Agent", userAgent)
	indexResp, err := client.Do(indexReq)
	if err != nil {
		return err
	}
	defer indexResp.Body.Close()
	indexBody, err := ioutil.ReadAll(indexResp.Body)
	if err != nil {
		return err
	}

	md5RE := regexp.MustCompile("MD5 = \"(\\w+)\"")
	md5Match := md5RE.FindStringSubmatch(string(indexBody))
	if len(md5Match) == 0 {
		return fmt.Errorf("peekyou md5 not found")
	}
	md5 := md5Match[1]

	serRE := regexp.MustCompile("serialized =\"([^\"]+)\"")
	serMatch := serRE.FindStringSubmatch(string(indexBody))
	if len(serMatch) == 0 {
		return fmt.Errorf("peekyou serialized not found")
	}
	ser := serMatch[1]

	webRE := regexp.MustCompile("web_results_search =\"([^\"]+)\"")
	webMatch := webRE.FindStringSubmatch(string(indexBody))
	if len(webMatch) == 0 {
		return fmt.Errorf("peekyou web_results_search not found")
	}
	web := webMatch[1]

	cachePayload := fmt.Sprintf("id=%s&serialized=%s&web_results_search=%s&sections=undefined", md5, ser, web)
	cacheReq, err := http.NewRequest("POST", "https://www.peekyou.com/web_results_new/check_live_results_cache.php", strings.NewReader(cachePayload))
	if err != nil {
		return err
	}
	cacheReq.Header.Set("User-Agent", userAgent)
	cacheReq.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	_, err = client.Do(cacheReq)
	if err != nil {
		return err
	}

	resultsPayload := fmt.Sprintf("id=%s&serialized=%s&web_results_search=%s", md5, ser, web)
	resultsReq, err := http.NewRequest("POST", "https://www.peekyou.com/web_results_new/web_results_checker.php", strings.NewReader(resultsPayload))
	if err != nil {
		return err
	}
	resultsReq.Header.Set("User-Agent", userAgent)
	resultsReq.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	resultsResp, err := client.Do(resultsReq)
	if err != nil {
		return err
	}
	defer resultsResp.Body.Close()

	var data peekYouData
	if err := json.NewDecoder(resultsResp.Body).Decode(&data); err != nil {
		return err
	}

	profilesMatch := make([][]string, 0)
	profilesUnkMatch := make([][]string, 0)

	for source, sourceData := range data.Results {
		if sourceData.Data == "" {
			continue
		}
		sourceHTML, err := base64.StdEncoding.DecodeString(sourceData.Data)
		if err != nil {
			log.Println("peekyou b64decode", err)
			continue
		}
		profileRE := regexp.MustCompile("href=\"([^\"]+?)\"[ targe=\"_blnkofwCicjvsp:T\\.P(\\'\\/u);\">?]+\\s+<img title=\"[\\w \\-\\.]+\"class=\"blur\" id=\"(\\w+)_src\"")
		profileMatches := profileRE.FindAllStringSubmatch(string(sourceHTML), -1)
		for _, profileMatch := range profileMatches {
			profileURL := profileMatch[1]
			picURL := fmt.Sprintf("https://pkimgcdn.peekyou.com/%s.jpeg", profileMatch[2])
			pic, err := images.DownloadJPG(picURL)
			if err == nil && images.IsSameImage(venmoPic, pic) {
				profilesMatch = append(profilesMatch, []string{source, profileURL})
				log.Printf("Match Found! %s %s", picURL, user.PictureURL)
			} else {
				profilesUnkMatch = append(profilesUnkMatch, []string{source, profileURL})
			}
		}
	}

	if len(profilesMatch) == 0 && len(profilesUnkMatch) == 0 {
		return fmt.Errorf("peekyou no results found")
	}

	user.PeekYouResults = map[string]interface{}{
		"ResultsMatch":    profilesMatch,
		"ResultsUnkMatch": profilesUnkMatch,
	}

	return nil
}

type peekYouData struct {
	Results map[string](struct {
		Data string `json:"data"`
	}) `json:"results"`
}

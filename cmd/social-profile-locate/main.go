package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/sshh12/venmo-research/storage"
)

var defaultPicURLs = []string{
	"https://s3.amazonaws.com/venmo/no-image.gif",
}

func main() {
	store, err := storage.NewPostgresStore()
	if err != nil {
		log.Fatal(err)
		return
	}
	users, _ := store.SampleUsers(10)
	for _, user := range users {
		processUser(&user)
	}
}

func processUser(user *storage.User) {
	for _, picURL := range defaultPicURLs {
		if picURL == user.PictureURL {
			return
		}
	}
	fmt.Println(user.Name, user.PictureURL)
	results, err := googleScrape(user.Name)
	fmt.Println(len(results), err)
	for _, result := range results {
		if strings.HasPrefix(result.ResultURL, "https://www.linkedin.com/in/") {
			fmt.Println(result.ResultURL)
		} else if strings.HasPrefix(result.ResultURL, "https://www.facebook.com/") {
			fmt.Println(result.ResultURL)
		} else if strings.HasPrefix(result.ResultURL, "https://twitter.com/") {
			fmt.Println(result.ResultURL)
		} else if strings.HasPrefix(result.ResultURL, "https://www.instagram.com/") {
			fmt.Println(result.ResultURL)
		} else {
			fmt.Println("else", result.ResultURL)
		}
	}
}

type GoogleResult struct {
	ResultURL   string
	ResultTitle string
	ResultDesc  string
}

func googleRequest(searchURL string) (*http.Response, error) {

	baseClient := &http.Client{}

	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4183.121 Safari/537.36")

	res, err := baseClient.Do(req)

	if err != nil {
		return nil, err
	} else {
		return res, nil
	}
}

func googleResultParser(response *http.Response) ([]GoogleResult, error) {
	doc, err := goquery.NewDocumentFromResponse(response)
	if err != nil {
		return nil, err
	}
	results := []GoogleResult{}
	sel := doc.Find("div.g")
	for i := range sel.Nodes {
		item := sel.Eq(i)
		linkTag := item.Find("a")
		link, _ := linkTag.Attr("href")
		titleTag := item.Find("h3.r")
		descTag := item.Find("span")
		desc := descTag.Text()
		title := titleTag.Text()
		link = strings.Trim(link, " ")
		if link != "" && link != "#" {
			result := GoogleResult{
				link,
				title,
				desc,
			}
			results = append(results, result)
		}
	}
	return results, err
}

func googleScrape(searchTerm string) ([]GoogleResult, error) {
	url := fmt.Sprintf("https://www.google.com/search?q=%s&num=100&hl=en", strings.Replace(searchTerm, " ", "+", -1))
	res, err := googleRequest(url)
	if err != nil {
		return nil, err
	}
	scrapes, err := googleResultParser(res)
	if err != nil {
		return nil, err
	} else {
		return scrapes, nil
	}
}

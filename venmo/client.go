package venmo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const defaultUserAgent string = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/35.0.1916.47 Safari/537.36"

// Client is an instance for webscraping venmo
type Client struct {
	httpClient *http.Client
	reqCnt     uint64
	token      string
	mux        sync.Mutex
}

type venmoFeed struct {
	Data   []venmoFeedItem `json:"data"`
	Paging venmoPaging     `json:"paging"`
}

type venmoFeedItem struct {
	PaymentID    int                `json:"payment_id"`
	StoryID      string             `json:"story_id"`
	Message      string             `json:"message"`
	Type         string             `json:"type"`
	Actor        venmoUser          `json:"actor"`
	Transactions []venmoTransaction `json:"transactions"`
	Created      string             `json:"created_time"`
	Updated      string             `json:"updated_time"`
}

type venmoPaging struct {
	NextURL string `json:"next"`
	PrevURL string `json:"previous"`
}

type venmoTransaction struct {
	Target interface{} `json:"target"`
}

type venmoUser struct {
	Username   string `json:"username"`
	PictureURL string `json:"picture"`
	Name       string `json:"name"`
	FirstName  string `json:"firstname"`
	LastName   string `json:"lastname"`
	Created    string `json:"date_created"`
	IsBusiness bool   `json:"is_business"`
	Canceled   bool   `json:"cancelled"`
	ID         string `json:"id"`
	ExternalID string `json:"external_id"`
}

// NewClient creates a venmo client
func NewClient(token string) *Client {
	client := &Client{token: token, httpClient: &http.Client{}}
	return client
}

// FetchFeed gets the feed of a user
func (client *Client) FetchFeed(userID int) ([]venmoFeedItem, error) {
	var body venmoFeed
	err := client.doRateLimitedRequest("GET", fmt.Sprintf("https://venmo.com/api/v5/users/%d/feed", userID), &body)
	if err != nil {
		return nil, err
	}
	items := body.Data
	curURL := body.Paging.PrevURL
	for curURL != "" {
		var pageBody venmoFeed
		err := client.doRateLimitedRequest("GET", curURL, &pageBody)
		if err != nil {
			return nil, err
		}
		if len(pageBody.Data) == 0 {
			break
		}
		items = append(items, pageBody.Data...)
		curURL = pageBody.Paging.PrevURL
	}
	curURL = body.Paging.NextURL
	for curURL != "" {
		var pageBody venmoFeed
		err := client.doRateLimitedRequest("GET", curURL, &pageBody)
		if err != nil {
			return nil, err
		}
		if len(pageBody.Data) == 0 {
			break
		}
		items = append(items, pageBody.Data...)
		curURL = pageBody.Paging.NextURL
	}
	return items, nil
}

func (client *Client) doRateLimitedRequest(method string, url string, respType interface{}) error {
	client.mux.Lock()
	client.mux.Unlock()
	err := client.doRequest(method, url, respType)
	if err != nil {
		client.mux.Lock()
		for err != nil {
			fmt.Print("Rate limited, waiting.", err, url)
			time.Sleep(5 * time.Minute)
			err = client.doRequest(method, url, respType)
		}
		client.mux.Unlock()
		fmt.Println("...done.")
	}
	return nil
}

func (client *Client) doRequest(method string, url string, respType interface{}) error {
	req, err := http.NewRequest(method, url, nil)
	req.Header.Add("User-Agent", defaultUserAgent)
	req.AddCookie(&http.Cookie{Name: "api_access_token", Value: client.token})
	if err != nil {
		return err
	}
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// temp, _ := ioutil.ReadAll(resp.Body)
	// fmt.Println(string(temp))
	if err := json.NewDecoder(resp.Body).Decode(&respType); err != nil {
		return err
	}
	return nil
}

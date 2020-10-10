package venmo

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

const defaultUserAgent string = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/35.0.1916.47 Safari/537.36"

// Client is an instance for webscraping venmo
type Client struct {
	Token      string
	httpClient *http.Client
	reqCnt     uint64
	mux        sync.Mutex
}

type venmoFeed struct {
	Data   []FeedItem  `json:"data"`
	Paging venmoPaging `json:"paging"`
}

// FeedItem is an item in a user's venmo feed
type FeedItem struct {
	PaymentID    int                `json:"payment_id"`
	StoryID      string             `json:"story_id"`
	Message      string             `json:"message"`
	Type         string             `json:"type"`
	Actor        User               `json:"actor"`
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

// User is a venmo user
type User struct {
	Username   string `json:"username"`
	PictureURL string `json:"picture"`
	Name       string `json:"name"`
	FirstName  string `json:"firstname"`
	LastName   string `json:"lastname"`
	Created    string `json:"date_created"`
	IsBusiness bool   `json:"is_business"`
	Cancelled  bool   `json:"cancelled"`
	ID         string `json:"id"`
	ExternalID string `json:"external_id"`
}

// CastTargetToUser attempts to convert interface to user
func CastTargetToUser(data interface{}) (*User, error) {
	user, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Could not cast to user")
	}
	userID, ok := user["id"].(string)
	if !ok {
		return nil, fmt.Errorf("Could not cast user ID field")
	}
	extID, ok := user["external_id"].(string)
	if !ok {
		return nil, fmt.Errorf("Could not cast ext ID field")
	}
	username, ok := user["username"].(string)
	if !ok {
		return nil, fmt.Errorf("Could not cast username field")
	}
	picURL, ok := user["picture"].(string)
	if !ok {
		return nil, fmt.Errorf("Could not cast username field")
	}
	name, ok := user["name"].(string)
	if !ok {
		return nil, fmt.Errorf("Could not cast name field")
	}
	firstname, ok := user["firstname"].(string)
	if !ok {
		return nil, fmt.Errorf("Could not cast firstname field")
	}
	lastname, ok := user["lastname"].(string)
	if !ok {
		return nil, fmt.Errorf("Could not cast lastname field")
	}
	created, ok := user["date_created"].(string)
	if !ok {
		created = ""
	}
	isBus, ok := user["is_business"].(bool)
	if !ok {
		isBus = false
	}
	cancelled, ok := user["cancelled"].(bool)
	if !ok {
		return nil, fmt.Errorf("Could not cast cancelled field")
	}
	return &User{
		ID:         userID,
		ExternalID: extID,
		Username:   username,
		PictureURL: picURL,
		Name:       name,
		FirstName:  firstname,
		LastName:   lastname,
		Created:    created,
		IsBusiness: isBus,
		Cancelled:  cancelled,
	}, nil
}

// NewClient creates a venmo client
func NewClient(token string) *Client {
	client := &Client{Token: token, httpClient: &http.Client{}}
	return client
}

// NewClientFromLogin creates a venmo client from a username (or email/phone) and password
func NewClientFromLogin(user string, password string) *Client {
	// TODO
	// https://github.com/mmohades/Venmo/blob/9fdf5fdd106d35906729ffd69f424d3540bc81d2/venmo_api/apis/auth_api.py
	return nil
}

// FetchFeed gets the feed of a user
func (client *Client) FetchFeed(userID int) ([]FeedItem, error) {
	var body venmoFeed
	err := client.doRateLimitedRequest("GET", fmt.Sprintf("https://venmo.com/api/v5/users/%d/feed", userID), &body)
	if err != nil {
		return nil, err
	}
	items := body.Data
	if len(body.Data) != 0 {
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
			log.Println("Rate limited, waiting.", err, url)
			time.Sleep(5 * time.Minute)
			err = client.doRequest(method, url, respType)
		}
		client.mux.Unlock()
		log.Println("...done.")
	}
	return nil
}

func (client *Client) doRequest(method string, url string, respType interface{}) error {
	req, err := http.NewRequest(method, url, nil)
	req.Header.Add("User-Agent", defaultUserAgent)
	req.AddCookie(&http.Cookie{Name: "api_access_token", Value: client.Token})
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

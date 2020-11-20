package main

import (
	"fmt"
	"log"
	"time"

	"github.com/sshh12/venmo-research/facebook"
	"github.com/sshh12/venmo-research/storage"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

// RunFacebookScraper scrapes facebook
func RunFacebookScraper(store *storage.Store, workerCnt int, selPath string, selDriver string, selPort int, selHeadless bool, selXvfb bool, fbUser string, fbPass string) {

	opts := []selenium.ServiceOption{
		selenium.ChromeDriver(selDriver),
		selenium.Output(nil),
	}
	if selXvfb {
		opts = append(opts, selenium.StartFrameBuffer())
	}
	service, err := selenium.NewSeleniumService(selPath, selPort, opts...)
	if err != nil {
		log.Print(err)
		return
	}
	defer service.Stop()

	done := make(chan error)
	for i := 0; i < workerCnt; i++ {
		go scrapeFacebook(store, selPort, selHeadless, fbUser, fbPass, done)
	}
	for i := 0; i < workerCnt; i++ {
		fmt.Print(<-done)
	}
}

func scrapeFacebook(store *storage.Store, selPort int, selHeadless bool, fbUser string, fbPass string, done chan<- error) {
	log.Printf("Facebook worker started (%s)", fbUser)
	caps := selenium.Capabilities{"browserName": "chrome"}
	if selHeadless {
		chrCaps := chrome.Capabilities{
			Args: []string{
				"--no-sandbox",
				"--headless",
			},
			W3C: true,
		}
		caps.AddChrome(chrCaps)
	}
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", selPort))
	if err != nil {
		done <- err
		return
	}
	defer wd.Quit()

	if err := facebook.LoginToFacebook(wd, fbUser, fbPass); err != nil {
		done <- err
		return
	}

	for {
		users, err := store.SampleUsersWithoutFacebookResults(500)
		if err != nil {
			done <- err
			return
		}
		if len(users) == 0 {
			time.Sleep(300 * time.Second)
			continue
		}
		for _, user := range users {
			if err := findOnFacebook(&user, wd); err != nil {
				log.Print(err)
			} else {
				store.UpdateUser(&user)
			}
		}
	}
}

func findOnFacebook(user *storage.User, wd selenium.WebDriver) error {
	results, err := facebook.SearchPeople(wd, user.Name)
	if err != nil {
		return err
	}
	data := make([]map[string]string, 0)
	if len(results) == 0 {
		return fmt.Errorf("no results found")
	}
	for _, result := range results {
		log.Print(result.URL, " ", result.Name)
		data = append(data, map[string]string{
			"url":  result.URL,
			"name": result.Name,
		})
	}
	user.FacebookResults = map[string]interface{}{
		"results": data,
	}
	return nil
}

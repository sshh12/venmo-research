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

// RunPeekYouLocScraper scrapes facebook
func RunPeekYouLocScraper(store *storage.Store, workerCnt int, selPath string, selDriver string, selPort int, selHeadless bool, selXvfb bool, fbUser string, fbPass string) {

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
		go scrapePeekYouFacebook(store, selPort, selHeadless, fbUser, fbPass, done)
	}
	for i := 0; i < workerCnt; i++ {
		fmt.Print(<-done)
	}
}

func scrapePeekYouFacebook(store *storage.Store, selPort int, selHeadless bool, fbUser string, fbPass string, done chan<- error) {
	log.Printf("PeekYou Facebook worker started (%s)", fbUser)
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
		users, err := store.SampleUsersWithPeekYouMatchWithoutProfile(200)
		if err != nil {
			done <- err
			return
		}
		if len(users) == 0 {
			time.Sleep(300 * time.Second)
			continue
		}
		for _, user := range users {
			facebookURL := ""
			for _, result := range user.PeekYouResults["ResultsMatch"].([]interface{}) {
				// Only consider first facebook profile found
				profilePair := result.([]interface{})
				if profilePair[0].(string) == "facebook" {
					facebookURL = profilePair[1].(string)
					break
				}
			}
			if facebookURL == "" {
				continue
			}
			profile, err := facebook.ExtractProfileData(wd, facebookURL)
			if err != nil {
				log.Println("facebook: ", facebookURL, err)
				continue
			}
			if len(profile.InfoList) == 0 {
				log.Println("facebook: no profile data found")
				continue
			}
			log.Println(user.Name, user.Username, profile.InfoList)
			user.FacebookProfile = map[string]interface{}{
				"name": profile.Name,
				"url":  profile.URL,
				"info": profile.InfoList,
			}
			if err := store.UpdateUser(&user); err != nil {
				log.Println(err)
			}
		}
	}
}

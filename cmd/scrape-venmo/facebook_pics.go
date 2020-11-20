package main

import (
	"fmt"
	"log"
	"time"

	"github.com/sshh12/venmo-research/facebook"
	"github.com/sshh12/venmo-research/images"
	"github.com/sshh12/venmo-research/storage"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

// RunFacebookPicsScraper scrapes facebook
func RunFacebookPicsScraper(store *storage.Store, workerCnt int, selPath string, selDriver string, selPort int, selHeadless bool, selXvfb bool, fbUser string, fbPass string) {

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
		go scrapeFacebookPics(store, selPort, selHeadless, fbUser, fbPass, done)
	}
	for i := 0; i < workerCnt; i++ {
		fmt.Print(<-done)
	}
}

func scrapeFacebookPics(store *storage.Store, selPort int, selHeadless bool, fbUser string, fbPass string, done chan<- error) {
	log.Printf("Facebook pictures worker started (%s)", fbUser)
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

	usersProcessed := 0
	usersFound := 0

	for {
		users, err := store.SampleUsersWithFacebookResultsWithoutProfile(500)
		if err != nil {
			done <- err
			return
		}
		if len(users) == 0 {
			time.Sleep(300 * time.Second)
			continue
		}
		for _, user := range users {
			log.Printf("Worker Stats -- Found %d of %d", usersFound, usersProcessed)
			usersProcessed++
			venmoPic, err := images.DownloadJPG(user.PictureURL)
			if err != nil {
				log.Println(venmoPic, err)
				continue
			}
			var match *facebook.ProfileResult
			for _, p := range user.FacebookResults["results"].([]interface{}) {
				url := p.(map[string]interface{})["url"].(string)
				profile, err := facebook.ExtractProfileData(wd, url)
				if err != nil {
					log.Println(url, err)
					continue
				}
				if profile.Pic != nil && images.IsSameImage(profile.Pic, venmoPic) {
					log.Printf("Match Found! %s %s", profile.URL, user.PictureURL)
					match = profile
					break
				}
			}
			if match != nil {
				usersFound++
				user.FacebookProfile = map[string]interface{}{
					"name": match.Name,
					"url":  match.URL,
					"info": match.InfoList,
				}
				if err := store.UpdateUser(&user); err != nil {
					log.Println(err)
				}
			}
		}
	}
}

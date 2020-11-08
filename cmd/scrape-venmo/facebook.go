package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/sshh12/venmo-research/storage"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

// RunFacebookScraper scrapes facebook
func RunFacebookScraper(store *storage.Store, workerCnt int, selPath string, selDriver string, selPort int, selHeadless bool, fbUser string, fbPass string) {

	opts := []selenium.ServiceOption{
		// selenium.StartFrameBuffer(),
		selenium.ChromeDriver(selDriver),
		selenium.Output(nil),
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

	if err := wd.Get("https://www.facebook.com/"); err != nil {
		done <- err
		return
	}
	time.Sleep(1 * time.Second)
	userInput, err := wd.FindElement(selenium.ByXPATH, "/html/body/div[1]/div[2]/div[1]/div/div/div/div[2]/div/div[1]/form/div[1]/div[1]/input")
	if err != nil {
		done <- err
		return
	}
	passwordInput, err := wd.FindElement(selenium.ByXPATH, "/html/body/div[1]/div[2]/div[1]/div/div/div/div[2]/div/div[1]/form/div[1]/div[2]/input")
	if err != nil {
		done <- err
		return
	}
	loginBtn, err := wd.FindElement(selenium.ByXPATH, "/html/body/div[1]/div[2]/div[1]/div/div/div/div[2]/div/div[1]/form/div[2]/button")
	if err != nil {
		done <- err
		return
	}

	userInput.Click()
	userInput.SendKeys(fbUser)
	passwordInput.Click()
	passwordInput.SendKeys(fbPass)
	loginBtn.Click()
	time.Sleep(3 * time.Second)

	for {
		users, err := store.SampleUsersWithoutFacebookResults(1000)
		if err != nil {
			done <- err
			return
		}
		for _, user := range users {
			if err := searchFacebook(&user, wd); err != nil {
				log.Print(err)
			} else {
				store.UpdateUser(&user)
			}
		}
	}
}

func searchFacebook(user *storage.User, wd selenium.WebDriver) error {
	searchURL := fmt.Sprintf("https://www.facebook.com/search/people/?q=%s", strings.ReplaceAll(user.Name, " ", "%20"))
	if err := wd.Get(searchURL); err != nil {
		return err
	}
	time.Sleep(3 * time.Second)
	source, err := wd.PageSource()
	if err != nil {
		return err
	}
	rg := regexp.MustCompile("href=\"(https:\\/\\/www.facebook.com\\/[^\"]+?)\" role=\"link\" tabindex=\"0\"><span>([ \\w]+?)<\\/span")
	matches := rg.FindAllStringSubmatch(source, -1)
	results := make([]map[string]string, 0)
	for _, match := range matches {
		profileURL := strings.ReplaceAll(strings.ReplaceAll(match[1], "&amp;ref=br_rs", ""), "?ref=br_rs", "")
		name := match[2]
		results = append(results, map[string]string{
			"url":  profileURL,
			"name": name,
		})
		log.Print(profileURL, " ", name)
	}
	if len(results) == 0 {
		return fmt.Errorf("no results found")
	}
	user.FacebookResults = map[string]interface{}{
		"results": results,
	}
	return nil
}

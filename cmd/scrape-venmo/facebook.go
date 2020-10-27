package main

import (
	"fmt"
	"log"
	"time"

	"github.com/sshh12/venmo-research/storage"
	"github.com/tebeka/selenium"
)

// RunFacebookScraper scrapes facebook
func RunFacebookScraper(store *storage.Store, selPath string, selDriver string, selPort int, fbUser string, fbPass string) {

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
	caps := selenium.Capabilities{"browserName": "chrome"}
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", selPort))
	if err != nil {
		log.Print(err)
		return
	}
	defer wd.Quit()

	if err := wd.Get("https://www.facebook.com/"); err != nil {
		log.Print(err)
		return
	}
	userInput, err := wd.FindElement(selenium.ByXPATH, "/html/body/div[1]/div[2]/div[1]/div/div/div/div[2]/div/div[1]/form/div[1]/div[1]/input")
	if err != nil {
		log.Print(err)
		return
	}
	passwordInput, err := wd.FindElement(selenium.ByXPATH, "/html/body/div[1]/div[2]/div[1]/div/div/div/div[2]/div/div[1]/form/div[1]/div[2]/input")
	if err != nil {
		log.Print(err)
		return
	}
	loginBtn, err := wd.FindElement(selenium.ByXPATH, "/html/body/div[1]/div[2]/div[1]/div/div/div/div[2]/div/div[1]/form/div[2]/button")
	if err != nil {
		log.Print(err)
		return
	}

	userInput.Click()
	userInput.SendKeys(fbUser)
	passwordInput.Click()
	passwordInput.SendKeys(fbPass)
	loginBtn.Click()

	time.Sleep(10 * time.Second)
}

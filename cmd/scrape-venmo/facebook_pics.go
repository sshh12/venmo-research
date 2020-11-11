package main

import (
	"fmt"
	"image"
	draw "image/draw"
	jpeg "image/jpeg"
	"log"
	"net/http"
	"strings"
	"time"

	"regexp"

	"github.com/sshh12/venmo-research/storage"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	images "github.com/vitali-fedulov/images"
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

	if err := loginToFacebook(wd, fbUser, fbPass); err != nil {
		done <- err
		return
	}

	usersProcessed := 0
	usersFound := 0

	for {
		users, err := store.SampleUsersWithFacebookResults(1000)
		if err != nil {
			done <- err
			return
		}
		for _, user := range users {
			log.Printf("Worker Stats -- Found %d of %d", usersFound, usersProcessed)
			usersProcessed++
			venmoPic, err := downloadJPG(user.PictureURL)
			if err != nil {
				log.Println(venmoPic, err)
				continue
			}
			var match *facebookProfile
			for _, p := range user.FacebookResults["results"].([]interface{}) {
				url := p.(map[string]interface{})["url"].(string)
				profile, err := extractFacebookProfileData(wd, url)
				if err != nil {
					log.Println(url, err)
					continue
				}
				if profile.Pic != nil && imgSim(profile.Pic, venmoPic) {
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

func imgSim(imgA *image.RGBA, imgB *image.RGBA) bool {
	hashA, imgSizeA := images.Hash(imgA)
	hashB, imgSizeB := images.Hash(imgB)
	return images.Similar(hashA, hashB, imgSizeA, imgSizeB)
}

func downloadJPG(url string) (*image.RGBA, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	img, err := jpeg.Decode(resp.Body)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	m := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(m, m.Bounds(), img, b.Min, draw.Src)
	return m, nil
}

type facebookProfile struct {
	URL      string
	PicURL   string
	Name     string
	Pic      *image.RGBA
	InfoList []string
}

func extractFacebookProfileData(wd selenium.WebDriver, profileURL string) (*facebookProfile, error) {
	if err := wd.Get(profileURL); err != nil {
		return nil, err
	}
	time.Sleep(2 * time.Second)
	source, err := wd.PageSource()
	if err != nil {
		return nil, err
	}

	rgPic := regexp.MustCompile("width=\"100%\" xlink:href=\"([^\"]+?)\"")
	matchPic := rgPic.FindStringSubmatch(source)
	if len(matchPic) == 0 {
		return nil, fmt.Errorf("profile pic not found")
	}
	picURL := strings.ReplaceAll(matchPic[1], "&amp;", "&")
	pic, err := downloadJPG(picURL)
	if err != nil {
		log.Println("fetch profile pic", err)
	}

	rgName := regexp.MustCompile(">([^<]+?)<\\/h1>")
	matchName := rgName.FindStringSubmatch(source)
	if len(matchName) == 0 {
		return nil, fmt.Errorf("name not found")
	}
	name := matchName[1]

	foundInfo := make([]string, 0)

	rgInfoLinked := regexp.MustCompile("dir=\"auto\">([\\w,\\.'\"/ ]+?) <a[ \\w=\":/\\.\\-]+?><div class=\"\\w+\"><span>([^\"]+?)<\\/span")
	matchesInfoLinked := rgInfoLinked.FindAllStringSubmatch(source, -1)
	for _, m := range matchesInfoLinked {
		foundInfo = append(foundInfo, m[1]+" "+m[2])
	}

	rgInfo := regexp.MustCompile("dir=\"auto\">([\\w,\\.'\"/ ]+?)<\\/span>")
	matchesInfo := rgInfo.FindAllStringSubmatch(source, -1)
	for _, m := range matchesInfo {
		foundInfo = append(foundInfo, m[1])
	}

	infoList := make([]string, 0)
	for _, item := range foundInfo {
		if strings.HasPrefix(item, "Lives ") || strings.HasPrefix(item, "From ") {
			infoList = append(infoList, item)
		}
	}

	profile := &facebookProfile{
		URL:      profileURL,
		PicURL:   picURL,
		Pic:      pic,
		Name:     name,
		InfoList: infoList,
	}
	return profile, nil
}

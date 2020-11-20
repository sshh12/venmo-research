package facebook

import (
	"fmt"
	"image"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/sshh12/venmo-research/images"
	"github.com/tebeka/selenium"
)

// SearchResult result of facebook search
type SearchResult struct {
	URL  string
	Name string
}

// ProfileResult facebook profile metadata
type ProfileResult struct {
	URL      string
	PicURL   string
	Name     string
	Pic      *image.RGBA
	InfoList []string
}

// SearchPeople search for people
func SearchPeople(wd selenium.WebDriver, name string) ([]SearchResult, error) {
	searchURL := fmt.Sprintf("https://www.facebook.com/search/people/?q=%s", strings.ReplaceAll(name, " ", "%20"))
	if err := wd.Get(searchURL); err != nil {
		return nil, err
	}
	time.Sleep(3 * time.Second)
	source, err := wd.PageSource()
	if err != nil {
		return nil, err
	}
	rg := regexp.MustCompile("href=\"(https:\\/\\/www.facebook.com\\/[^\"]+?)\" role=\"link\" tabindex=\"0\"><span>([ \\w]+?)<\\/span")
	matches := rg.FindAllStringSubmatch(source, -1)
	results := make([]SearchResult, 0)
	for _, match := range matches {
		profileURL := strings.ReplaceAll(strings.ReplaceAll(match[1], "&amp;ref=br_rs", ""), "?ref=br_rs", "")
		name := match[2]
		results = append(results, SearchResult{
			URL:  profileURL,
			Name: name,
		})
	}
	return results, nil
}

// LoginToFacebook login to facebook
func LoginToFacebook(wd selenium.WebDriver, user string, password string) error {
	if err := wd.Get("https://www.facebook.com/"); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	userInput, err := wd.FindElement(selenium.ByXPATH, "/html/body/div[1]/div[2]/div[1]/div/div/div/div[2]/div/div[1]/form/div[1]/div[1]/input")
	if err != nil {
		return err
	}
	passwordInput, err := wd.FindElement(selenium.ByXPATH, "/html/body/div[1]/div[2]/div[1]/div/div/div/div[2]/div/div[1]/form/div[1]/div[2]/input")
	if err != nil {
		return err
	}
	loginBtn, err := wd.FindElement(selenium.ByXPATH, "/html/body/div[1]/div[2]/div[1]/div/div/div/div[2]/div/div[1]/form/div[2]/button")
	if err != nil {
		return err
	}
	userInput.Click()
	userInput.SendKeys(user)
	passwordInput.Click()
	passwordInput.SendKeys(password)
	loginBtn.Click()
	time.Sleep(3 * time.Second)
	return nil
}

// ExtractProfileData visits a profile and downloads their data
func ExtractProfileData(wd selenium.WebDriver, profileURL string) (*ProfileResult, error) {
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
	pic, err := images.DownloadJPG(picURL)
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

	profile := &ProfileResult{
		URL:      profileURL,
		PicURL:   picURL,
		Pic:      pic,
		Name:     name,
		InfoList: infoList,
	}
	return profile, nil
}

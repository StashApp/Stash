package scraper

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

// Timeout to get the image. Includes transfer time. May want to make this
// configurable at some point.
const imageGetTimeout = time.Second * 30

func setPerformerImage(p *models.ScrapedPerformer, globalConfig GlobalConfig) error {
	if p == nil || p.Image == nil || !strings.HasPrefix(*p.Image, "http") {
		// nothing to do
		return nil
	}

	img, err := getImage(*p.Image, globalConfig)
	if err != nil {
		return err
	}

	p.Image = img

	return nil
}

func setSceneImage(s *models.ScrapedScene, globalConfig GlobalConfig) error {
	// don't try to get the image if it doesn't appear to be a URL
	if s == nil || s.Image == nil || !strings.HasPrefix(*s.Image, "http") {
		// nothing to do
		return nil
	}

	img, err := getImage(*s.Image, globalConfig)
	if err != nil {
		return err
	}

	s.Image = img

	return nil
}

func getImage(url string, globalConfig GlobalConfig) (*string, error) {
	client := &http.Client{
		Timeout: imageGetTimeout,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	userAgent := globalConfig.UserAgent
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	// assume is a URL for now
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// determine the image type and set the base64 type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}

	img := "data:" + contentType + ";base64," + utils.GetBase64StringFromData(body)
	return &img, nil
}

func getStashPerformerImage(stashURL string, performerID string, globalConfig GlobalConfig) (*string, error) {
	return getImage(stashURL+"/performer/"+performerID+"/image", globalConfig)
}

func getStashSceneImage(stashURL string, sceneID string, globalConfig GlobalConfig) (*string, error) {
	return getImage(stashURL+"/scene/"+sceneID+"/screenshot", globalConfig)
}

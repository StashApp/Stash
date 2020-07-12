package scraper

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/stashapp/stash/pkg/models"
)

type config struct {
	ID   string
	path string

	// The name of the scraper. This is displayed in the UI.
	Name string `yaml:"name"`

	// Configuration for querying performers by name
	PerformerByName *scraperTypeConfig `yaml:"performerByName"`

	// Configuration for querying performers by a Performer fragment
	PerformerByFragment *scraperTypeConfig `yaml:"performerByFragment"`

	// Configuration for querying a performer by a URL
	PerformerByURL []*scrapeByURLConfig `yaml:"performerByURL"`

	// Configuration for querying scenes by a Scene fragment
	SceneByFragment *scraperTypeConfig `yaml:"sceneByFragment"`

	// Configuration for querying a scene by a URL
	SceneByURL []*scrapeByURLConfig `yaml:"sceneByURL"`

	// Scraper debugging options
	DebugOptions *scraperDebugOptions `yaml:"debug"`

	// Stash server configuration
	StashServer *stashServer `yaml:"stashServer"`

	// Xpath scraping configurations
	XPathScrapers xPathScrapers `yaml:"xPathScrapers"`
}

type stashServer struct {
	URL string `yaml:"url"`
}

type scraperTypeConfig struct {
	Action  scraperAction `yaml:"action"`
	Script  []string      `yaml:"script,flow"`
	Scraper string        `yaml:"scraper"`

	// for xpath name scraper only
	QueryURL string `yaml:"queryURL"`
}

type scrapeByURLConfig struct {
	scraperTypeConfig `yaml:",inline"`
	URL               []string `yaml:"url,flow"`
}

func (c scrapeByURLConfig) matchesURL(url string) bool {
	for _, thisURL := range c.URL {
		if strings.Contains(url, thisURL) {
			return true
		}
	}

	return false
}

type scraperDebugOptions struct {
	PrintHTML bool `yaml:"printHTML"`
}

func loadScraperFromYAML(id string, reader io.Reader) (*config, error) {
	ret := &config{}

	parser := yaml.NewDecoder(reader)
	parser.SetStrict(true)
	err := parser.Decode(&ret)
	if err != nil {
		return nil, err
	}

	ret.ID = id

	return ret, nil
}

func loadScraperFromYAMLFile(path string) (*config, error) {
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		return nil, err
	}

	// set id to the filename
	id := filepath.Base(path)
	id = id[:strings.LastIndex(id, ".")]

	ret, err := loadScraperFromYAML(id, file)
	if err != nil {
		return nil, err
	}

	ret.path = path

	return ret, nil
}

func (c config) toScraper() *models.Scraper {
	ret := models.Scraper{
		ID:   c.ID,
		Name: c.Name,
	}

	performer := models.ScraperSpec{}
	if c.PerformerByName != nil {
		performer.SupportedScrapes = append(performer.SupportedScrapes, models.ScrapeTypeName)
	}
	if c.PerformerByFragment != nil {
		performer.SupportedScrapes = append(performer.SupportedScrapes, models.ScrapeTypeFragment)
	}
	if len(c.PerformerByURL) > 0 {
		performer.SupportedScrapes = append(performer.SupportedScrapes, models.ScrapeTypeURL)
		for _, v := range c.PerformerByURL {
			performer.Urls = append(performer.Urls, v.URL...)
		}
	}

	if len(performer.SupportedScrapes) > 0 {
		ret.Performer = &performer
	}

	scene := models.ScraperSpec{}
	if c.SceneByFragment != nil {
		scene.SupportedScrapes = append(scene.SupportedScrapes, models.ScrapeTypeFragment)
	}
	if len(c.SceneByURL) > 0 {
		scene.SupportedScrapes = append(scene.SupportedScrapes, models.ScrapeTypeURL)
		for _, v := range c.SceneByURL {
			scene.Urls = append(scene.Urls, v.URL...)
		}
	}

	if len(scene.SupportedScrapes) > 0 {
		ret.Scene = &scene
	}

	return &ret
}

func (c config) supportsPerformers() bool {
	return c.PerformerByName != nil || c.PerformerByFragment != nil || len(c.PerformerByURL) > 0
}

func (c config) matchesPerformerURL(url string) bool {
	for _, scraper := range c.PerformerByURL {
		if scraper.matchesURL(url) {
			return true
		}
	}

	return false
}

func (c config) ScrapePerformerNames(name string, globalConfig GlobalConfig) ([]*models.ScrapedPerformer, error) {
	if c.PerformerByName != nil {
		s := getScraper(*c.PerformerByName, c, globalConfig)
		return s.scrapePerformersByName(name)
	}

	return nil, nil
}

func (c config) ScrapePerformer(scrapedPerformer models.ScrapedPerformerInput, globalConfig GlobalConfig) (*models.ScrapedPerformer, error) {
	if c.PerformerByFragment != nil {
		s := getScraper(*c.PerformerByFragment, c, globalConfig)
		return s.scrapePerformerByFragment(scrapedPerformer)
	}

	// try to match against URL if present
	if scrapedPerformer.URL != nil && *scrapedPerformer.URL != "" {
		return c.ScrapePerformerURL(*scrapedPerformer.URL, globalConfig)
	}

	return nil, nil
}

func (c config) ScrapePerformerURL(url string, globalConfig GlobalConfig) (*models.ScrapedPerformer, error) {
	for _, scraper := range c.PerformerByURL {
		if scraper.matchesURL(url) {
			s := getScraper(scraper.scraperTypeConfig, c, globalConfig)
			ret, err := s.scrapePerformerByURL(url)
			if err != nil {
				return nil, err
			}

			if ret != nil {
				return ret, nil
			}
		}
	}

	return nil, nil
}

func (c config) supportsScenes() bool {
	return c.SceneByFragment != nil || len(c.SceneByURL) > 0
}

func (c config) matchesSceneURL(url string) bool {
	for _, scraper := range c.SceneByURL {
		if scraper.matchesURL(url) {
			return true
		}
	}

	return false
}

func (c config) ScrapeScene(scene models.SceneUpdateInput, globalConfig GlobalConfig) (*models.ScrapedScene, error) {
	if c.SceneByFragment != nil {
		s := getScraper(*c.SceneByFragment, c, globalConfig)
		return s.scrapeSceneByFragment(scene)
	}

	return nil, nil
}

func (c config) ScrapeSceneURL(url string, globalConfig GlobalConfig) (*models.ScrapedScene, error) {
	for _, scraper := range c.SceneByURL {
		if scraper.matchesURL(url) {
			s := getScraper(scraper.scraperTypeConfig, c, globalConfig)
			ret, err := s.scrapeSceneByURL(url)
			if err != nil {
				return nil, err
			}

			if ret != nil {
				return ret, nil
			}
		}
	}

	return nil, nil
}

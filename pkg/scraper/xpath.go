package scraper

import (
	"errors"
	"net/url"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
)

type commonXPathConfig map[string]string

func (c commonXPathConfig) applyCommon(src string) string {
	ret := src
	for commonKey, commonVal := range c {
		if strings.Contains(ret, commonKey) {
			ret = strings.Replace(ret, commonKey, commonVal, -1)
		}
	}

	return ret
}

type xpathScraperConfig map[string]interface{}

func createXPathScraperConfig(src map[interface{}]interface{}) xpathScraperConfig {
	ret := make(xpathScraperConfig)

	if src != nil {
		for k, v := range src {
			keyStr, isStr := k.(string)
			if isStr {
				ret[keyStr] = v
			}
		}
	}

	return ret
}

type xpathRegexConfig map[interface{}]interface{}
type xpathRegexConfigs []xpathRegexConfig

func (c xpathRegexConfig) apply(value string) string {
	regex := ""
	with := ""

	if regexI, _ := c["regex"]; regexI != nil {
		regex, _ = regexI.(string)
	}
	if withI, _ := c["with"]; withI != nil {
		with, _ = withI.(string)
	}

	if regex != "" {
		re, err := regexp.Compile(regex)
		if err != nil {
			logger.Warnf("Error compiling regex '%s': %s", regex, err.Error())
			return value
		}

		return re.ReplaceAllString(value, with)
	}

	return value
}

func (c xpathRegexConfigs) apply(value string) string {
	// apply regex in order
	for _, config := range c {
		value = config.apply(value)
	}

	// remove whitespace again
	value = commonPostProcess(value)

	return value
}

type xpathScraperAttrConfig map[interface{}]interface{}

func (c xpathScraperAttrConfig) getString(key string) string {
	ret, _ := c[key]

	if ret == nil {
		return ""
	}

	asStr, _ := ret.(string)
	return asStr
}

func (c xpathScraperAttrConfig) getSelector() string {
	const selectorKey = "selector"
	return c.getString(selectorKey)
}

func (c xpathScraperAttrConfig) getConcat() string {
	const concatKey = "concat"
	return c.getString(concatKey)
}

func (c xpathScraperAttrConfig) hasConcat() bool {
	return c.getConcat() != ""
}

func (c xpathScraperAttrConfig) getParseDate() string {
	const parseDateKey = "parseDate"
	return c.getString(parseDateKey)
}

func (c xpathScraperAttrConfig) getReplace() xpathRegexConfigs {
	const replaceKey = "replace"
	val, _ := c[replaceKey]

	var ret xpathRegexConfigs
	if val == nil {
		return ret
	}

	asSlice, _ := val.([]interface{})

	for _, v := range asSlice {
		asMap, _ := v.(map[interface{}]interface{})
		ret = append(ret, xpathRegexConfig(asMap))
	}

	return ret
}

func (c xpathScraperAttrConfig) concatenateResults(nodes []*html.Node) string {
	separator := c.getConcat()
	result := []string{}

	for _, elem := range nodes {
		text := htmlquery.InnerText(elem)
		text = commonPostProcess(text)

		result = append(result, text)
	}

	return strings.Join(result, separator)
}

func (c xpathScraperAttrConfig) parseDate(value string) string {
	parseDate := c.getParseDate()

	if parseDate == "" {
		return value
	}

	// try to parse the date using the pattern
	// if it fails, then just fall back to the original value
	parsedValue, err := time.Parse(parseDate, value)
	if err != nil {
		logger.Warnf("Error parsing date string '%s' using format '%s': %s", value, parseDate, err.Error())
		return value
	}

	// convert it into our date format
	const internalDateFormat = "2006-01-02"
	return parsedValue.Format(internalDateFormat)
}

func (c xpathScraperAttrConfig) replaceRegex(value string) string {
	replace := c.getReplace()
	return replace.apply(value)
}

func (c xpathScraperAttrConfig) postProcess(value string) string {
	// perform regex replacements first
	value = c.replaceRegex(value)
	value = c.parseDate(value)

	return value
}

func commonPostProcess(value string) string {
	value = strings.TrimSpace(value)

	// remove multiple whitespace and end lines
	re := regexp.MustCompile("\n")
	value = re.ReplaceAllString(value, "")
	re = regexp.MustCompile("  +")
	value = re.ReplaceAllString(value, " ")

	return value
}

func runXPathQuery(doc *html.Node, xpath string, common commonXPathConfig) []*html.Node {
	// apply common
	if common != nil {
		xpath = common.applyCommon(xpath)
	}

	found, err := htmlquery.QueryAll(doc, xpath)
	if err != nil {
		logger.Warnf("Error parsing xpath expression '%s': %s", xpath, err.Error())
		return nil
	}

	return found
}

func (s xpathScraperConfig) process(doc *html.Node, common commonXPathConfig) xPathResults {
	var ret xPathResults

	for k, value := range s {
		switch v := value.(type) {
		case string:
			found := runXPathQuery(doc, v, common)

			if len(found) > 0 {
				for i, elem := range found {
					text := htmlquery.InnerText(elem)
					text = commonPostProcess(text)

					ret = ret.setKey(i, k, text)
				}
			}
		case map[interface{}]interface{}:
			attrConfig := xpathScraperAttrConfig(v)

			found := runXPathQuery(doc, attrConfig.getSelector(), common)

			if len(found) > 0 {
				// check if we're concatenating the results into a single result
				if attrConfig.hasConcat() {
					result := attrConfig.concatenateResults(found)
					result = attrConfig.postProcess(result)
					const i = 0
					ret = ret.setKey(i, k, result)
				} else {
					for i, elem := range found {
						text := htmlquery.InnerText(elem)
						text = commonPostProcess(text)
						text = attrConfig.postProcess(text)

						ret = ret.setKey(i, k, text)
					}
				}
			}
		}
	}

	return ret
}

type xpathScrapers map[string]*xpathScraper

type xpathScraper struct {
	Common    commonXPathConfig  `yaml:"common"`
	Scene     xpathScraperConfig `yaml:"scene"`
	Performer xpathScraperConfig `yaml:"performer"`
}

const (
	XPathScraperConfigSceneTags       = "Tags"
	XPathScraperConfigScenePerformers = "Performers"
	XPathScraperConfigSceneStudio     = "Studio"
)

func (s xpathScraper) GetSceneSimple() xpathScraperConfig {
	// exclude the complex sub-configs
	ret := make(xpathScraperConfig)
	mapped := s.Scene

	if mapped != nil {
		for k, v := range mapped {
			if k != XPathScraperConfigSceneTags && k != XPathScraperConfigScenePerformers && k != XPathScraperConfigSceneStudio {
				ret[k] = v
			}
		}
	}

	return ret
}

func (s xpathScraper) getSceneSubMap(key string) xpathScraperConfig {
	var ret map[interface{}]interface{}
	mapped := s.Scene

	if mapped != nil {
		v, ok := mapped[key]
		if ok {
			ret, _ = v.(map[interface{}]interface{})
		}
	}

	if ret != nil {
		return createXPathScraperConfig(ret)
	}

	return nil
}

func (s xpathScraper) GetScenePerformers() xpathScraperConfig {
	return s.getSceneSubMap(XPathScraperConfigScenePerformers)
}

func (s xpathScraper) GetSceneTags() xpathScraperConfig {
	return s.getSceneSubMap(XPathScraperConfigSceneTags)
}

func (s xpathScraper) GetSceneStudio() xpathScraperConfig {
	return s.getSceneSubMap(XPathScraperConfigSceneStudio)
}

func (s xpathScraper) scrapePerformer(doc *html.Node) (*models.ScrapedPerformer, error) {
	var ret models.ScrapedPerformer

	performerMap := s.Performer
	if performerMap == nil {
		return nil, nil
	}

	results := performerMap.process(doc, s.Common)
	if len(results) > 0 {
		results[0].apply(&ret)
	}

	return &ret, nil
}

func (s xpathScraper) scrapePerformers(doc *html.Node) ([]*models.ScrapedPerformer, error) {
	var ret []*models.ScrapedPerformer

	performerMap := s.Performer
	if performerMap == nil {
		return nil, nil
	}

	results := performerMap.process(doc, s.Common)
	for _, r := range results {
		var p models.ScrapedPerformer
		r.apply(&p)
		ret = append(ret, &p)
	}

	return ret, nil
}

func (s xpathScraper) scrapeScene(doc *html.Node) (*models.ScrapedScene, error) {
	var ret models.ScrapedScene

	sceneMap := s.GetSceneSimple()
	if sceneMap == nil {
		return nil, nil
	}

	scenePerformersMap := s.GetScenePerformers()
	sceneTagsMap := s.GetSceneTags()
	sceneStudioMap := s.GetSceneStudio()

	results := sceneMap.process(doc, s.Common)
	if len(results) > 0 {
		results[0].apply(&ret)

		// now apply the performers and tags
		if scenePerformersMap != nil {
			performerResults := scenePerformersMap.process(doc, s.Common)

			for _, p := range performerResults {
				performer := &models.ScrapedScenePerformer{}
				p.apply(performer)
				ret.Performers = append(ret.Performers, performer)
			}
		}

		if sceneTagsMap != nil {
			tagResults := sceneTagsMap.process(doc, s.Common)

			for _, p := range tagResults {
				tag := &models.ScrapedSceneTag{}
				p.apply(tag)
				ret.Tags = append(ret.Tags, tag)
			}
		}

		if sceneStudioMap != nil {
			studioResults := sceneStudioMap.process(doc, s.Common)

			if len(studioResults) > 0 {
				studio := &models.ScrapedSceneStudio{}
				studioResults[0].apply(studio)
				ret.Studio = studio
			}
		}
	}

	return &ret, nil
}

type xPathResult map[string]string
type xPathResults []xPathResult

func (r xPathResult) apply(dest interface{}) {
	destVal := reflect.ValueOf(dest)

	// dest should be a pointer
	destVal = destVal.Elem()

	for key, value := range r {
		field := destVal.FieldByName(key)

		if field.IsValid() {
			var reflectValue reflect.Value
			if field.Kind() == reflect.Ptr {
				// need to copy the value, otherwise everything is set to the
				// same pointer
				localValue := value
				reflectValue = reflect.ValueOf(&localValue)
			} else {
				reflectValue = reflect.ValueOf(value)
			}

			field.Set(reflectValue)
		} else {
			logger.Errorf("Field %s does not exist in %T", key, dest)
		}
	}
}

func (r xPathResults) setKey(index int, key string, value string) xPathResults {
	if index >= len(r) {
		r = append(r, make(xPathResult))
	}

	r[index][key] = value
	return r
}

func loadURL(url string) (*html.Node, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.130 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	r, err := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	return html.Parse(r)
}

func scrapePerformerURLXpath(c scraperTypeConfig, url string) (*models.ScrapedPerformer, error) {
	scraper := c.scraperConfig.XPathScrapers[c.Scraper]

	if scraper == nil {
		return nil, errors.New("xpath scraper with name " + c.Scraper + " not found in config")
	}

	doc, err := loadURL(url)

	if err != nil {
		return nil, err
	}

	return scraper.scrapePerformer(doc)
}

func scrapeSceneURLXPath(c scraperTypeConfig, url string) (*models.ScrapedScene, error) {
	scraper := c.scraperConfig.XPathScrapers[c.Scraper]

	if scraper == nil {
		return nil, errors.New("xpath scraper with name " + c.Scraper + " not found in config")
	}

	doc, err := loadURL(url)

	if err != nil {
		return nil, err
	}

	return scraper.scrapeScene(doc)
}

func scrapePerformerNamesXPath(c scraperTypeConfig, name string) ([]*models.ScrapedPerformer, error) {
	scraper := c.scraperConfig.XPathScrapers[c.Scraper]

	if scraper == nil {
		return nil, errors.New("xpath scraper with name " + c.Scraper + " not found in config")
	}

	const placeholder = "{}"

	// replace the placeholder string with the URL-escaped name
	escapedName := url.QueryEscape(name)

	u := c.QueryURL
	u = strings.Replace(u, placeholder, escapedName, -1)

	doc, err := htmlquery.LoadURL(u)

	if err != nil {
		return nil, err
	}

	return scraper.scrapePerformers(doc)
}

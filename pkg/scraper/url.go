package scraper

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/net/html/charset"
	"golang.org/x/net/publicsuffix"

	"github.com/stashapp/stash/pkg/logger"
)

// Timeout for the scrape http request. Includes transfer time. May want to make this
// configurable at some point.
const scrapeGetTimeout = time.Second * 60
const scrapeDefaultSleep = time.Second * 2

// see: https://intoli.com/blog/not-possible-to-block-chrome-headless/
const bypassScript = `(function(w, n, wn) {
// Pass the Webdriver Test.
  Object.defineProperty(n, 'webdriver', {
    get: () => false,
  });

// Pass the Plugins Length Test.
// Overwrite the plugins property to use a custom getter.
   Object.defineProperty(n, 'plugins', {
	// This just needs to have length > 0 for the current test,
	// but we could mock the plugins too if necessary.
    get: () => [1, 2, 3, 4, 5],
  });

// Pass the Languages Test.
// Overwrite the plugins property to use a custom getter.
   Object.defineProperty(n, 'languages', {
     get: () => ['en-US', 'en'],
  });

// Pass the Chrome Test.
// We can mock this in as much depth as we need for the test.
   w.chrome = {
    runtime: {},
  };

// Pass the Permissions Test.
  const originalQuery = wn.permissions.query;
  return wn.permissions.query = (parameters) => (
    parameters.name === 'notifications' ?
      Promise.resolve({ state: Notification.permission }) :
      originalQuery(parameters)
  );
 })(window, navigator, window.navigator);`

func loadURL(url string, scraperConfig config, globalConfig GlobalConfig) (io.Reader, error) {
	driverOptions := scraperConfig.DriverOptions
	if driverOptions != nil && driverOptions.UseCDP {
		// get the page using chrome dp
		return urlFromCDP(url, *driverOptions, globalConfig)
	}

	// get the page using http.Client
	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, er := cookiejar.New(&options)
	if er != nil {
		return nil, er
	}

	setCookies(jar, scraperConfig)
	printCookies(jar, scraperConfig, "Jar cookies set from scraper")

	client := &http.Client{
		Transport: &http.Transport{ // ignore insecure certificates
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !globalConfig.GetScraperCertCheck()},
		},
		Timeout: scrapeGetTimeout,
		// defaultCheckRedirect code with max changed from 10 to 20
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 20 {
				return errors.New("stopped after 20 redirects")
			}
			return nil
		},
		Jar: jar,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	userAgent := globalConfig.GetScraperUserAgent()
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	if driverOptions != nil { // setting the Headers after the UA allows us to override it from inside the scraper
		for _, h := range driverOptions.Headers {
			if h.Key != "" {
				req.Header.Set(h.Key, h.Value)
				logger.Debugf("[scraper] adding header <%s:%s>", h.Key, h.Value)
			}
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http error %d:%s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	bodyReader := bytes.NewReader(body)
	printCookies(jar, scraperConfig, "Jar cookies found for scraper urls")

	return charset.NewReader(bodyReader, resp.Header.Get("Content-Type"))
}

// func urlFromCDP uses chrome cdp and DOM to load and process the url
// if remote is set as true in the scraperConfig  it will try to use localhost:9222
// else it will look for google-chrome in path
func urlFromCDP(url string, driverOptions scraperDriverOptions, globalConfig GlobalConfig) (io.Reader, error) {

	if !driverOptions.UseCDP {
		return nil, fmt.Errorf("Url shouldn't be feetched through CDP")
	}

	sleepDuration := scrapeDefaultSleep

	if driverOptions.Sleep > 0 {
		sleepDuration = time.Duration(driverOptions.Sleep) * time.Second
	}

	act := context.Background()

	// if scraperCDPPath is a remote address, then allocate accordingly
	cdpPath := globalConfig.GetScraperCDPPath()
	if cdpPath != "" {
		var cancelAct context.CancelFunc

		if isCDPPathHTTP(globalConfig) || isCDPPathWS(globalConfig) {
			remote := cdpPath

			// if CDPPath is http(s) then we need to get the websocket URL
			if isCDPPathHTTP(globalConfig) {
				var err error
				remote, err = getRemoteCDPWSAddress(remote)
				if err != nil {
					return nil, err
				}
			}

			act, cancelAct = chromedp.NewRemoteAllocator(context.Background(), remote)
		} else {
			// use a temporary user directory for chrome
			dir, err := ioutil.TempDir("", "stash-chromedp")
			if err != nil {
				return nil, err
			}
			defer os.RemoveAll(dir)

			opts := append(chromedp.DefaultExecAllocatorOptions[:],
				chromedp.UserDataDir(dir),
				chromedp.ExecPath(cdpPath),
			)
			act, cancelAct = chromedp.NewExecAllocator(act, opts...)
		}

		defer cancelAct()
	}

	ctx, cancel := chromedp.NewContext(act)
	defer cancel()

	// add a fixed timeout for the http request
	ctx, cancel = context.WithTimeout(ctx, scrapeGetTimeout)
	defer cancel()

	var res string
	headers := cdpHeaders(driverOptions)

	err := chromedp.Run(ctx,
		network.Enable(),
		setCDPCookies(driverOptions),
		printCDPCookies(driverOptions, "Cookies found"),
		network.SetExtraHTTPHeaders(network.Headers(headers)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			_, err = page.AddScriptToEvaluateOnNewDocument(bypassScript).Do(ctx)
			if err != nil {
				return err
			}
			return nil
		}),
		chromedp.Navigate(url),
		chromedp.Sleep(sleepDuration),
		setCDPClicks(driverOptions),
		chromedp.OuterHTML("html", &res, chromedp.ByQuery),
		printCDPCookies(driverOptions, "Cookies set"),
	)

	if err != nil {
		return nil, err
	}

	return strings.NewReader(res), nil
}

// click all xpaths listed in the scraper config
func setCDPClicks(driverOptions scraperDriverOptions) chromedp.Tasks {
	var tasks chromedp.Tasks
	for _, click := range driverOptions.Clicks { // for each click element find the node from the xpath and add a click action
		if click.XPath != "" {
			xpath := click.XPath
			waitDuration := scrapeDefaultSleep
			if click.Sleep > 0 {
				waitDuration = time.Duration(click.Sleep) * time.Second
			}

			action := chromedp.ActionFunc(func(ctx context.Context) error {
				var nodes []*cdp.Node
				if err := chromedp.Nodes(xpath, &nodes, chromedp.AtLeast(0)).Do(ctx); err != nil {
					logger.Debugf("Error %s looking for click xpath %s.\n", err, xpath)
					return err
				}
				if len(nodes) == 0 {
					logger.Debugf("Click xpath %s not found in page.\n", xpath)
					return nil
				}
				logger.Debugf("Clicking %s\n", xpath)
				return chromedp.MouseClickNode(nodes[0]).Do(ctx)
			})

			tasks = append(tasks, action)
			tasks = append(tasks, chromedp.Sleep(waitDuration))
		}

	}
	return tasks
}

// getRemoteCDPWSAddress returns the complete remote address that is required to access the cdp instance
func getRemoteCDPWSAddress(address string) (string, error) {
	resp, err := http.Get(address)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	remote := result["webSocketDebuggerUrl"].(string)
	logger.Debugf("Remote cdp instance found %s", remote)
	return remote, err
}

func cdpNetwork(enable bool) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		if enable {
			network.Enable().Do(ctx)
		} else {
			network.Disable().Do(ctx)
		}
		return nil
	})
}

func cdpHeaders(driverOptions scraperDriverOptions) map[string]interface{} {
	headers := map[string]interface{}{}
	if driverOptions.Headers != nil {
		for _, h := range driverOptions.Headers {
			if h.Key != "" {
				headers[h.Key] = h.Value
				logger.Debugf("[scraper] adding header <%s:%s>", h.Key, h.Value)
			}
		}
	}
	return headers
}

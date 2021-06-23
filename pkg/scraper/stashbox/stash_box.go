package stashbox

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Yamashou/gqlgenc/client"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scraper"
	"github.com/stashapp/stash/pkg/scraper/stashbox/graphql"
	"github.com/stashapp/stash/pkg/utils"
)

// Timeout to get the image. Includes transfer time. May want to make this
// configurable at some point.
const imageGetTimeout = time.Second * 30

// Client represents the client interface to a stash-box server instance.
type Client struct {
	client     *graphql.Client
	txnManager models.TransactionManager
	endpoint   string
}

// NewClient returns a new instance of a stash-box client.
func NewClient(box models.StashBox, txnManager models.TransactionManager) *Client {
	authHeader := func(req *http.Request) {
		req.Header.Set("ApiKey", box.APIKey)
	}

	client := &graphql.Client{
		Client: client.NewClient(http.DefaultClient, box.Endpoint, authHeader),
	}

	return &Client{
		client:     client,
		txnManager: txnManager,
		endpoint:   box.Endpoint,
	}
}

// QueryStashBoxScene queries stash-box for scenes using a query string.
func (c Client) QueryStashBoxScene(queryStr string) ([]*models.ScrapedScene, error) {
	scenes, err := c.client.SearchScene(context.TODO(), queryStr)
	if err != nil {
		return nil, err
	}

	sceneFragments := scenes.SearchScene

	var ret []*models.ScrapedScene
	for _, s := range sceneFragments {
		ss, err := c.sceneFragmentToScrapedScene(s)
		if err != nil {
			return nil, err
		}
		ret = append(ret, ss)
	}

	return ret, nil
}

// FindStashBoxScenesByFingerprints queries stash-box for scenes using every
// scene's MD5/OSHASH checksum, or PHash
func (c Client) FindStashBoxScenesByFingerprints(sceneIDs []int) ([]*models.ScrapedScene, error) {
	var fingerprints []*graphql.FingerprintQueryInput

	if err := c.txnManager.WithReadTxn(context.TODO(), func(r models.ReaderRepository) error {
		qb := r.Scene()

		for _, sceneID := range sceneIDs {
			scene, err := qb.Find(sceneID)
			if err != nil {
				return err
			}

			if scene == nil {
				return fmt.Errorf("scene with id %d not found", sceneID)
			}

			if scene.Checksum.Valid {
				fingerprints = append(fingerprints, &graphql.FingerprintQueryInput{
					Hash:      scene.Checksum.String,
					Algorithm: graphql.FingerprintAlgorithmMd5,
				})
			}

			if scene.OSHash.Valid {
				fingerprints = append(fingerprints, &graphql.FingerprintQueryInput{
					Hash:      scene.OSHash.String,
					Algorithm: graphql.FingerprintAlgorithmOshash,
				})
			}

			if scene.Phash.Valid {
				fingerprints = append(fingerprints, &graphql.FingerprintQueryInput{
					Hash:      utils.PhashToString(scene.Phash.Int64),
					Algorithm: graphql.FingerprintAlgorithmPhash,
				})
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return c.findStashBoxScenesByFingerprints(fingerprints)
}

func (c Client) findStashBoxScenesByFingerprints(fingerprints []*graphql.FingerprintQueryInput) ([]*models.ScrapedScene, error) {
	var ret []*models.ScrapedScene
	for i := 0; i < len(fingerprints); i += 100 {
		end := i + 100
		if end > len(fingerprints) {
			end = len(fingerprints)
		}
		scenes, err := c.client.FindScenesByFullFingerprints(context.TODO(), fingerprints[i:end])

		if err != nil {
			return nil, err
		}

		sceneFragments := scenes.FindScenesByFullFingerprints

		for _, s := range sceneFragments {
			ss, err := c.sceneFragmentToScrapedScene(s)
			if err != nil {
				return nil, err
			}
			ret = append(ret, ss)
		}
	}

	return ret, nil
}

func (c Client) SubmitStashBoxFingerprints(sceneIDs []string, endpoint string) (bool, error) {
	ids, err := utils.StringSliceToIntSlice(sceneIDs)
	if err != nil {
		return false, err
	}

	var fingerprints []graphql.FingerprintSubmission

	if err := c.txnManager.WithReadTxn(context.TODO(), func(r models.ReaderRepository) error {
		qb := r.Scene()

		for _, sceneID := range ids {
			scene, err := qb.Find(sceneID)
			if err != nil {
				return err
			}

			if scene == nil {
				continue
			}

			stashIDs, err := qb.GetStashIDs(sceneID)
			if err != nil {
				return err
			}

			sceneStashID := ""
			for _, stashID := range stashIDs {
				if stashID.Endpoint == endpoint {
					sceneStashID = stashID.StashID
				}
			}

			if sceneStashID != "" {
				if scene.Checksum.Valid && scene.Duration.Valid {
					fingerprint := graphql.FingerprintInput{
						Hash:      scene.Checksum.String,
						Algorithm: graphql.FingerprintAlgorithmMd5,
						Duration:  int(scene.Duration.Float64),
					}
					fingerprints = append(fingerprints, graphql.FingerprintSubmission{
						SceneID:     sceneStashID,
						Fingerprint: &fingerprint,
					})
				}

				if scene.OSHash.Valid && scene.Duration.Valid {
					fingerprint := graphql.FingerprintInput{
						Hash:      scene.OSHash.String,
						Algorithm: graphql.FingerprintAlgorithmOshash,
						Duration:  int(scene.Duration.Float64),
					}
					fingerprints = append(fingerprints, graphql.FingerprintSubmission{
						SceneID:     sceneStashID,
						Fingerprint: &fingerprint,
					})
				}

				if scene.Phash.Valid && scene.Duration.Valid {
					fingerprint := graphql.FingerprintInput{
						Hash:      utils.PhashToString(scene.Phash.Int64),
						Algorithm: graphql.FingerprintAlgorithmPhash,
						Duration:  int(scene.Duration.Float64),
					}
					fingerprints = append(fingerprints, graphql.FingerprintSubmission{
						SceneID:     sceneStashID,
						Fingerprint: &fingerprint,
					})
				}
			}
		}

		return nil
	}); err != nil {
		return false, err
	}

	return c.submitStashBoxFingerprints(fingerprints)
}

func (c Client) submitStashBoxFingerprints(fingerprints []graphql.FingerprintSubmission) (bool, error) {
	for _, fingerprint := range fingerprints {
		_, err := c.client.SubmitFingerprint(context.TODO(), fingerprint)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

// QueryStashBoxPerformer queries stash-box for performers using a query string.
func (c Client) QueryStashBoxPerformer(queryStr string) ([]*models.StashBoxPerformerQueryResult, error) {
	performers, err := c.queryStashBoxPerformer(queryStr)

	res := []*models.StashBoxPerformerQueryResult{
		{
			Query:   queryStr,
			Results: performers,
		},
	}
	return res, err
}

func (c Client) queryStashBoxPerformer(queryStr string) ([]*models.ScrapedScenePerformer, error) {
	performers, err := c.client.SearchPerformer(context.TODO(), queryStr)
	if err != nil {
		return nil, err
	}

	performerFragments := performers.SearchPerformer

	var ret []*models.ScrapedScenePerformer
	for _, fragment := range performerFragments {
		performer := performerFragmentToScrapedScenePerformer(*fragment)
		ret = append(ret, performer)
	}

	return ret, nil
}

// FindStashBoxPerformersByNames queries stash-box for performers by name
func (c Client) FindStashBoxPerformersByNames(performerIDs []string) ([]*models.StashBoxPerformerQueryResult, error) {
	ids, err := utils.StringSliceToIntSlice(performerIDs)
	if err != nil {
		return nil, err
	}

	var performers []*models.Performer

	if err := c.txnManager.WithReadTxn(context.TODO(), func(r models.ReaderRepository) error {
		qb := r.Performer()

		for _, performerID := range ids {
			performer, err := qb.Find(performerID)
			if err != nil {
				return err
			}

			if performer == nil {
				return fmt.Errorf("performer with id %d not found", performerID)
			}

			if performer.Name.Valid {
				performers = append(performers, performer)
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return c.findStashBoxPerformersByNames(performers)
}

func (c Client) findStashBoxPerformersByNames(performers []*models.Performer) ([]*models.StashBoxPerformerQueryResult, error) {
	var ret []*models.StashBoxPerformerQueryResult
	for _, performer := range performers {
		if performer.Name.Valid {
			performerResults, err := c.queryStashBoxPerformer(performer.Name.String)
			if err != nil {
				return nil, err
			}

			result := models.StashBoxPerformerQueryResult{
				Query:   strconv.Itoa(performer.ID),
				Results: performerResults,
			}

			ret = append(ret, &result)
		}
	}

	return ret, nil
}

func findURL(urls []*graphql.URLFragment, urlType string) *string {
	for _, u := range urls {
		if u.Type == urlType {
			ret := u.URL
			return &ret
		}
	}

	return nil
}

func enumToStringPtr(e fmt.Stringer, titleCase bool) *string {
	if e != nil {
		ret := e.String()
		if titleCase {
			ret = strings.Title(strings.ToLower(ret))
		}
		return &ret
	}

	return nil
}

func formatMeasurements(m graphql.MeasurementsFragment) *string {
	if m.BandSize != nil && m.CupSize != nil && m.Hip != nil && m.Waist != nil {
		ret := fmt.Sprintf("%d%s-%d-%d", *m.BandSize, *m.CupSize, *m.Waist, *m.Hip)
		return &ret
	}

	return nil
}

func formatCareerLength(start, end *int) *string {
	if start == nil && end == nil {
		return nil
	}

	var ret string
	if end == nil {
		ret = fmt.Sprintf("%d -", *start)
	} else if start == nil {
		ret = fmt.Sprintf("- %d", *end)
	} else {
		ret = fmt.Sprintf("%d - %d", *start, *end)
	}

	return &ret
}

func formatBodyModifications(m []*graphql.BodyModificationFragment) *string {
	if len(m) == 0 {
		return nil
	}

	var retSlice []string
	for _, f := range m {
		if f.Description == nil {
			retSlice = append(retSlice, f.Location)
		} else {
			retSlice = append(retSlice, fmt.Sprintf("%s, %s", f.Location, *f.Description))
		}
	}

	ret := strings.Join(retSlice, "; ")
	return &ret
}

func fetchImage(url string) (*string, error) {
	client := &http.Client{
		Timeout: imageGetTimeout,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

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

func performerFragmentToScrapedScenePerformer(p graphql.PerformerFragment) *models.ScrapedScenePerformer {
	id := p.ID
	images := []string{}
	for _, image := range p.Images {
		images = append(images, image.URL)
	}
	sp := &models.ScrapedScenePerformer{
		Name:         p.Name,
		Country:      p.Country,
		Measurements: formatMeasurements(p.Measurements),
		CareerLength: formatCareerLength(p.CareerStartYear, p.CareerEndYear),
		Tattoos:      formatBodyModifications(p.Tattoos),
		Piercings:    formatBodyModifications(p.Piercings),
		Twitter:      findURL(p.Urls, "TWITTER"),
		RemoteSiteID: &id,
		Images:       images,
		// TODO - tags not currently supported
	}

	if p.Height != nil && *p.Height > 0 {
		hs := strconv.Itoa(*p.Height)
		sp.Height = &hs
	}

	if p.Birthdate != nil {
		b := p.Birthdate.Date
		sp.Birthdate = &b
	}

	if p.Gender != nil {
		sp.Gender = enumToStringPtr(p.Gender, false)
	}

	if p.Ethnicity != nil {
		sp.Ethnicity = enumToStringPtr(p.Ethnicity, true)
	}

	if p.EyeColor != nil {
		sp.EyeColor = enumToStringPtr(p.EyeColor, true)
	}

	if p.BreastType != nil {
		sp.FakeTits = enumToStringPtr(p.BreastType, true)
	}

	return sp
}

func studioFragmentToScrapedSceneStudio(s *graphql.StudioFragment) *models.ScrapedSceneStudio {
	studioID := s.ID
	var image *string
	if len(s.Images) > 0 {
		image = &s.Images[0].URL
	}
	ss := models.ScrapedSceneStudio{
		Name:         s.Name,
		URL:          findURL(s.Urls, "HOME"),
		Image:        image,
		RemoteSiteID: &studioID,
	}

	return &ss
}

func getFirstImage(images []*graphql.ImageFragment) *string {
	ret, err := fetchImage(images[0].URL)
	if err != nil {
		logger.Warnf("Error fetching image %s: %s", images[0].URL, err.Error())
	}

	return ret
}

func getFingerprints(scene *graphql.SceneFragment) []*models.StashBoxFingerprint {
	fingerprints := []*models.StashBoxFingerprint{}
	for _, fp := range scene.Fingerprints {
		fingerprint := models.StashBoxFingerprint{
			Algorithm: fp.Algorithm.String(),
			Hash:      fp.Hash,
			Duration:  fp.Duration,
		}
		fingerprints = append(fingerprints, &fingerprint)
	}
	return fingerprints
}

func (c Client) sceneFragmentToScrapedScene(s *graphql.SceneFragment) (*models.ScrapedScene, error) {
	stashID := s.ID
	ss := &models.ScrapedScene{
		Title:        s.Title,
		Date:         s.Date,
		Details:      s.Details,
		URL:          findURL(s.Urls, "STUDIO"),
		Duration:     s.Duration,
		RemoteSiteID: &stashID,
		Fingerprints: getFingerprints(s),
		// Image
		// stash_id
	}

	if len(s.Images) > 0 {
		// TODO - #454 code sorts images by aspect ratio according to a wanted
		// orientation. I'm just grabbing the first for now
		ss.Image = getFirstImage(s.Images)
	}

	if s.Studio != nil {
		scrapedStudio := studioFragmentToScrapedSceneStudio(s.Studio)

		studio, err := c.matchScrapedSceneStudio(s.Studio)
		if err != nil {
			return nil, err
		}
		if studio != nil {
			id := strconv.Itoa(studio.ID)
			scrapedStudio.ID = &id
		}
		ss.Studio = scrapedStudio
	}

	for _, p := range s.Performers {
		sp := performerFragmentToScrapedScenePerformer(p.Performer)

		performer, err := c.matchScrapedScenePerformer(p.Performer)
		if err != nil {
			return nil, err
		}
		if performer != nil {
			id := strconv.Itoa(performer.ID)
			sp.ID = &id
		}
		ss.Performers = append(ss.Performers, sp)
	}

	for _, t := range s.Tags {
		st := &models.ScrapedSceneTag{
			Name: t.Name,
		}

		if err := c.txnManager.WithTxn(context.TODO(), func(r models.Repository) error {
			return scraper.MatchScrapedSceneTag(r.Tag(), st)
		}); err != nil {
			return nil, err
		}

		ss.Tags = append(ss.Tags, st)
	}

	return ss, nil
}

func (c Client) FindStashBoxPerformerByID(id string) (*models.ScrapedScenePerformer, error) {
	performer, err := c.client.FindPerformerByID(context.TODO(), id)
	if err != nil {
		return nil, err
	}

	ret := performerFragmentToScrapedScenePerformer(*performer.FindPerformer)
	return ret, nil
}

func (c Client) FindStashBoxPerformerByName(name string) (*models.ScrapedScenePerformer, error) {
	performers, err := c.client.SearchPerformer(context.TODO(), name)
	if err != nil {
		return nil, err
	}

	var ret *models.ScrapedScenePerformer
	for _, performer := range performers.SearchPerformer {
		if strings.EqualFold(performer.Name, name) {
			ret = performerFragmentToScrapedScenePerformer(*performer)
		}
	}

	return ret, nil
}

func (c Client) FindStashBoxSceneByID(id string) (*models.ScrapedScene, error) {
	scene, err := c.client.FindSceneByID(context.TODO(), id)
	if err != nil {
		return nil, err
	}

	return c.sceneFragmentToScrapedScene(scene.FindScene)
}

func (c Client) matchScrapedScenePerformer(fragment graphql.PerformerFragment) (*models.Performer, error) {
	var err error
	var performers []*models.Performer
	if err := c.txnManager.WithTxn(context.TODO(), func(r models.Repository) error {
		pqb := r.Performer()
		performers, err = pqb.FindByStashID(fragment.ID, c.endpoint)
		return err
	}); err != nil {
		return nil, err
	}

	if len(performers) == 0 {
		// Check if the performer exists with an old, merged id.
		// If that is the case, replace the old with the new id.
		if err := c.txnManager.WithTxn(context.TODO(), func(r models.Repository) error {
			pqb := r.Performer()
			for _, mergedID := range fragment.MergedIds {
				performers, err = pqb.FindByStashID(mergedID, c.endpoint)
				if err != nil {
					return err
				}

				if len(performers) > 0 {
					for _, performer := range performers {
						stashIDs, err := pqb.GetStashIDs(performer.ID)
						if err != nil {
							return err
						}
						newStashIDs := []models.StashID{{
							StashID:  fragment.ID,
							Endpoint: c.endpoint,
						}}
						for _, stashID := range stashIDs {
							if stashID.StashID != mergedID || stashID.Endpoint != c.endpoint {
								newStashIDs = append(newStashIDs, *stashID)
							}
						}
						pqb.UpdateStashIDs(performer.ID, newStashIDs)
					}
					return nil
				}
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	if len(performers) == 0 {
		// Check if performer with the same name already exists.
		if err := c.txnManager.WithTxn(context.TODO(), func(r models.Repository) error {
			pqb := r.Performer()
			performers, err = pqb.FindByNames([]string{fragment.Name}, true)
			if err != nil {
				return err
			}
			if len(performers) == 1 {
				stashIDs, err := pqb.GetStashIDs(performers[0].ID)
				if err != nil {
					return err
				}
				newStashIDs := []models.StashID{{
					StashID:  fragment.ID,
					Endpoint: c.endpoint,
				}}
				for _, stashID := range stashIDs {
					newStashIDs = append(newStashIDs, *stashID)
				}
				pqb.UpdateStashIDs(performers[0].ID, newStashIDs)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	if len(performers) == 1 {
		return performers[0], nil
	} else if len(performers) > 1 {
		var ids []int
		for _, p := range performers {
			ids = append(ids, p.ID)
		}
		logger.Errorf("Multiple performers with same stashID found: %v", ids)
		return performers[0], nil
	}

	return nil, nil
}

func (c Client) matchScrapedSceneStudio(fragment *graphql.StudioFragment) (*models.Studio, error) {
	var err error
	var studios []*models.Studio
	if err := c.txnManager.WithTxn(context.TODO(), func(r models.Repository) error {
		studios, err = r.Studio().FindByStashID(fragment.ID, c.endpoint)
		return err
	}); err != nil {
		return nil, err
	}

	if len(studios) == 0 {
		// Check if studio with the same name already exists.
		if err := c.txnManager.WithTxn(context.TODO(), func(r models.Repository) error {
			sqb := r.Studio()
			studio, err := sqb.FindByName(fragment.Name, true)
			if err != nil {
				return err
			}
			if studio != nil {
				stashIDs, err := sqb.GetStashIDs(studio.ID)
				if err != nil {
					return err
				}
				newStashIDs := []models.StashID{{
					StashID:  fragment.ID,
					Endpoint: c.endpoint,
				}}
				for _, stashID := range stashIDs {
					newStashIDs = append(newStashIDs, *stashID)
				}
				sqb.UpdateStashIDs(studio.ID, newStashIDs)

			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	if len(studios) == 1 {
		return studios[0], nil
	} else if len(studios) > 1 {
		var ids []int
		for _, s := range studios {
			ids = append(ids, s.ID)
		}
		logger.Errorf("Multiple studios with same stashID found: %v", ids)
		return studios[0], nil
	}

	return nil, nil
}

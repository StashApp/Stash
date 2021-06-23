package api

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/manager"
	"github.com/stashapp/stash/pkg/manager/config"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scraper"
	"github.com/stashapp/stash/pkg/scraper/stashbox"
	"github.com/stashapp/stash/pkg/utils"
)

// deprecated
func (r *queryResolver) ScrapeFreeones(ctx context.Context, performer_name string) (*models.ScrapedPerformer, error) {
	scrapedPerformer := models.ScrapedPerformerInput{
		Name: &performer_name,
	}
	return manager.GetInstance().ScraperCache.ScrapePerformer(scraper.FreeonesScraperID, scrapedPerformer)
}

// deprecated
func (r *queryResolver) ScrapeFreeonesPerformerList(ctx context.Context, query string) ([]string, error) {
	scrapedPerformers, err := manager.GetInstance().ScraperCache.ScrapePerformerList(scraper.FreeonesScraperID, query)

	if err != nil {
		return nil, err
	}

	var ret []string
	for _, v := range scrapedPerformers {
		name := v.Name
		ret = append(ret, *name)
	}

	return ret, nil
}

func (r *queryResolver) ListPerformerScrapers(ctx context.Context) ([]*models.Scraper, error) {
	return manager.GetInstance().ScraperCache.ListPerformerScrapers(), nil
}

func (r *queryResolver) ListSceneScrapers(ctx context.Context) ([]*models.Scraper, error) {
	return manager.GetInstance().ScraperCache.ListSceneScrapers(), nil
}

func (r *queryResolver) ListGalleryScrapers(ctx context.Context) ([]*models.Scraper, error) {
	return manager.GetInstance().ScraperCache.ListGalleryScrapers(), nil
}

func (r *queryResolver) ListMovieScrapers(ctx context.Context) ([]*models.Scraper, error) {
	return manager.GetInstance().ScraperCache.ListMovieScrapers(), nil
}

func (r *queryResolver) ScrapePerformerList(ctx context.Context, scraperID string, query string) ([]*models.ScrapedPerformer, error) {
	if query == "" {
		return nil, nil
	}

	return manager.GetInstance().ScraperCache.ScrapePerformerList(scraperID, query)
}

func (r *queryResolver) ScrapePerformer(ctx context.Context, scraperID string, scrapedPerformer models.ScrapedPerformerInput) (*models.ScrapedPerformer, error) {
	return manager.GetInstance().ScraperCache.ScrapePerformer(scraperID, scrapedPerformer)
}

func (r *queryResolver) ScrapePerformerURL(ctx context.Context, url string) (*models.ScrapedPerformer, error) {
	return manager.GetInstance().ScraperCache.ScrapePerformerURL(url)
}

func (r *queryResolver) ScrapeScene(ctx context.Context, scraperID string, scene models.SceneUpdateInput) (*models.ScrapedScene, error) {
	return manager.GetInstance().ScraperCache.ScrapeScene(scraperID, scene)
}

func (r *queryResolver) ScrapeSceneURL(ctx context.Context, url string) (*models.ScrapedScene, error) {
	return manager.GetInstance().ScraperCache.ScrapeSceneURL(url)
}

func (r *queryResolver) ScrapeGallery(ctx context.Context, scraperID string, gallery models.GalleryUpdateInput) (*models.ScrapedGallery, error) {
	return manager.GetInstance().ScraperCache.ScrapeGallery(scraperID, gallery)
}

func (r *queryResolver) ScrapeGalleryURL(ctx context.Context, url string) (*models.ScrapedGallery, error) {
	return manager.GetInstance().ScraperCache.ScrapeGalleryURL(url)
}

func (r *queryResolver) ScrapeMovieURL(ctx context.Context, url string) (*models.ScrapedMovie, error) {
	return manager.GetInstance().ScraperCache.ScrapeMovieURL(url)
}

func (r *queryResolver) QueryStashBoxScene(ctx context.Context, input models.StashBoxSceneQueryInput) ([]*models.ScrapedScene, error) {
	boxes := config.GetInstance().GetStashBoxes()

	if input.StashBoxIndex < 0 || input.StashBoxIndex >= len(boxes) {
		return nil, fmt.Errorf("invalid stash_box_index %d", input.StashBoxIndex)
	}

	client := stashbox.NewClient(*boxes[input.StashBoxIndex], r.txnManager)

	if len(input.SceneIds) > 0 {
		ids, err := utils.StringSliceToIntSlice(input.SceneIds)
		if err != nil {
			return nil, err
		}
		return client.FindStashBoxScenesByFingerprints(ids)
	}

	if input.Q != nil {
		return client.QueryStashBoxScene(*input.Q)
	}

	return nil, nil
}

func (r *queryResolver) QueryStashBoxPerformer(ctx context.Context, input models.StashBoxPerformerQueryInput) ([]*models.StashBoxPerformerQueryResult, error) {
	boxes := config.GetInstance().GetStashBoxes()

	if input.StashBoxIndex < 0 || input.StashBoxIndex >= len(boxes) {
		return nil, fmt.Errorf("invalid stash_box_index %d", input.StashBoxIndex)
	}

	client := stashbox.NewClient(*boxes[input.StashBoxIndex], r.txnManager)

	if len(input.PerformerIds) > 0 {
		return client.FindStashBoxPerformersByNames(input.PerformerIds)
	}

	if input.Q != nil {
		return client.QueryStashBoxPerformer(*input.Q)
	}

	return nil, nil
}

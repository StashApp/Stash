package api

import (
	"context"
	"sort"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
)

type Resolver struct{}

func (r *Resolver) Gallery() models.GalleryResolver {
	return &galleryResolver{r}
}
func (r *Resolver) Mutation() models.MutationResolver {
	return &mutationResolver{r}
}
func (r *Resolver) Performer() models.PerformerResolver {
	return &performerResolver{r}
}
func (r *Resolver) Query() models.QueryResolver {
	return &queryResolver{r}
}
func (r *Resolver) Scene() models.SceneResolver {
	return &sceneResolver{r}
}
func (r *Resolver) Image() models.ImageResolver {
	return &imageResolver{r}
}
func (r *Resolver) SceneMarker() models.SceneMarkerResolver {
	return &sceneMarkerResolver{r}
}
func (r *Resolver) SceneError() models.SceneErrorResolver {
	return &sceneErrorResolver{r}
}
func (r *Resolver) Studio() models.StudioResolver {
	return &studioResolver{r}
}
func (r *Resolver) Movie() models.MovieResolver {
	return &movieResolver{r}
}
func (r *Resolver) Subscription() models.SubscriptionResolver {
	return &subscriptionResolver{r}
}
func (r *Resolver) Tag() models.TagResolver {
	return &tagResolver{r}
}

func (r *Resolver) ScrapedSceneTag() models.ScrapedSceneTagResolver {
	return &scrapedSceneTagResolver{r}
}

func (r *Resolver) ScrapedSceneMovie() models.ScrapedSceneMovieResolver {
	return &scrapedSceneMovieResolver{r}
}

func (r *Resolver) ScrapedScenePerformer() models.ScrapedScenePerformerResolver {
	return &scrapedScenePerformerResolver{r}
}

func (r *Resolver) ScrapedSceneStudio() models.ScrapedSceneStudioResolver {
	return &scrapedSceneStudioResolver{r}
}

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }

type galleryResolver struct{ *Resolver }
type performerResolver struct{ *Resolver }
type sceneResolver struct{ *Resolver }
type sceneMarkerResolver struct{ *Resolver }
type imageResolver struct{ *Resolver }
type sceneErrorResolver struct{ *Resolver }
type studioResolver struct{ *Resolver }
type movieResolver struct{ *Resolver }
type tagResolver struct{ *Resolver }
type scrapedSceneTagResolver struct{ *Resolver }
type scrapedSceneMovieResolver struct{ *Resolver }
type scrapedScenePerformerResolver struct{ *Resolver }
type scrapedSceneStudioResolver struct{ *Resolver }

func (r *queryResolver) MarkerWall(ctx context.Context, q *string) ([]*models.SceneMarker, error) {
	qb := models.NewSceneMarkerQueryBuilder()
	return qb.Wall(q)
}

func (r *queryResolver) SceneWall(ctx context.Context, q *string) ([]*models.Scene, error) {
	qb := models.NewSceneQueryBuilder()
	return qb.Wall(q)
}

func (r *queryResolver) MarkerStrings(ctx context.Context, q *string, sort *string) ([]*models.MarkerStringsResultType, error) {
	qb := models.NewSceneMarkerQueryBuilder()
	return qb.GetMarkerStrings(q, sort)
}

func (r *queryResolver) ValidGalleriesForScene(ctx context.Context, scene_id *string) ([]*models.Gallery, error) {
	if scene_id == nil {
		panic("nil scene id") // TODO make scene_id mandatory
	}
	sceneID, _ := strconv.Atoi(*scene_id)
	sqb := models.NewSceneQueryBuilder()
	scene, err := sqb.Find(sceneID)
	if err != nil {
		return nil, err
	}

	qb := models.NewGalleryQueryBuilder()
	validGalleries, err := qb.ValidGalleriesForScenePath(scene.Path)
	sceneGallery, _ := qb.FindBySceneID(sceneID, nil)
	if sceneGallery != nil {
		validGalleries = append(validGalleries, sceneGallery)
	}
	return validGalleries, nil
}

func (r *queryResolver) Stats(ctx context.Context) (*models.StatsResultType, error) {
	scenesQB := models.NewSceneQueryBuilder()
	scenesCount, _ := scenesQB.Count()
	scenesSize, _ := scenesQB.Size()
	imageQB := models.NewImageQueryBuilder()
	imageCount, _ := imageQB.Count()
	imageSize, _ := imageQB.Size()
	galleryQB := models.NewGalleryQueryBuilder()
	galleryCount, _ := galleryQB.Count()
	performersQB := models.NewPerformerQueryBuilder()
	performersCount, _ := performersQB.Count()
	studiosQB := models.NewStudioQueryBuilder()
	studiosCount, _ := studiosQB.Count()
	moviesQB := models.NewMovieQueryBuilder()
	moviesCount, _ := moviesQB.Count()
	tagsQB := models.NewTagQueryBuilder()
	tagsCount, _ := tagsQB.Count()
	return &models.StatsResultType{
		SceneCount:     scenesCount,
		ScenesSize:     int(scenesSize),
		ImageCount:     imageCount,
		ImagesSize:     int(imageSize),
		GalleryCount:   galleryCount,
		PerformerCount: performersCount,
		StudioCount:    studiosCount,
		MovieCount:     moviesCount,
		TagCount:       tagsCount,
	}, nil
}

func (r *queryResolver) Version(ctx context.Context) (*models.Version, error) {
	version, hash, buildtime := GetVersion()

	return &models.Version{
		Version:   &version,
		Hash:      hash,
		BuildTime: buildtime,
	}, nil
}

//Gets latest version (git shorthash commit for now)
func (r *queryResolver) Latestversion(ctx context.Context) (*models.ShortVersion, error) {
	ver, url, err := GetLatestVersion(true)
	if err == nil {
		logger.Infof("Retrieved latest hash: %s", ver)
	} else {
		logger.Errorf("Error while retrieving latest hash: %s", err)
	}

	return &models.ShortVersion{
		Shorthash: ver,
		URL:       url,
	}, err
}

// Get scene marker tags which show up under the video.
func (r *queryResolver) SceneMarkerTags(ctx context.Context, scene_id string) ([]*models.SceneMarkerTag, error) {
	sceneID, _ := strconv.Atoi(scene_id)
	sqb := models.NewSceneMarkerQueryBuilder()
	sceneMarkers, err := sqb.FindBySceneID(sceneID, nil)
	if err != nil {
		return nil, err
	}

	tags := make(map[int]*models.SceneMarkerTag)
	var keys []int
	tqb := models.NewTagQueryBuilder()
	for _, sceneMarker := range sceneMarkers {
		markerPrimaryTag, err := tqb.Find(sceneMarker.PrimaryTagID, nil)
		if err != nil {
			return nil, err
		}
		_, hasKey := tags[markerPrimaryTag.ID]
		var sceneMarkerTag *models.SceneMarkerTag
		if !hasKey {
			sceneMarkerTag = &models.SceneMarkerTag{Tag: markerPrimaryTag}
			tags[markerPrimaryTag.ID] = sceneMarkerTag
			keys = append(keys, markerPrimaryTag.ID)
		} else {
			sceneMarkerTag = tags[markerPrimaryTag.ID]
		}
		tags[markerPrimaryTag.ID].SceneMarkers = append(tags[markerPrimaryTag.ID].SceneMarkers, sceneMarker)
	}

	// Sort so that primary tags that show up earlier in the video are first.
	sort.Slice(keys, func(i, j int) bool {
		a := tags[keys[i]]
		b := tags[keys[j]]
		return a.SceneMarkers[0].Seconds < b.SceneMarkers[0].Seconds
	})

	var result []*models.SceneMarkerTag
	for _, key := range keys {
		result = append(result, tags[key])
	}

	return result, nil
}

// wasFieldIncluded returns true if the given field was included in the request.
// Slices are unmarshalled to empty slices even if the field was omitted. This
// method determines if it was omitted altogether.
func wasFieldIncluded(ctx context.Context, field string) bool {
	rctx := graphql.GetRequestContext(ctx)

	_, ret := rctx.Variables[field]
	return ret
}

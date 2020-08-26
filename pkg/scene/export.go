package scene

import (
	"fmt"
	"math"
	"strconv"

	"github.com/stashapp/stash/pkg/manager/jsonschema"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

// ToBasicJSON converts a scene object into its JSON object equivalent. It
// does not convert the relationships to other objects, with the exception
// of cover image.
func ToBasicJSON(reader models.SceneReader, scene *models.Scene) (*jsonschema.Scene, error) {
	newSceneJSON := jsonschema.Scene{
		CreatedAt: models.JSONTime{Time: scene.CreatedAt.Timestamp},
		UpdatedAt: models.JSONTime{Time: scene.UpdatedAt.Timestamp},
	}

	if scene.Checksum.Valid {
		newSceneJSON.Checksum = scene.Checksum.String
	}

	if scene.OSHash.Valid {
		newSceneJSON.OSHash = scene.OSHash.String
	}

	if scene.Title.Valid {
		newSceneJSON.Title = scene.Title.String
	}

	if scene.URL.Valid {
		newSceneJSON.URL = scene.URL.String
	}

	if scene.Date.Valid {
		newSceneJSON.Date = utils.GetYMDFromDatabaseDate(scene.Date.String)
	}

	if scene.Rating.Valid {
		newSceneJSON.Rating = int(scene.Rating.Int64)
	}

	newSceneJSON.OCounter = scene.OCounter

	if scene.Details.Valid {
		newSceneJSON.Details = scene.Details.String
	}

	newSceneJSON.File = getSceneFileJSON(scene)

	cover, err := reader.GetSceneCover(scene.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting scene cover: %s", err.Error())
	}

	if len(cover) > 0 {
		newSceneJSON.Cover = utils.GetBase64StringFromData(cover)
	}

	return &newSceneJSON, nil
}

func getSceneFileJSON(scene *models.Scene) *jsonschema.SceneFile {
	ret := &jsonschema.SceneFile{}

	if scene.Size.Valid {
		ret.Size = scene.Size.String
	}

	if scene.Duration.Valid {
		ret.Duration = getDecimalString(scene.Duration.Float64)
	}

	if scene.VideoCodec.Valid {
		ret.VideoCodec = scene.VideoCodec.String
	}

	if scene.AudioCodec.Valid {
		ret.AudioCodec = scene.AudioCodec.String
	}

	if scene.Format.Valid {
		ret.Format = scene.Format.String
	}

	if scene.Width.Valid {
		ret.Width = int(scene.Width.Int64)
	}

	if scene.Height.Valid {
		ret.Height = int(scene.Height.Int64)
	}

	if scene.Framerate.Valid {
		ret.Framerate = getDecimalString(scene.Framerate.Float64)
	}

	if scene.Bitrate.Valid {
		ret.Bitrate = int(scene.Bitrate.Int64)
	}

	return ret
}

// GetStudioName returns the name of the provided scene's studio. It returns an
// empty string if there is no studio assigned to the scene.
func GetStudioName(reader models.StudioReader, scene *models.Scene) (string, error) {
	if scene.StudioID.Valid {
		studio, err := reader.Find(int(scene.StudioID.Int64))
		if err != nil {
			return "", err
		}

		if studio != nil {
			return studio.Name.String, nil
		}
	}

	return "", nil
}

// GetGalleryChecksum returns the checksum of the provided scene. It returns an
// empty string if there is no gallery assigned to the scene.
func GetGalleryChecksum(reader models.GalleryReader, scene *models.Scene) (string, error) {
	gallery, err := reader.FindBySceneID(scene.ID)
	if err != nil {
		return "", fmt.Errorf("error getting scene gallery: %s", err.Error())
	}

	if gallery != nil {
		return gallery.Checksum, nil
	}

	return "", nil
}

// GetPerformerNames returns a slice of performer names corresponding to the
// provided scene's performers.
func GetPerformerNames(reader models.PerformerReader, scene *models.Scene) ([]string, error) {
	performers, err := reader.FindNamesBySceneID(scene.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting scene performers: %s", err.Error())
	}

	var results []string
	for _, performer := range performers {
		if performer.Name.Valid {
			results = append(results, performer.Name.String)
		}
	}

	return results, nil
}

// GetTagNames returns a slice of tag names corresponding to the provided
// scene's tags.
func GetTagNames(reader models.TagReader, scene *models.Scene) ([]string, error) {
	tags, err := reader.FindBySceneID(scene.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting scene tags: %s", err.Error())
	}

	return getTagNames(tags), nil
}

func getTagNames(tags []*models.Tag) []string {
	var results []string
	for _, tag := range tags {
		if tag.Name != "" {
			results = append(results, tag.Name)
		}
	}

	return results
}

// GetSceneMoviesJSON returns a slice of SceneMovie JSON representation objects
// corresponding to the provided scene's scene movie relationships.
func GetSceneMoviesJSON(movieReader models.MovieReader, joinReader models.JoinReader, scene *models.Scene) ([]jsonschema.SceneMovie, error) {
	sceneMovies, err := joinReader.GetSceneMovies(scene.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting scene movies: %s", err.Error())
	}

	var results []jsonschema.SceneMovie
	for _, sceneMovie := range sceneMovies {
		movie, err := movieReader.Find(sceneMovie.MovieID)
		if err != nil {
			return nil, fmt.Errorf("error getting movie: %s", err.Error())
		}

		if movie.Name.Valid {
			sceneMovieJSON := jsonschema.SceneMovie{
				MovieName:  movie.Name.String,
				SceneIndex: int(sceneMovie.SceneIndex.Int64),
			}
			results = append(results, sceneMovieJSON)
		}
	}

	return results, nil
}

// GetSceneMarkersJSON returns a slice of SceneMarker JSON representation
// objects corresponding to the provided scene's markers.
func GetSceneMarkersJSON(markerReader models.SceneMarkerReader, tagReader models.TagReader, scene *models.Scene) ([]jsonschema.SceneMarker, error) {
	sceneMarkers, err := markerReader.FindBySceneID(scene.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting scene markers: %s", err.Error())
	}

	var results []jsonschema.SceneMarker

	for _, sceneMarker := range sceneMarkers {
		primaryTag, err := tagReader.Find(sceneMarker.PrimaryTagID)
		if err != nil {
			return nil, fmt.Errorf("invalid primary tag for scene marker: %s", err.Error())
		}

		sceneMarkerTags, err := tagReader.FindBySceneMarkerID(sceneMarker.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid tags for scene marker: %s", err.Error())
		}

		sceneMarkerJSON := jsonschema.SceneMarker{
			Title:      sceneMarker.Title,
			Seconds:    getDecimalString(sceneMarker.Seconds),
			PrimaryTag: primaryTag.Name,
			Tags:       getTagNames(sceneMarkerTags),
			CreatedAt:  models.JSONTime{Time: sceneMarker.CreatedAt.Timestamp},
			UpdatedAt:  models.JSONTime{Time: sceneMarker.UpdatedAt.Timestamp},
		}

		results = append(results, sceneMarkerJSON)
	}

	return results, nil
}

func getDecimalString(num float64) string {
	if num == 0 {
		return ""
	}

	precision := getPrecision(num)
	if precision == 0 {
		precision = 1
	}
	return fmt.Sprintf("%."+strconv.Itoa(precision)+"f", num)
}

func getPrecision(num float64) int {
	if num == 0 {
		return 0
	}

	e := 1.0
	p := 0
	for (math.Round(num*e) / e) != num {
		e *= 10
		p++
	}
	return p
}

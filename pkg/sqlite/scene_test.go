// +build integration

package sqlite_test

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

func TestSceneFind(t *testing.T) {
	withTxn(func(r models.Repository) error {
		// assume that the first scene is sceneWithGalleryPath
		sqb := r.Scene()

		const sceneIdx = 0
		sceneID := sceneIDs[sceneIdx]
		scene, err := sqb.Find(sceneID)

		if err != nil {
			t.Errorf("Error finding scene: %s", err.Error())
		}

		assert.Equal(t, getSceneStringValue(sceneIdx, "Path"), scene.Path)

		sceneID = 0
		scene, err = sqb.Find(sceneID)

		if err != nil {
			t.Errorf("Error finding scene: %s", err.Error())
		}

		assert.Nil(t, scene)

		return nil
	})
}

func TestSceneFindByPath(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()

		const sceneIdx = 1
		scenePath := getSceneStringValue(sceneIdx, "Path")
		scene, err := sqb.FindByPath(scenePath)

		if err != nil {
			t.Errorf("Error finding scene: %s", err.Error())
		}

		assert.Equal(t, sceneIDs[sceneIdx], scene.ID)
		assert.Equal(t, scenePath, scene.Path)

		scenePath = "not exist"
		scene, err = sqb.FindByPath(scenePath)

		if err != nil {
			t.Errorf("Error finding scene: %s", err.Error())
		}

		assert.Nil(t, scene)

		return nil
	})
}

func TestSceneCountByPerformerID(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		count, err := sqb.CountByPerformerID(performerIDs[performerIdxWithScene])

		if err != nil {
			t.Errorf("Error counting scenes: %s", err.Error())
		}

		assert.Equal(t, 1, count)

		count, err = sqb.CountByPerformerID(0)

		if err != nil {
			t.Errorf("Error counting scenes: %s", err.Error())
		}

		assert.Equal(t, 0, count)

		return nil
	})
}

func TestSceneWall(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()

		const sceneIdx = 2
		wallQuery := getSceneStringValue(sceneIdx, "Details")
		scenes, err := sqb.Wall(&wallQuery)

		if err != nil {
			t.Errorf("Error finding scenes: %s", err.Error())
		}

		assert.Len(t, scenes, 1)
		scene := scenes[0]
		assert.Equal(t, sceneIDs[sceneIdx], scene.ID)
		assert.Equal(t, getSceneStringValue(sceneIdx, "Path"), scene.Path)

		wallQuery = "not exist"
		scenes, err = sqb.Wall(&wallQuery)

		if err != nil {
			t.Errorf("Error finding scene: %s", err.Error())
		}

		assert.Len(t, scenes, 0)

		return nil
	})
}

func TestSceneQueryQ(t *testing.T) {
	const sceneIdx = 2

	q := getSceneStringValue(sceneIdx, titleField)

	withTxn(func(r models.Repository) error {
		sqb := r.Scene()

		sceneQueryQ(t, sqb, q, sceneIdx)

		return nil
	})
}

func queryScene(t *testing.T, sqb models.SceneReader, sceneFilter *models.SceneFilterType, findFilter *models.FindFilterType) []*models.Scene {
	t.Helper()
	scenes, _, err := sqb.Query(sceneFilter, findFilter)
	if err != nil {
		t.Errorf("Error querying scene: %s", err.Error())
	}

	return scenes
}

func sceneQueryQ(t *testing.T, sqb models.SceneReader, q string, expectedSceneIdx int) {
	filter := models.FindFilterType{
		Q: &q,
	}
	scenes := queryScene(t, sqb, nil, &filter)

	assert.Len(t, scenes, 1)
	scene := scenes[0]
	assert.Equal(t, sceneIDs[expectedSceneIdx], scene.ID)

	// no Q should return all results
	filter.Q = nil
	scenes = queryScene(t, sqb, nil, &filter)

	assert.Len(t, scenes, totalScenes)
}

func TestSceneQueryPath(t *testing.T) {
	const sceneIdx = 1
	scenePath := getSceneStringValue(sceneIdx, "Path")

	pathCriterion := models.StringCriterionInput{
		Value:    scenePath,
		Modifier: models.CriterionModifierEquals,
	}

	verifyScenesPath(t, pathCriterion)

	pathCriterion.Modifier = models.CriterionModifierNotEquals
	verifyScenesPath(t, pathCriterion)

	pathCriterion.Modifier = models.CriterionModifierMatchesRegex
	pathCriterion.Value = "scene_.*1_Path"
	verifyScenesPath(t, pathCriterion)

	pathCriterion.Modifier = models.CriterionModifierNotMatchesRegex
	verifyScenesPath(t, pathCriterion)
}

func TestSceneQueryURL(t *testing.T) {
	const sceneIdx = 1
	scenePath := getSceneStringValue(sceneIdx, urlField)

	urlCriterion := models.StringCriterionInput{
		Value:    scenePath,
		Modifier: models.CriterionModifierEquals,
	}

	filter := models.SceneFilterType{
		URL: &urlCriterion,
	}

	verifyFn := func(s *models.Scene) {
		t.Helper()
		verifyNullString(t, s.URL, urlCriterion)
	}

	verifySceneQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierNotEquals
	verifySceneQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierMatchesRegex
	urlCriterion.Value = "scene_.*1_URL"
	verifySceneQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierNotMatchesRegex
	verifySceneQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierIsNull
	urlCriterion.Value = ""
	verifySceneQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierNotNull
	verifySceneQuery(t, filter, verifyFn)
}

func TestSceneQueryPathOr(t *testing.T) {
	const scene1Idx = 1
	const scene2Idx = 2

	scene1Path := getSceneStringValue(scene1Idx, "Path")
	scene2Path := getSceneStringValue(scene2Idx, "Path")

	sceneFilter := models.SceneFilterType{
		Path: &models.StringCriterionInput{
			Value:    scene1Path,
			Modifier: models.CriterionModifierEquals,
		},
		Or: &models.SceneFilterType{
			Path: &models.StringCriterionInput{
				Value:    scene2Path,
				Modifier: models.CriterionModifierEquals,
			},
		},
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Scene()

		scenes := queryScene(t, sqb, &sceneFilter, nil)

		assert.Len(t, scenes, 2)
		assert.Equal(t, scene1Path, scenes[0].Path)
		assert.Equal(t, scene2Path, scenes[1].Path)

		return nil
	})
}

func TestSceneQueryPathAndRating(t *testing.T) {
	const sceneIdx = 1
	scenePath := getSceneStringValue(sceneIdx, "Path")
	sceneRating := getRating(sceneIdx)

	sceneFilter := models.SceneFilterType{
		Path: &models.StringCriterionInput{
			Value:    scenePath,
			Modifier: models.CriterionModifierEquals,
		},
		And: &models.SceneFilterType{
			Rating: &models.IntCriterionInput{
				Exact:    int(sceneRating.Int64),
				Upper:    int(sceneRating.Int64),
				Lower:    int(sceneRating.Int64),
				Modifier: models.CriterionModifierEquals,
			},
		},
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Scene()

		scenes := queryScene(t, sqb, &sceneFilter, nil)

		assert.Len(t, scenes, 1)
		assert.Equal(t, scenePath, scenes[0].Path)
		assert.Equal(t, sceneRating.Int64, scenes[0].Rating.Int64)

		return nil
	})
}

func TestSceneQueryPathNotRating(t *testing.T) {
	const sceneIdx = 1

	sceneRating := getRating(sceneIdx)

	pathCriterion := models.StringCriterionInput{
		Value:    "scene_.*1_Path",
		Modifier: models.CriterionModifierMatchesRegex,
	}

	ratingCriterion := models.IntCriterionInput{
		Exact:    int(sceneRating.Int64),
		Upper:    int(sceneRating.Int64),
		Lower:    int(sceneRating.Int64),
		Modifier: models.CriterionModifierEquals,
	}

	sceneFilter := models.SceneFilterType{
		Path: &pathCriterion,
		Not: &models.SceneFilterType{
			Rating: &ratingCriterion,
		},
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Scene()

		scenes := queryScene(t, sqb, &sceneFilter, nil)

		for _, scene := range scenes {
			verifyString(t, scene.Path, pathCriterion)
			ratingCriterion.Modifier = models.CriterionModifierNotEquals
			verifyInt64(t, scene.Rating, ratingCriterion)
		}

		return nil
	})
}

func TestSceneIllegalQuery(t *testing.T) {
	assert := assert.New(t)

	const sceneIdx = 1
	subFilter := models.SceneFilterType{
		Path: &models.StringCriterionInput{
			Value:    getSceneStringValue(sceneIdx, "Path"),
			Modifier: models.CriterionModifierEquals,
		},
	}

	sceneFilter := &models.SceneFilterType{
		And: &subFilter,
		Or:  &subFilter,
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Scene()

		_, _, err := sqb.Query(sceneFilter, nil)
		assert.NotNil(err)

		sceneFilter.Or = nil
		sceneFilter.Not = &subFilter
		_, _, err = sqb.Query(sceneFilter, nil)
		assert.NotNil(err)

		sceneFilter.And = nil
		sceneFilter.Or = &subFilter
		_, _, err = sqb.Query(sceneFilter, nil)
		assert.NotNil(err)

		return nil
	})
}

func verifySceneQuery(t *testing.T, filter models.SceneFilterType, verifyFn func(s *models.Scene)) {
	withTxn(func(r models.Repository) error {
		t.Helper()
		sqb := r.Scene()

		scenes := queryScene(t, sqb, &filter, nil)

		// assume it should find at least one
		assert.Greater(t, len(scenes), 0)

		for _, scene := range scenes {
			verifyFn(scene)
		}

		return nil
	})
}

func verifyScenesPath(t *testing.T, pathCriterion models.StringCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		sceneFilter := models.SceneFilterType{
			Path: &pathCriterion,
		}

		scenes := queryScene(t, sqb, &sceneFilter, nil)

		for _, scene := range scenes {
			verifyString(t, scene.Path, pathCriterion)
		}

		return nil
	})
}

func verifyNullString(t *testing.T, value sql.NullString, criterion models.StringCriterionInput) {
	t.Helper()
	assert := assert.New(t)
	if criterion.Modifier == models.CriterionModifierIsNull {
		if value.Valid && value.String == "" {
			// correct
			return
		}
		assert.False(value.Valid, "expect is null values to be null")
	}
	if criterion.Modifier == models.CriterionModifierNotNull {
		assert.True(value.Valid, "expect is null values to be null")
		assert.Greater(len(value.String), 0)
	}
	if criterion.Modifier == models.CriterionModifierEquals {
		assert.Equal(criterion.Value, value.String)
	}
	if criterion.Modifier == models.CriterionModifierNotEquals {
		assert.NotEqual(criterion.Value, value.String)
	}
	if criterion.Modifier == models.CriterionModifierMatchesRegex {
		assert.True(value.Valid)
		assert.Regexp(regexp.MustCompile(criterion.Value), value)
	}
	if criterion.Modifier == models.CriterionModifierNotMatchesRegex {
		if !value.Valid {
			// correct
			return
		}
		assert.NotRegexp(regexp.MustCompile(criterion.Value), value)
	}
}

func verifyString(t *testing.T, value string, criterion models.StringCriterionInput) {
	t.Helper()
	assert := assert.New(t)
	if criterion.Modifier == models.CriterionModifierEquals {
		assert.Equal(criterion.Value, value)
	}
	if criterion.Modifier == models.CriterionModifierNotEquals {
		assert.NotEqual(criterion.Value, value)
	}
	if criterion.Modifier == models.CriterionModifierMatchesRegex {
		assert.Regexp(regexp.MustCompile(criterion.Value), value)
	}
	if criterion.Modifier == models.CriterionModifierNotMatchesRegex {
		assert.NotRegexp(regexp.MustCompile(criterion.Value), value)
	}
}

func TestSceneQueryRating(t *testing.T) {
	const rating = 3
	ratingCriterion := models.IntCriterionInput{
		Exact:    rating,
		Upper:    rating,
		Lower:    rating,
		Modifier: models.CriterionModifierEquals,
	}

	verifyScenesRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierNotEquals
	verifyScenesRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyScenesRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierLessThan
	verifyScenesRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierIsNull
	verifyScenesRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierNotNull
	verifyScenesRating(t, ratingCriterion)
}

func verifyScenesRating(t *testing.T, ratingCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		sceneFilter := models.SceneFilterType{
			Rating: &ratingCriterion,
		}

		scenes := queryScene(t, sqb, &sceneFilter, nil)

		for _, scene := range scenes {
			verifyInt64(t, scene.Rating, ratingCriterion)
		}

		return nil
	})
}

func verifyInt64(t *testing.T, value sql.NullInt64, criterion models.IntCriterionInput) {
	t.Helper()
	assert := assert.New(t)
	if criterion.Modifier == models.CriterionModifierIsNull {
		assert.False(value.Valid, "expect is null values to be null")
	}
	if criterion.Modifier == models.CriterionModifierNotNull {
		assert.True(value.Valid, "expect is null values to be null")
	}
	if criterion.Modifier == models.CriterionModifierEquals {
		assert.Equal(int64(criterion.Exact), value.Int64)
	}
	if criterion.Modifier == models.CriterionModifierNotEquals {
		assert.NotEqual(int64(criterion.Exact), value.Int64)
	}
	if criterion.Modifier == models.CriterionModifierGreaterThan {
		assert.True(value.Int64 > int64(criterion.Lower))
	}
	if criterion.Modifier == models.CriterionModifierLessThan {
		assert.True(value.Int64 < int64(criterion.Upper))
	}
}

func TestSceneQueryOCounter(t *testing.T) {
	const oCounter = 1
	oCounterCriterion := models.IntCriterionInput{
		Exact:    oCounter,
		Upper:    oCounter,
		Lower:    oCounter,
		Modifier: models.CriterionModifierEquals,
	}

	verifyScenesOCounter(t, oCounterCriterion)

	oCounterCriterion.Modifier = models.CriterionModifierNotEquals
	verifyScenesOCounter(t, oCounterCriterion)

	oCounterCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyScenesOCounter(t, oCounterCriterion)

	oCounterCriterion.Modifier = models.CriterionModifierLessThan
	verifyScenesOCounter(t, oCounterCriterion)
}

func verifyScenesOCounter(t *testing.T, oCounterCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		sceneFilter := models.SceneFilterType{
			OCounter: &oCounterCriterion,
		}

		scenes := queryScene(t, sqb, &sceneFilter, nil)

		for _, scene := range scenes {
			verifyInt(t, scene.OCounter, oCounterCriterion)
		}

		return nil
	})
}

func verifyInt(t *testing.T, value int, criterion models.IntCriterionInput) {
	t.Helper()
	assert := assert.New(t)
	if criterion.Modifier == models.CriterionModifierEquals {
		assert.Equal(criterion.Exact, value)
	}
	if criterion.Modifier == models.CriterionModifierNotEquals {
		assert.NotEqual(criterion.Exact, value)
	}
	if criterion.Modifier == models.CriterionModifierGreaterThan {
		assert.Greater(value, criterion.Lower)
	}
	if criterion.Modifier == models.CriterionModifierLessThan {
		assert.Less(value, criterion.Upper)
	}
}

func TestSceneQueryDuration(t *testing.T) {
	duration := 200.432

	durationCriterion := models.IntCriterionInput{
		Exact:    int(duration),
		Upper:    int(duration),
		Lower:    int(duration),
		Modifier: models.CriterionModifierEquals,
	}
	verifyScenesDuration(t, durationCriterion)

	durationCriterion.Modifier = models.CriterionModifierNotEquals
	verifyScenesDuration(t, durationCriterion)

	durationCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyScenesDuration(t, durationCriterion)

	durationCriterion.Modifier = models.CriterionModifierLessThan
	verifyScenesDuration(t, durationCriterion)

	durationCriterion.Modifier = models.CriterionModifierIsNull
	verifyScenesDuration(t, durationCriterion)

	durationCriterion.Modifier = models.CriterionModifierNotNull
	verifyScenesDuration(t, durationCriterion)
}

func verifyScenesDuration(t *testing.T, durationCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		sceneFilter := models.SceneFilterType{
			Duration: &durationCriterion,
		}

		scenes := queryScene(t, sqb, &sceneFilter, nil)

		for _, scene := range scenes {
			if durationCriterion.Modifier == models.CriterionModifierEquals {
				assert.True(t, scene.Duration.Float64 >= float64(durationCriterion.Exact) && scene.Duration.Float64 < float64(durationCriterion.Exact+1))
			} else if durationCriterion.Modifier == models.CriterionModifierNotEquals {
				assert.True(t, scene.Duration.Float64 < float64(durationCriterion.Exact) || scene.Duration.Float64 >= float64(durationCriterion.Exact+1))
			} else {
				verifyFloat64(t, scene.Duration, durationCriterion)
			}
		}

		return nil
	})
}

func verifyFloat64(t *testing.T, value sql.NullFloat64, criterion models.IntCriterionInput) {
	assert := assert.New(t)
	if criterion.Modifier == models.CriterionModifierIsNull {
		assert.False(value.Valid, "expect is null values to be null")
	}
	if criterion.Modifier == models.CriterionModifierNotNull {
		assert.True(value.Valid, "expect is null values to be null")
	}
	if criterion.Modifier == models.CriterionModifierEquals {
		assert.Equal(float64(criterion.Exact), value.Float64)
	}
	if criterion.Modifier == models.CriterionModifierNotEquals {
		assert.NotEqual(float64(criterion.Exact), value.Float64)
	}
	if criterion.Modifier == models.CriterionModifierGreaterThan {
		assert.True(value.Float64 > float64(criterion.Lower))
	}
	if criterion.Modifier == models.CriterionModifierLessThan {
		assert.True(value.Float64 < float64(criterion.Upper))
	}
}

func TestSceneQueryResolution(t *testing.T) {
	verifyScenesResolution(t, models.ResolutionEnumLow)
	verifyScenesResolution(t, models.ResolutionEnumStandard)
	verifyScenesResolution(t, models.ResolutionEnumStandardHd)
	verifyScenesResolution(t, models.ResolutionEnumFullHd)
	verifyScenesResolution(t, models.ResolutionEnumFourK)
	verifyScenesResolution(t, models.ResolutionEnum("unknown"))
}

func verifyScenesResolution(t *testing.T, resolution models.ResolutionEnum) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		sceneFilter := models.SceneFilterType{
			Resolution: &resolution,
		}

		scenes := queryScene(t, sqb, &sceneFilter, nil)

		for _, scene := range scenes {
			verifySceneResolution(t, scene.Height, resolution)
		}

		return nil
	})
}

func verifySceneResolution(t *testing.T, height sql.NullInt64, resolution models.ResolutionEnum) {
	assert := assert.New(t)
	h := height.Int64

	switch resolution {
	case models.ResolutionEnumLow:
		assert.True(h < 480)
	case models.ResolutionEnumStandard:
		assert.True(h >= 480 && h < 720)
	case models.ResolutionEnumStandardHd:
		assert.True(h >= 720 && h < 1080)
	case models.ResolutionEnumFullHd:
		assert.True(h >= 1080 && h < 2160)
	case models.ResolutionEnumFourK:
		assert.True(h >= 2160)
	}
}

func TestSceneQueryHasMarkers(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		hasMarkers := "true"
		sceneFilter := models.SceneFilterType{
			HasMarkers: &hasMarkers,
		}

		q := getSceneStringValue(sceneIdxWithMarker, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		scenes := queryScene(t, sqb, &sceneFilter, &findFilter)

		assert.Len(t, scenes, 1)
		assert.Equal(t, sceneIDs[sceneIdxWithMarker], scenes[0].ID)

		hasMarkers = "false"
		scenes = queryScene(t, sqb, &sceneFilter, &findFilter)
		assert.Len(t, scenes, 0)

		findFilter.Q = nil
		scenes = queryScene(t, sqb, &sceneFilter, &findFilter)

		assert.NotEqual(t, 0, len(scenes))

		// ensure non of the ids equal the one with gallery
		for _, scene := range scenes {
			assert.NotEqual(t, sceneIDs[sceneIdxWithMarker], scene.ID)
		}

		return nil
	})
}

func TestSceneQueryIsMissingGallery(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		isMissing := "galleries"
		sceneFilter := models.SceneFilterType{
			IsMissing: &isMissing,
		}

		q := getSceneStringValue(sceneIdxWithGallery, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		scenes := queryScene(t, sqb, &sceneFilter, &findFilter)

		assert.Len(t, scenes, 0)

		findFilter.Q = nil
		scenes = queryScene(t, sqb, &sceneFilter, &findFilter)

		// ensure non of the ids equal the one with gallery
		for _, scene := range scenes {
			assert.NotEqual(t, sceneIDs[sceneIdxWithGallery], scene.ID)
		}

		return nil
	})
}

func TestSceneQueryIsMissingStudio(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		isMissing := "studio"
		sceneFilter := models.SceneFilterType{
			IsMissing: &isMissing,
		}

		q := getSceneStringValue(sceneIdxWithStudio, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		scenes := queryScene(t, sqb, &sceneFilter, &findFilter)

		assert.Len(t, scenes, 0)

		findFilter.Q = nil
		scenes = queryScene(t, sqb, &sceneFilter, &findFilter)

		// ensure non of the ids equal the one with studio
		for _, scene := range scenes {
			assert.NotEqual(t, sceneIDs[sceneIdxWithStudio], scene.ID)
		}

		return nil
	})
}

func TestSceneQueryIsMissingMovies(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		isMissing := "movie"
		sceneFilter := models.SceneFilterType{
			IsMissing: &isMissing,
		}

		q := getSceneStringValue(sceneIdxWithMovie, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		scenes := queryScene(t, sqb, &sceneFilter, &findFilter)

		assert.Len(t, scenes, 0)

		findFilter.Q = nil
		scenes = queryScene(t, sqb, &sceneFilter, &findFilter)

		// ensure non of the ids equal the one with movies
		for _, scene := range scenes {
			assert.NotEqual(t, sceneIDs[sceneIdxWithMovie], scene.ID)
		}

		return nil
	})
}

func TestSceneQueryIsMissingPerformers(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		isMissing := "performers"
		sceneFilter := models.SceneFilterType{
			IsMissing: &isMissing,
		}

		q := getSceneStringValue(sceneIdxWithPerformer, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		scenes := queryScene(t, sqb, &sceneFilter, &findFilter)

		assert.Len(t, scenes, 0)

		findFilter.Q = nil
		scenes = queryScene(t, sqb, &sceneFilter, &findFilter)

		assert.True(t, len(scenes) > 0)

		// ensure non of the ids equal the one with movies
		for _, scene := range scenes {
			assert.NotEqual(t, sceneIDs[sceneIdxWithPerformer], scene.ID)
		}

		return nil
	})
}

func TestSceneQueryIsMissingDate(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		isMissing := "date"
		sceneFilter := models.SceneFilterType{
			IsMissing: &isMissing,
		}

		scenes := queryScene(t, sqb, &sceneFilter, nil)

		assert.True(t, len(scenes) > 0)

		// ensure date is null, empty or "0001-01-01"
		for _, scene := range scenes {
			assert.True(t, !scene.Date.Valid || scene.Date.String == "" || scene.Date.String == "0001-01-01")
		}

		return nil
	})
}

func TestSceneQueryIsMissingTags(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		isMissing := "tags"
		sceneFilter := models.SceneFilterType{
			IsMissing: &isMissing,
		}

		q := getSceneStringValue(sceneIdxWithTwoTags, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		scenes := queryScene(t, sqb, &sceneFilter, &findFilter)

		assert.Len(t, scenes, 0)

		findFilter.Q = nil
		scenes = queryScene(t, sqb, &sceneFilter, &findFilter)

		assert.True(t, len(scenes) > 0)

		return nil
	})
}

func TestSceneQueryIsMissingRating(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		isMissing := "rating"
		sceneFilter := models.SceneFilterType{
			IsMissing: &isMissing,
		}

		scenes := queryScene(t, sqb, &sceneFilter, nil)

		assert.True(t, len(scenes) > 0)

		// ensure date is null, empty or "0001-01-01"
		for _, scene := range scenes {
			assert.True(t, !scene.Rating.Valid)
		}

		return nil
	})
}

func TestSceneQueryPerformers(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		performerCriterion := models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(performerIDs[performerIdxWithScene]),
				strconv.Itoa(performerIDs[performerIdx1WithScene]),
			},
			Modifier: models.CriterionModifierIncludes,
		}

		sceneFilter := models.SceneFilterType{
			Performers: &performerCriterion,
		}

		scenes := queryScene(t, sqb, &sceneFilter, nil)

		assert.Len(t, scenes, 2)

		// ensure ids are correct
		for _, scene := range scenes {
			assert.True(t, scene.ID == sceneIDs[sceneIdxWithPerformer] || scene.ID == sceneIDs[sceneIdxWithTwoPerformers])
		}

		performerCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(performerIDs[performerIdx1WithScene]),
				strconv.Itoa(performerIDs[performerIdx2WithScene]),
			},
			Modifier: models.CriterionModifierIncludesAll,
		}

		scenes = queryScene(t, sqb, &sceneFilter, nil)

		assert.Len(t, scenes, 1)
		assert.Equal(t, sceneIDs[sceneIdxWithTwoPerformers], scenes[0].ID)

		performerCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(performerIDs[performerIdx1WithScene]),
			},
			Modifier: models.CriterionModifierExcludes,
		}

		q := getSceneStringValue(sceneIdxWithTwoPerformers, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		scenes = queryScene(t, sqb, &sceneFilter, &findFilter)
		assert.Len(t, scenes, 0)

		return nil
	})
}

func TestSceneQueryTags(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		tagCriterion := models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdxWithScene]),
				strconv.Itoa(tagIDs[tagIdx1WithScene]),
			},
			Modifier: models.CriterionModifierIncludes,
		}

		sceneFilter := models.SceneFilterType{
			Tags: &tagCriterion,
		}

		scenes := queryScene(t, sqb, &sceneFilter, nil)
		assert.Len(t, scenes, 2)

		// ensure ids are correct
		for _, scene := range scenes {
			assert.True(t, scene.ID == sceneIDs[sceneIdxWithTag] || scene.ID == sceneIDs[sceneIdxWithTwoTags])
		}

		tagCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdx1WithScene]),
				strconv.Itoa(tagIDs[tagIdx2WithScene]),
			},
			Modifier: models.CriterionModifierIncludesAll,
		}

		scenes = queryScene(t, sqb, &sceneFilter, nil)

		assert.Len(t, scenes, 1)
		assert.Equal(t, sceneIDs[sceneIdxWithTwoTags], scenes[0].ID)

		tagCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdx1WithScene]),
			},
			Modifier: models.CriterionModifierExcludes,
		}

		q := getSceneStringValue(sceneIdxWithTwoTags, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		scenes = queryScene(t, sqb, &sceneFilter, &findFilter)
		assert.Len(t, scenes, 0)

		return nil
	})
}

func TestSceneQueryPerformerTags(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		tagCriterion := models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdxWithPerformer]),
				strconv.Itoa(tagIDs[tagIdx1WithPerformer]),
			},
			Modifier: models.CriterionModifierIncludes,
		}

		sceneFilter := models.SceneFilterType{
			PerformerTags: &tagCriterion,
		}

		scenes := queryScene(t, sqb, &sceneFilter, nil)
		assert.Len(t, scenes, 2)

		// ensure ids are correct
		for _, scene := range scenes {
			assert.True(t, scene.ID == sceneIDs[sceneIdxWithPerformerTag] || scene.ID == sceneIDs[sceneIdxWithPerformerTwoTags])
		}

		tagCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdx1WithPerformer]),
				strconv.Itoa(tagIDs[tagIdx2WithPerformer]),
			},
			Modifier: models.CriterionModifierIncludesAll,
		}

		scenes = queryScene(t, sqb, &sceneFilter, nil)

		assert.Len(t, scenes, 1)
		assert.Equal(t, sceneIDs[sceneIdxWithPerformerTwoTags], scenes[0].ID)

		tagCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdx1WithPerformer]),
			},
			Modifier: models.CriterionModifierExcludes,
		}

		q := getSceneStringValue(sceneIdxWithPerformerTwoTags, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		scenes = queryScene(t, sqb, &sceneFilter, &findFilter)
		assert.Len(t, scenes, 0)

		return nil
	})
}

func TestSceneQueryStudio(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		studioCriterion := models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithScene]),
			},
			Modifier: models.CriterionModifierIncludes,
			Depth:    0,
		}

		sceneFilter := models.SceneFilterType{
			Studios: &studioCriterion,
		}

		scenes := queryScene(t, sqb, &sceneFilter, nil)

		assert.Len(t, scenes, 1)

		// ensure id is correct
		assert.Equal(t, sceneIDs[sceneIdxWithStudio], scenes[0].ID)

		studioCriterion = models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithScene]),
			},
			Modifier: models.CriterionModifierExcludes,
			Depth:    0,
		}

		q := getSceneStringValue(sceneIdxWithStudio, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		scenes = queryScene(t, sqb, &sceneFilter, &findFilter)
		assert.Len(t, scenes, 0)

		return nil
	})
}

func TestSceneQueryStudioDepth(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		studioCriterion := models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithGrandChild]),
			},
			Modifier: models.CriterionModifierIncludes,
			Depth:    2,
		}

		sceneFilter := models.SceneFilterType{
			Studios: &studioCriterion,
		}

		scenes := queryScene(t, sqb, &sceneFilter, nil)
		assert.Len(t, scenes, 1)

		studioCriterion.Depth = 1

		scenes = queryScene(t, sqb, &sceneFilter, nil)
		assert.Len(t, scenes, 0)

		studioCriterion.Value = []string{strconv.Itoa(studioIDs[studioIdxWithParentAndChild])}
		scenes = queryScene(t, sqb, &sceneFilter, nil)
		assert.Len(t, scenes, 1)

		// ensure id is correct
		assert.Equal(t, sceneIDs[sceneIdxWithGrandChildStudio], scenes[0].ID)

		studioCriterion = models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithGrandChild]),
			},
			Modifier: models.CriterionModifierExcludes,
			Depth:    2,
		}

		q := getSceneStringValue(sceneIdxWithGrandChildStudio, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		scenes = queryScene(t, sqb, &sceneFilter, &findFilter)
		assert.Len(t, scenes, 0)

		studioCriterion.Depth = 1
		scenes = queryScene(t, sqb, &sceneFilter, &findFilter)
		assert.Len(t, scenes, 1)

		studioCriterion.Value = []string{strconv.Itoa(studioIDs[studioIdxWithParentAndChild])}
		scenes = queryScene(t, sqb, &sceneFilter, &findFilter)
		assert.Len(t, scenes, 0)

		return nil
	})
}

func TestSceneQueryMovies(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		movieCriterion := models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(movieIDs[movieIdxWithScene]),
			},
			Modifier: models.CriterionModifierIncludes,
		}

		sceneFilter := models.SceneFilterType{
			Movies: &movieCriterion,
		}

		scenes := queryScene(t, sqb, &sceneFilter, nil)

		assert.Len(t, scenes, 1)

		// ensure id is correct
		assert.Equal(t, sceneIDs[sceneIdxWithMovie], scenes[0].ID)

		movieCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(movieIDs[movieIdxWithScene]),
			},
			Modifier: models.CriterionModifierExcludes,
		}

		q := getSceneStringValue(sceneIdxWithMovie, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		scenes = queryScene(t, sqb, &sceneFilter, &findFilter)
		assert.Len(t, scenes, 0)

		return nil
	})
}

func TestSceneQuerySorting(t *testing.T) {
	sort := titleField
	direction := models.SortDirectionEnumAsc
	findFilter := models.FindFilterType{
		Sort:      &sort,
		Direction: &direction,
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		scenes := queryScene(t, sqb, nil, &findFilter)

		// scenes should be in same order as indexes
		firstScene := scenes[0]
		lastScene := scenes[len(scenes)-1]

		assert.Equal(t, sceneIDs[0], firstScene.ID)
		assert.Equal(t, sceneIDs[sceneIdxWithSpacedName], lastScene.ID)

		// sort in descending order
		direction = models.SortDirectionEnumDesc

		scenes = queryScene(t, sqb, nil, &findFilter)
		firstScene = scenes[0]
		lastScene = scenes[len(scenes)-1]

		assert.Equal(t, sceneIDs[sceneIdxWithSpacedName], firstScene.ID)
		assert.Equal(t, sceneIDs[0], lastScene.ID)

		return nil
	})
}

func TestSceneQueryPagination(t *testing.T) {
	perPage := 1
	findFilter := models.FindFilterType{
		PerPage: &perPage,
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		scenes := queryScene(t, sqb, nil, &findFilter)

		assert.Len(t, scenes, 1)

		firstID := scenes[0].ID

		page := 2
		findFilter.Page = &page
		scenes = queryScene(t, sqb, nil, &findFilter)

		assert.Len(t, scenes, 1)
		secondID := scenes[0].ID
		assert.NotEqual(t, firstID, secondID)

		perPage = 2
		page = 1

		scenes = queryScene(t, sqb, nil, &findFilter)
		assert.Len(t, scenes, 2)
		assert.Equal(t, firstID, scenes[0].ID)
		assert.Equal(t, secondID, scenes[1].ID)

		return nil
	})
}

func TestSceneQueryTagCount(t *testing.T) {
	const tagCount = 1
	tagCountCriterion := models.IntCriterionInput{
		Exact:    tagCount,
		Upper:    tagCount,
		Lower:    tagCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyScenesTagCount(t, tagCountCriterion)

	tagCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyScenesTagCount(t, tagCountCriterion)

	tagCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyScenesTagCount(t, tagCountCriterion)

	tagCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyScenesTagCount(t, tagCountCriterion)
}

func verifyScenesTagCount(t *testing.T, tagCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		sceneFilter := models.SceneFilterType{
			TagCount: &tagCountCriterion,
		}

		scenes := queryScene(t, sqb, &sceneFilter, nil)
		assert.Greater(t, len(scenes), 0)

		for _, scene := range scenes {
			ids, err := sqb.GetTagIDs(scene.ID)
			if err != nil {
				return err
			}
			verifyInt(t, len(ids), tagCountCriterion)
		}

		return nil
	})
}

func TestSceneQueryPerformerCount(t *testing.T) {
	const performerCount = 1
	performerCountCriterion := models.IntCriterionInput{
		Exact:    performerCount,
		Upper:    performerCount,
		Lower:    performerCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyScenesPerformerCount(t, performerCountCriterion)

	performerCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyScenesPerformerCount(t, performerCountCriterion)

	performerCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyScenesPerformerCount(t, performerCountCriterion)

	performerCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyScenesPerformerCount(t, performerCountCriterion)
}

func verifyScenesPerformerCount(t *testing.T, performerCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()
		sceneFilter := models.SceneFilterType{
			PerformerCount: &performerCountCriterion,
		}

		scenes := queryScene(t, sqb, &sceneFilter, nil)
		assert.Greater(t, len(scenes), 0)

		for _, scene := range scenes {
			ids, err := sqb.GetPerformerIDs(scene.ID)
			if err != nil {
				return err
			}
			verifyInt(t, len(ids), performerCountCriterion)
		}

		return nil
	})
}

func TestSceneCountByTagID(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()

		sceneCount, err := sqb.CountByTagID(tagIDs[tagIdxWithScene])

		if err != nil {
			t.Errorf("error calling CountByTagID: %s", err.Error())
		}

		assert.Equal(t, 1, sceneCount)

		sceneCount, err = sqb.CountByTagID(0)

		if err != nil {
			t.Errorf("error calling CountByTagID: %s", err.Error())
		}

		assert.Equal(t, 0, sceneCount)

		return nil
	})
}

func TestSceneCountByMovieID(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()

		sceneCount, err := sqb.CountByMovieID(movieIDs[movieIdxWithScene])

		if err != nil {
			t.Errorf("error calling CountByMovieID: %s", err.Error())
		}

		assert.Equal(t, 1, sceneCount)

		sceneCount, err = sqb.CountByMovieID(0)

		if err != nil {
			t.Errorf("error calling CountByMovieID: %s", err.Error())
		}

		assert.Equal(t, 0, sceneCount)

		return nil
	})
}

func TestSceneCountByStudioID(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()

		sceneCount, err := sqb.CountByStudioID(studioIDs[studioIdxWithScene])

		if err != nil {
			t.Errorf("error calling CountByStudioID: %s", err.Error())
		}

		assert.Equal(t, 1, sceneCount)

		sceneCount, err = sqb.CountByStudioID(0)

		if err != nil {
			t.Errorf("error calling CountByStudioID: %s", err.Error())
		}

		assert.Equal(t, 0, sceneCount)

		return nil
	})
}

func TestFindByMovieID(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()

		scenes, err := sqb.FindByMovieID(movieIDs[movieIdxWithScene])

		if err != nil {
			t.Errorf("error calling FindByMovieID: %s", err.Error())
		}

		assert.Len(t, scenes, 1)
		assert.Equal(t, sceneIDs[sceneIdxWithMovie], scenes[0].ID)

		scenes, err = sqb.FindByMovieID(0)

		if err != nil {
			t.Errorf("error calling FindByMovieID: %s", err.Error())
		}

		assert.Len(t, scenes, 0)

		return nil
	})
}

func TestFindByPerformerID(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Scene()

		scenes, err := sqb.FindByPerformerID(performerIDs[performerIdxWithScene])

		if err != nil {
			t.Errorf("error calling FindByPerformerID: %s", err.Error())
		}

		assert.Len(t, scenes, 1)
		assert.Equal(t, sceneIDs[sceneIdxWithPerformer], scenes[0].ID)

		scenes, err = sqb.FindByPerformerID(0)

		if err != nil {
			t.Errorf("error calling FindByPerformerID: %s", err.Error())
		}

		assert.Len(t, scenes, 0)

		return nil
	})
}

func TestSceneUpdateSceneCover(t *testing.T) {
	if err := withTxn(func(r models.Repository) error {
		qb := r.Scene()

		// create performer to test against
		const name = "TestSceneUpdateSceneCover"
		scene := models.Scene{
			Path:     name,
			Checksum: sql.NullString{String: utils.MD5FromString(name), Valid: true},
		}
		created, err := qb.Create(scene)
		if err != nil {
			return fmt.Errorf("Error creating scene: %s", err.Error())
		}

		image := []byte("image")
		err = qb.UpdateCover(created.ID, image)
		if err != nil {
			return fmt.Errorf("Error updating scene cover: %s", err.Error())
		}

		// ensure image set
		storedImage, err := qb.GetCover(created.ID)
		if err != nil {
			return fmt.Errorf("Error getting image: %s", err.Error())
		}
		assert.Equal(t, storedImage, image)

		// set nil image
		err = qb.UpdateCover(created.ID, nil)
		if err == nil {
			return fmt.Errorf("Expected error setting nil image")
		}

		return nil
	}); err != nil {
		t.Error(err.Error())
	}
}

func TestSceneDestroySceneCover(t *testing.T) {
	if err := withTxn(func(r models.Repository) error {
		qb := r.Scene()

		// create performer to test against
		const name = "TestSceneDestroySceneCover"
		scene := models.Scene{
			Path:     name,
			Checksum: sql.NullString{String: utils.MD5FromString(name), Valid: true},
		}
		created, err := qb.Create(scene)
		if err != nil {
			return fmt.Errorf("Error creating scene: %s", err.Error())
		}

		image := []byte("image")
		err = qb.UpdateCover(created.ID, image)
		if err != nil {
			return fmt.Errorf("Error updating scene image: %s", err.Error())
		}

		err = qb.DestroyCover(created.ID)
		if err != nil {
			return fmt.Errorf("Error destroying scene cover: %s", err.Error())
		}

		// image should be nil
		storedImage, err := qb.GetCover(created.ID)
		if err != nil {
			return fmt.Errorf("Error getting image: %s", err.Error())
		}
		assert.Nil(t, storedImage)

		return nil
	}); err != nil {
		t.Error(err.Error())
	}
}

func TestSceneStashIDs(t *testing.T) {
	if err := withTxn(func(r models.Repository) error {
		qb := r.Scene()

		// create scene to test against
		const name = "TestSceneStashIDs"
		scene := models.Scene{
			Path:     name,
			Checksum: sql.NullString{String: utils.MD5FromString(name), Valid: true},
		}
		created, err := qb.Create(scene)
		if err != nil {
			return fmt.Errorf("Error creating scene: %s", err.Error())
		}

		testStashIDReaderWriter(t, qb, created.ID)
		return nil
	}); err != nil {
		t.Error(err.Error())
	}
}

func TestSceneQueryQTrim(t *testing.T) {
	if err := withTxn(func(r models.Repository) error {
		qb := r.Scene()

		expectedID := sceneIDs[sceneIdxWithSpacedName]

		type test struct {
			query string
			id    int
			count int
		}
		tests := []test{
			{query: " zzz    yyy    ", id: expectedID, count: 1},
			{query: "   \"zzz yyy xxx\" ", id: expectedID, count: 1},
			{query: "zzz", id: expectedID, count: 1},
			{query: "\" zzz    yyy    \"", count: 0},
			{query: "\"zzz    yyy\"", count: 0},
			{query: "\" zzz yyy\"", count: 0},
			{query: "\"zzz yyy  \"", count: 0},
		}

		for _, tst := range tests {
			f := models.FindFilterType{
				Q: &tst.query,
			}
			scenes := queryScene(t, qb, nil, &f)

			assert.Len(t, scenes, tst.count)
			if len(scenes) > 0 {
				assert.Equal(t, tst.id, scenes[0].ID)
			}
		}

		findFilter := models.FindFilterType{}
		scenes := queryScene(t, qb, nil, &findFilter)
		assert.NotEqual(t, 0, len(scenes))

		return nil
	}); err != nil {
		t.Error(err.Error())
	}
}

// TODO Update
// TODO IncrementOCounter
// TODO DecrementOCounter
// TODO ResetOCounter
// TODO Destroy
// TODO FindByChecksum
// TODO Count
// TODO SizeCount
// TODO All

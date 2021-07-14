// +build integration

package sqlite_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stashapp/stash/pkg/models"
)

func TestGalleryFind(t *testing.T) {
	withTxn(func(r models.Repository) error {
		gqb := r.Gallery()

		const galleryIdx = 0
		gallery, err := gqb.Find(galleryIDs[galleryIdx])

		if err != nil {
			t.Errorf("Error finding gallery: %s", err.Error())
		}

		assert.Equal(t, getGalleryStringValue(galleryIdx, "Path"), gallery.Path.String)

		gallery, err = gqb.Find(0)

		if err != nil {
			t.Errorf("Error finding gallery: %s", err.Error())
		}

		assert.Nil(t, gallery)

		return nil
	})
}

func TestGalleryFindByChecksum(t *testing.T) {
	withTxn(func(r models.Repository) error {
		gqb := r.Gallery()

		const galleryIdx = 0
		galleryChecksum := getGalleryStringValue(galleryIdx, "Checksum")
		gallery, err := gqb.FindByChecksum(galleryChecksum)

		if err != nil {
			t.Errorf("Error finding gallery: %s", err.Error())
		}

		assert.Equal(t, getGalleryStringValue(galleryIdx, "Path"), gallery.Path.String)

		galleryChecksum = "not exist"
		gallery, err = gqb.FindByChecksum(galleryChecksum)

		if err != nil {
			t.Errorf("Error finding gallery: %s", err.Error())
		}

		assert.Nil(t, gallery)

		return nil
	})
}

func TestGalleryFindByPath(t *testing.T) {
	withTxn(func(r models.Repository) error {
		gqb := r.Gallery()

		const galleryIdx = 0
		galleryPath := getGalleryStringValue(galleryIdx, "Path")
		gallery, err := gqb.FindByPath(galleryPath)

		if err != nil {
			t.Errorf("Error finding gallery: %s", err.Error())
		}

		assert.Equal(t, galleryPath, gallery.Path.String)

		galleryPath = "not exist"
		gallery, err = gqb.FindByPath(galleryPath)

		if err != nil {
			t.Errorf("Error finding gallery: %s", err.Error())
		}

		assert.Nil(t, gallery)

		return nil
	})
}

func TestGalleryFindBySceneID(t *testing.T) {
	withTxn(func(r models.Repository) error {
		gqb := r.Gallery()

		sceneID := sceneIDs[sceneIdxWithGallery]
		galleries, err := gqb.FindBySceneID(sceneID)

		if err != nil {
			t.Errorf("Error finding gallery: %s", err.Error())
		}

		assert.Equal(t, getGalleryStringValue(galleryIdxWithScene, "Path"), galleries[0].Path.String)

		galleries, err = gqb.FindBySceneID(0)

		if err != nil {
			t.Errorf("Error finding gallery: %s", err.Error())
		}

		assert.Nil(t, galleries)

		return nil
	})
}

func TestGalleryQueryQ(t *testing.T) {
	withTxn(func(r models.Repository) error {
		const galleryIdx = 0

		q := getGalleryStringValue(galleryIdx, pathField)

		sqb := r.Gallery()

		galleryQueryQ(t, sqb, q, galleryIdx)

		return nil
	})
}

func galleryQueryQ(t *testing.T, qb models.GalleryReader, q string, expectedGalleryIdx int) {
	filter := models.FindFilterType{
		Q: &q,
	}
	galleries, _, err := qb.Query(nil, &filter)
	if err != nil {
		t.Errorf("Error querying gallery: %s", err.Error())
	}

	assert.Len(t, galleries, 1)
	gallery := galleries[0]
	assert.Equal(t, galleryIDs[expectedGalleryIdx], gallery.ID)

	// no Q should return all results
	filter.Q = nil
	galleries, _, err = qb.Query(nil, &filter)
	if err != nil {
		t.Errorf("Error querying gallery: %s", err.Error())
	}

	assert.Len(t, galleries, totalGalleries)
}

func TestGalleryQueryPath(t *testing.T) {
	withTxn(func(r models.Repository) error {
		const galleryIdx = 1
		galleryPath := getGalleryStringValue(galleryIdx, "Path")

		pathCriterion := models.StringCriterionInput{
			Value:    galleryPath,
			Modifier: models.CriterionModifierEquals,
		}

		verifyGalleriesPath(t, r.Gallery(), pathCriterion)

		pathCriterion.Modifier = models.CriterionModifierNotEquals
		verifyGalleriesPath(t, r.Gallery(), pathCriterion)

		pathCriterion.Modifier = models.CriterionModifierMatchesRegex
		pathCriterion.Value = "gallery.*1_Path"
		verifyGalleriesPath(t, r.Gallery(), pathCriterion)

		pathCriterion.Modifier = models.CriterionModifierNotMatchesRegex
		verifyGalleriesPath(t, r.Gallery(), pathCriterion)

		return nil
	})
}

func verifyGalleriesPath(t *testing.T, sqb models.GalleryReader, pathCriterion models.StringCriterionInput) {
	galleryFilter := models.GalleryFilterType{
		Path: &pathCriterion,
	}

	galleries, _, err := sqb.Query(&galleryFilter, nil)
	if err != nil {
		t.Errorf("Error querying gallery: %s", err.Error())
	}

	for _, gallery := range galleries {
		verifyNullString(t, gallery.Path, pathCriterion)
	}
}

func TestGalleryQueryPathOr(t *testing.T) {
	const gallery1Idx = 1
	const gallery2Idx = 2

	gallery1Path := getGalleryStringValue(gallery1Idx, "Path")
	gallery2Path := getGalleryStringValue(gallery2Idx, "Path")

	galleryFilter := models.GalleryFilterType{
		Path: &models.StringCriterionInput{
			Value:    gallery1Path,
			Modifier: models.CriterionModifierEquals,
		},
		Or: &models.GalleryFilterType{
			Path: &models.StringCriterionInput{
				Value:    gallery2Path,
				Modifier: models.CriterionModifierEquals,
			},
		},
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()

		galleries := queryGallery(t, sqb, &galleryFilter, nil)

		assert.Len(t, galleries, 2)
		assert.Equal(t, gallery1Path, galleries[0].Path.String)
		assert.Equal(t, gallery2Path, galleries[1].Path.String)

		return nil
	})
}

func TestGalleryQueryPathAndRating(t *testing.T) {
	const galleryIdx = 1
	galleryPath := getGalleryStringValue(galleryIdx, "Path")
	galleryRating := getRating(galleryIdx)

	galleryFilter := models.GalleryFilterType{
		Path: &models.StringCriterionInput{
			Value:    galleryPath,
			Modifier: models.CriterionModifierEquals,
		},
		And: &models.GalleryFilterType{
			Rating: &models.IntCriterionInput{
				Value:    int(galleryRating.Int64),
				Modifier: models.CriterionModifierEquals,
			},
		},
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()

		galleries := queryGallery(t, sqb, &galleryFilter, nil)

		assert.Len(t, galleries, 1)
		assert.Equal(t, galleryPath, galleries[0].Path.String)
		assert.Equal(t, galleryRating.Int64, galleries[0].Rating.Int64)

		return nil
	})
}

func TestGalleryQueryPathNotRating(t *testing.T) {
	const galleryIdx = 1

	galleryRating := getRating(galleryIdx)

	pathCriterion := models.StringCriterionInput{
		Value:    "gallery_.*1_Path",
		Modifier: models.CriterionModifierMatchesRegex,
	}

	ratingCriterion := models.IntCriterionInput{
		Value:    int(galleryRating.Int64),
		Modifier: models.CriterionModifierEquals,
	}

	galleryFilter := models.GalleryFilterType{
		Path: &pathCriterion,
		Not: &models.GalleryFilterType{
			Rating: &ratingCriterion,
		},
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()

		galleries := queryGallery(t, sqb, &galleryFilter, nil)

		for _, gallery := range galleries {
			verifyNullString(t, gallery.Path, pathCriterion)
			ratingCriterion.Modifier = models.CriterionModifierNotEquals
			verifyInt64(t, gallery.Rating, ratingCriterion)
		}

		return nil
	})
}

func TestGalleryIllegalQuery(t *testing.T) {
	assert := assert.New(t)

	const galleryIdx = 1
	subFilter := models.GalleryFilterType{
		Path: &models.StringCriterionInput{
			Value:    getGalleryStringValue(galleryIdx, "Path"),
			Modifier: models.CriterionModifierEquals,
		},
	}

	galleryFilter := &models.GalleryFilterType{
		And: &subFilter,
		Or:  &subFilter,
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()

		_, _, err := sqb.Query(galleryFilter, nil)
		assert.NotNil(err)

		galleryFilter.Or = nil
		galleryFilter.Not = &subFilter
		_, _, err = sqb.Query(galleryFilter, nil)
		assert.NotNil(err)

		galleryFilter.And = nil
		galleryFilter.Or = &subFilter
		_, _, err = sqb.Query(galleryFilter, nil)
		assert.NotNil(err)

		return nil
	})
}

func TestGalleryQueryURL(t *testing.T) {
	const sceneIdx = 1
	galleryURL := getGalleryStringValue(sceneIdx, urlField)

	urlCriterion := models.StringCriterionInput{
		Value:    galleryURL,
		Modifier: models.CriterionModifierEquals,
	}

	filter := models.GalleryFilterType{
		URL: &urlCriterion,
	}

	verifyFn := func(g *models.Gallery) {
		t.Helper()
		verifyNullString(t, g.URL, urlCriterion)
	}

	verifyGalleryQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierNotEquals
	verifyGalleryQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierMatchesRegex
	urlCriterion.Value = "gallery_.*1_URL"
	verifyGalleryQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierNotMatchesRegex
	verifyGalleryQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierIsNull
	urlCriterion.Value = ""
	verifyGalleryQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierNotNull
	verifyGalleryQuery(t, filter, verifyFn)
}

func verifyGalleryQuery(t *testing.T, filter models.GalleryFilterType, verifyFn func(s *models.Gallery)) {
	withTxn(func(r models.Repository) error {
		t.Helper()
		sqb := r.Gallery()

		galleries := queryGallery(t, sqb, &filter, nil)

		// assume it should find at least one
		assert.Greater(t, len(galleries), 0)

		for _, gallery := range galleries {
			verifyFn(gallery)
		}

		return nil
	})
}

func TestGalleryQueryRating(t *testing.T) {
	const rating = 3
	ratingCriterion := models.IntCriterionInput{
		Value:    rating,
		Modifier: models.CriterionModifierEquals,
	}

	verifyGalleriesRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierNotEquals
	verifyGalleriesRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyGalleriesRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierLessThan
	verifyGalleriesRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierIsNull
	verifyGalleriesRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierNotNull
	verifyGalleriesRating(t, ratingCriterion)
}

func verifyGalleriesRating(t *testing.T, ratingCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()
		galleryFilter := models.GalleryFilterType{
			Rating: &ratingCriterion,
		}

		galleries, _, err := sqb.Query(&galleryFilter, nil)
		if err != nil {
			t.Errorf("Error querying gallery: %s", err.Error())
		}

		for _, gallery := range galleries {
			verifyInt64(t, gallery.Rating, ratingCriterion)
		}

		return nil
	})
}

func TestGalleryQueryIsMissingScene(t *testing.T) {
	withTxn(func(r models.Repository) error {
		qb := r.Gallery()
		isMissing := "scenes"
		galleryFilter := models.GalleryFilterType{
			IsMissing: &isMissing,
		}

		q := getGalleryStringValue(galleryIdxWithScene, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		galleries, _, err := qb.Query(&galleryFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying gallery: %s", err.Error())
		}

		assert.Len(t, galleries, 0)

		findFilter.Q = nil
		galleries, _, err = qb.Query(&galleryFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying gallery: %s", err.Error())
		}

		// ensure non of the ids equal the one with gallery
		for _, gallery := range galleries {
			assert.NotEqual(t, galleryIDs[galleryIdxWithScene], gallery.ID)
		}

		return nil
	})
}

func queryGallery(t *testing.T, sqb models.GalleryReader, galleryFilter *models.GalleryFilterType, findFilter *models.FindFilterType) []*models.Gallery {
	galleries, _, err := sqb.Query(galleryFilter, findFilter)
	if err != nil {
		t.Errorf("Error querying gallery: %s", err.Error())
	}

	return galleries
}

func TestGalleryQueryIsMissingStudio(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()
		isMissing := "studio"
		galleryFilter := models.GalleryFilterType{
			IsMissing: &isMissing,
		}

		q := getGalleryStringValue(galleryIdxWithStudio, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		galleries := queryGallery(t, sqb, &galleryFilter, &findFilter)

		assert.Len(t, galleries, 0)

		findFilter.Q = nil
		galleries = queryGallery(t, sqb, &galleryFilter, &findFilter)

		// ensure non of the ids equal the one with studio
		for _, gallery := range galleries {
			assert.NotEqual(t, galleryIDs[galleryIdxWithStudio], gallery.ID)
		}

		return nil
	})
}

func TestGalleryQueryIsMissingPerformers(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()
		isMissing := "performers"
		galleryFilter := models.GalleryFilterType{
			IsMissing: &isMissing,
		}

		q := getGalleryStringValue(galleryIdxWithPerformer, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		galleries := queryGallery(t, sqb, &galleryFilter, &findFilter)

		assert.Len(t, galleries, 0)

		findFilter.Q = nil
		galleries = queryGallery(t, sqb, &galleryFilter, &findFilter)

		assert.True(t, len(galleries) > 0)

		// ensure non of the ids equal the one with movies
		for _, gallery := range galleries {
			assert.NotEqual(t, galleryIDs[galleryIdxWithPerformer], gallery.ID)
		}

		return nil
	})
}

func TestGalleryQueryIsMissingTags(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()
		isMissing := "tags"
		galleryFilter := models.GalleryFilterType{
			IsMissing: &isMissing,
		}

		q := getGalleryStringValue(galleryIdxWithTwoTags, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		galleries := queryGallery(t, sqb, &galleryFilter, &findFilter)

		assert.Len(t, galleries, 0)

		findFilter.Q = nil
		galleries = queryGallery(t, sqb, &galleryFilter, &findFilter)

		assert.True(t, len(galleries) > 0)

		return nil
	})
}

func TestGalleryQueryPerformers(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()
		performerCriterion := models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(performerIDs[performerIdxWithGallery]),
				strconv.Itoa(performerIDs[performerIdx1WithGallery]),
			},
			Modifier: models.CriterionModifierIncludes,
		}

		galleryFilter := models.GalleryFilterType{
			Performers: &performerCriterion,
		}

		galleries := queryGallery(t, sqb, &galleryFilter, nil)

		assert.Len(t, galleries, 2)

		// ensure ids are correct
		for _, gallery := range galleries {
			assert.True(t, gallery.ID == galleryIDs[galleryIdxWithPerformer] || gallery.ID == galleryIDs[galleryIdxWithTwoPerformers])
		}

		performerCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(performerIDs[performerIdx1WithGallery]),
				strconv.Itoa(performerIDs[performerIdx2WithGallery]),
			},
			Modifier: models.CriterionModifierIncludesAll,
		}

		galleries = queryGallery(t, sqb, &galleryFilter, nil)

		assert.Len(t, galleries, 1)
		assert.Equal(t, galleryIDs[galleryIdxWithTwoPerformers], galleries[0].ID)

		performerCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(performerIDs[performerIdx1WithGallery]),
			},
			Modifier: models.CriterionModifierExcludes,
		}

		q := getGalleryStringValue(galleryIdxWithTwoPerformers, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		galleries = queryGallery(t, sqb, &galleryFilter, &findFilter)
		assert.Len(t, galleries, 0)

		return nil
	})
}

func TestGalleryQueryTags(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()
		tagCriterion := models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdxWithGallery]),
				strconv.Itoa(tagIDs[tagIdx1WithGallery]),
			},
			Modifier: models.CriterionModifierIncludes,
			Depth:    0,
		}

		galleryFilter := models.GalleryFilterType{
			Tags: &tagCriterion,
		}

		galleries := queryGallery(t, sqb, &galleryFilter, nil)
		assert.Len(t, galleries, 2)

		// ensure ids are correct
		for _, gallery := range galleries {
			assert.True(t, gallery.ID == galleryIDs[galleryIdxWithTag] || gallery.ID == galleryIDs[galleryIdxWithTwoTags])
		}

		tagCriterion = models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdx1WithGallery]),
				strconv.Itoa(tagIDs[tagIdx2WithGallery]),
			},
			Modifier: models.CriterionModifierIncludesAll,
			Depth:    0,
		}

		galleries = queryGallery(t, sqb, &galleryFilter, nil)

		assert.Len(t, galleries, 1)
		assert.Equal(t, galleryIDs[galleryIdxWithTwoTags], galleries[0].ID)

		tagCriterion = models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdx1WithGallery]),
			},
			Modifier: models.CriterionModifierExcludes,
			Depth:    0,
		}

		q := getGalleryStringValue(galleryIdxWithTwoTags, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		galleries = queryGallery(t, sqb, &galleryFilter, &findFilter)
		assert.Len(t, galleries, 0)

		return nil
	})
}

func TestGalleryQueryStudio(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()
		studioCriterion := models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithGallery]),
			},
			Modifier: models.CriterionModifierIncludes,
			Depth:    0,
		}

		galleryFilter := models.GalleryFilterType{
			Studios: &studioCriterion,
		}

		galleries := queryGallery(t, sqb, &galleryFilter, nil)

		assert.Len(t, galleries, 1)

		// ensure id is correct
		assert.Equal(t, galleryIDs[galleryIdxWithStudio], galleries[0].ID)

		studioCriterion = models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithGallery]),
			},
			Modifier: models.CriterionModifierExcludes,
			Depth:    0,
		}

		q := getGalleryStringValue(galleryIdxWithStudio, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		galleries = queryGallery(t, sqb, &galleryFilter, &findFilter)
		assert.Len(t, galleries, 0)

		return nil
	})
}

func TestGalleryQueryStudioDepth(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()
		studioCriterion := models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithGrandChild]),
			},
			Modifier: models.CriterionModifierIncludes,
			Depth:    2,
		}

		galleryFilter := models.GalleryFilterType{
			Studios: &studioCriterion,
		}

		galleries := queryGallery(t, sqb, &galleryFilter, nil)
		assert.Len(t, galleries, 1)

		studioCriterion.Depth = 1

		galleries = queryGallery(t, sqb, &galleryFilter, nil)
		assert.Len(t, galleries, 0)

		studioCriterion.Value = []string{strconv.Itoa(studioIDs[studioIdxWithParentAndChild])}
		galleries = queryGallery(t, sqb, &galleryFilter, nil)
		assert.Len(t, galleries, 1)

		// ensure id is correct
		assert.Equal(t, galleryIDs[galleryIdxWithGrandChildStudio], galleries[0].ID)

		studioCriterion = models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithGrandChild]),
			},
			Modifier: models.CriterionModifierExcludes,
			Depth:    2,
		}

		q := getGalleryStringValue(galleryIdxWithGrandChildStudio, pathField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		galleries = queryGallery(t, sqb, &galleryFilter, &findFilter)
		assert.Len(t, galleries, 0)

		studioCriterion.Depth = 1
		galleries = queryGallery(t, sqb, &galleryFilter, &findFilter)
		assert.Len(t, galleries, 1)

		studioCriterion.Value = []string{strconv.Itoa(studioIDs[studioIdxWithParentAndChild])}
		galleries = queryGallery(t, sqb, &galleryFilter, &findFilter)
		assert.Len(t, galleries, 0)

		return nil
	})
}

func TestGalleryQueryPerformerTags(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()
		tagCriterion := models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdxWithPerformer]),
				strconv.Itoa(tagIDs[tagIdx1WithPerformer]),
			},
			Modifier: models.CriterionModifierIncludes,
			Depth:    0,
		}

		galleryFilter := models.GalleryFilterType{
			PerformerTags: &tagCriterion,
		}

		galleries := queryGallery(t, sqb, &galleryFilter, nil)
		assert.Len(t, galleries, 2)

		// ensure ids are correct
		for _, gallery := range galleries {
			assert.True(t, gallery.ID == galleryIDs[galleryIdxWithPerformerTag] || gallery.ID == galleryIDs[galleryIdxWithPerformerTwoTags])
		}

		tagCriterion = models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdx1WithPerformer]),
				strconv.Itoa(tagIDs[tagIdx2WithPerformer]),
			},
			Modifier: models.CriterionModifierIncludesAll,
			Depth:    0,
		}

		galleries = queryGallery(t, sqb, &galleryFilter, nil)

		assert.Len(t, galleries, 1)
		assert.Equal(t, galleryIDs[galleryIdxWithPerformerTwoTags], galleries[0].ID)

		tagCriterion = models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdx1WithPerformer]),
			},
			Modifier: models.CriterionModifierExcludes,
			Depth:    0,
		}

		q := getGalleryStringValue(galleryIdxWithPerformerTwoTags, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		galleries = queryGallery(t, sqb, &galleryFilter, &findFilter)
		assert.Len(t, galleries, 0)

		return nil
	})
}

func TestGalleryQueryTagCount(t *testing.T) {
	const tagCount = 1
	tagCountCriterion := models.IntCriterionInput{
		Value:    tagCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyGalleriesTagCount(t, tagCountCriterion)

	tagCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyGalleriesTagCount(t, tagCountCriterion)

	tagCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyGalleriesTagCount(t, tagCountCriterion)

	tagCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyGalleriesTagCount(t, tagCountCriterion)
}

func verifyGalleriesTagCount(t *testing.T, tagCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()
		galleryFilter := models.GalleryFilterType{
			TagCount: &tagCountCriterion,
		}

		galleries := queryGallery(t, sqb, &galleryFilter, nil)
		assert.Greater(t, len(galleries), 0)

		for _, gallery := range galleries {
			ids, err := sqb.GetTagIDs(gallery.ID)
			if err != nil {
				return err
			}
			verifyInt(t, len(ids), tagCountCriterion)
		}

		return nil
	})
}

func TestGalleryQueryPerformerCount(t *testing.T) {
	const performerCount = 1
	performerCountCriterion := models.IntCriterionInput{
		Value:    performerCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyGalleriesPerformerCount(t, performerCountCriterion)

	performerCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyGalleriesPerformerCount(t, performerCountCriterion)

	performerCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyGalleriesPerformerCount(t, performerCountCriterion)

	performerCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyGalleriesPerformerCount(t, performerCountCriterion)
}

func verifyGalleriesPerformerCount(t *testing.T, performerCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()
		galleryFilter := models.GalleryFilterType{
			PerformerCount: &performerCountCriterion,
		}

		galleries := queryGallery(t, sqb, &galleryFilter, nil)
		assert.Greater(t, len(galleries), 0)

		for _, gallery := range galleries {
			ids, err := sqb.GetPerformerIDs(gallery.ID)
			if err != nil {
				return err
			}
			verifyInt(t, len(ids), performerCountCriterion)
		}

		return nil
	})
}

func TestGalleryQueryAverageResolution(t *testing.T) {
	withTxn(func(r models.Repository) error {
		qb := r.Gallery()
		resolution := models.ResolutionEnumLow
		galleryFilter := models.GalleryFilterType{
			AverageResolution: &resolution,
		}

		// not verifying average - just ensure we get at least one
		galleries := queryGallery(t, qb, &galleryFilter, nil)
		assert.Greater(t, len(galleries), 0)

		return nil
	})
}

func TestGalleryQueryImageCount(t *testing.T) {
	const imageCount = 0
	imageCountCriterion := models.IntCriterionInput{
		Value:    imageCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyGalleriesImageCount(t, imageCountCriterion)

	imageCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyGalleriesImageCount(t, imageCountCriterion)

	imageCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyGalleriesImageCount(t, imageCountCriterion)

	imageCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyGalleriesImageCount(t, imageCountCriterion)
}

func verifyGalleriesImageCount(t *testing.T, imageCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Gallery()
		galleryFilter := models.GalleryFilterType{
			ImageCount: &imageCountCriterion,
		}

		galleries := queryGallery(t, sqb, &galleryFilter, nil)
		assert.Greater(t, len(galleries), -1)

		for _, gallery := range galleries {
			pp := 0

			_, count, err := r.Image().Query(&models.ImageFilterType{
				Galleries: &models.MultiCriterionInput{
					Value:    []string{strconv.Itoa(gallery.ID)},
					Modifier: models.CriterionModifierIncludes,
				},
			}, &models.FindFilterType{
				PerPage: &pp,
			})
			if err != nil {
				return err
			}
			verifyInt(t, count, imageCountCriterion)
		}

		return nil
	})
}

// TODO Count
// TODO All
// TODO Query
// TODO Update
// TODO Destroy

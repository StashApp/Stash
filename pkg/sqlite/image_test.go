// +build integration

package sqlite_test

import (
	"database/sql"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stashapp/stash/pkg/models"
)

func TestImageFind(t *testing.T) {
	withTxn(func(r models.Repository) error {
		// assume that the first image is imageWithGalleryPath
		sqb := r.Image()

		const imageIdx = 0
		imageID := imageIDs[imageIdx]
		image, err := sqb.Find(imageID)

		if err != nil {
			t.Errorf("Error finding image: %s", err.Error())
		}

		assert.Equal(t, getImageStringValue(imageIdx, "Path"), image.Path)

		imageID = 0
		image, err = sqb.Find(imageID)

		if err != nil {
			t.Errorf("Error finding image: %s", err.Error())
		}

		assert.Nil(t, image)

		return nil
	})
}

func TestImageFindByPath(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()

		const imageIdx = 1
		imagePath := getImageStringValue(imageIdx, "Path")
		image, err := sqb.FindByPath(imagePath)

		if err != nil {
			t.Errorf("Error finding image: %s", err.Error())
		}

		assert.Equal(t, imageIDs[imageIdx], image.ID)
		assert.Equal(t, imagePath, image.Path)

		imagePath = "not exist"
		image, err = sqb.FindByPath(imagePath)

		if err != nil {
			t.Errorf("Error finding image: %s", err.Error())
		}

		assert.Nil(t, image)

		return nil
	})
}

func TestImageQueryQ(t *testing.T) {
	withTxn(func(r models.Repository) error {
		const imageIdx = 2

		q := getImageStringValue(imageIdx, titleField)

		sqb := r.Image()

		imageQueryQ(t, sqb, q, imageIdx)

		return nil
	})
}

func imageQueryQ(t *testing.T, sqb models.ImageReader, q string, expectedImageIdx int) {
	filter := models.FindFilterType{
		Q: &q,
	}
	images, _, err := sqb.Query(nil, &filter)
	if err != nil {
		t.Errorf("Error querying image: %s", err.Error())
	}

	assert.Len(t, images, 1)
	image := images[0]
	assert.Equal(t, imageIDs[expectedImageIdx], image.ID)

	count, err := sqb.QueryCount(nil, &filter)
	if err != nil {
		t.Errorf("Error querying image: %s", err.Error())
	}
	assert.Equal(t, len(images), count)

	// no Q should return all results
	filter.Q = nil
	images, _, err = sqb.Query(nil, &filter)
	if err != nil {
		t.Errorf("Error querying image: %s", err.Error())
	}

	assert.Len(t, images, totalImages)
}

func TestImageQueryPath(t *testing.T) {
	const imageIdx = 1
	imagePath := getImageStringValue(imageIdx, "Path")

	pathCriterion := models.StringCriterionInput{
		Value:    imagePath,
		Modifier: models.CriterionModifierEquals,
	}

	verifyImagePath(t, pathCriterion, 1)

	pathCriterion.Modifier = models.CriterionModifierNotEquals
	verifyImagePath(t, pathCriterion, totalImages-1)

	pathCriterion.Modifier = models.CriterionModifierMatchesRegex
	pathCriterion.Value = "image_.*01_Path"
	verifyImagePath(t, pathCriterion, 1) // TODO - 2 if zip path is included

	pathCriterion.Modifier = models.CriterionModifierNotMatchesRegex
	verifyImagePath(t, pathCriterion, totalImages-1) // TODO - -2 if zip path is included
}

func verifyImagePath(t *testing.T, pathCriterion models.StringCriterionInput, expected int) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		imageFilter := models.ImageFilterType{
			Path: &pathCriterion,
		}

		images, _, err := sqb.Query(&imageFilter, nil)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.Equal(t, expected, len(images), "number of returned images")

		for _, image := range images {
			verifyString(t, image.Path, pathCriterion)
		}

		return nil
	})
}

func TestImageQueryPathOr(t *testing.T) {
	const image1Idx = 1
	const image2Idx = 2

	image1Path := getImageStringValue(image1Idx, "Path")
	image2Path := getImageStringValue(image2Idx, "Path")

	imageFilter := models.ImageFilterType{
		Path: &models.StringCriterionInput{
			Value:    image1Path,
			Modifier: models.CriterionModifierEquals,
		},
		Or: &models.ImageFilterType{
			Path: &models.StringCriterionInput{
				Value:    image2Path,
				Modifier: models.CriterionModifierEquals,
			},
		},
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Image()

		images := queryImages(t, sqb, &imageFilter, nil)

		assert.Len(t, images, 2)
		assert.Equal(t, image1Path, images[0].Path)
		assert.Equal(t, image2Path, images[1].Path)

		return nil
	})
}

func TestImageQueryPathAndRating(t *testing.T) {
	const imageIdx = 1
	imagePath := getImageStringValue(imageIdx, "Path")
	imageRating := getRating(imageIdx)

	imageFilter := models.ImageFilterType{
		Path: &models.StringCriterionInput{
			Value:    imagePath,
			Modifier: models.CriterionModifierEquals,
		},
		And: &models.ImageFilterType{
			Rating: &models.IntCriterionInput{
				Value:    int(imageRating.Int64),
				Modifier: models.CriterionModifierEquals,
			},
		},
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Image()

		images := queryImages(t, sqb, &imageFilter, nil)

		assert.Len(t, images, 1)
		assert.Equal(t, imagePath, images[0].Path)
		assert.Equal(t, imageRating.Int64, images[0].Rating.Int64)

		return nil
	})
}

func TestImageQueryPathNotRating(t *testing.T) {
	const imageIdx = 1

	imageRating := getRating(imageIdx)

	pathCriterion := models.StringCriterionInput{
		Value:    "image_.*1_Path",
		Modifier: models.CriterionModifierMatchesRegex,
	}

	ratingCriterion := models.IntCriterionInput{
		Value:    int(imageRating.Int64),
		Modifier: models.CriterionModifierEquals,
	}

	imageFilter := models.ImageFilterType{
		Path: &pathCriterion,
		Not: &models.ImageFilterType{
			Rating: &ratingCriterion,
		},
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Image()

		images := queryImages(t, sqb, &imageFilter, nil)

		for _, image := range images {
			verifyString(t, image.Path, pathCriterion)
			ratingCriterion.Modifier = models.CriterionModifierNotEquals
			verifyInt64(t, image.Rating, ratingCriterion)
		}

		return nil
	})
}

func TestImageIllegalQuery(t *testing.T) {
	assert := assert.New(t)

	const imageIdx = 1
	subFilter := models.ImageFilterType{
		Path: &models.StringCriterionInput{
			Value:    getImageStringValue(imageIdx, "Path"),
			Modifier: models.CriterionModifierEquals,
		},
	}

	imageFilter := &models.ImageFilterType{
		And: &subFilter,
		Or:  &subFilter,
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Image()

		_, _, err := sqb.Query(imageFilter, nil)
		assert.NotNil(err)

		imageFilter.Or = nil
		imageFilter.Not = &subFilter
		_, _, err = sqb.Query(imageFilter, nil)
		assert.NotNil(err)

		imageFilter.And = nil
		imageFilter.Or = &subFilter
		_, _, err = sqb.Query(imageFilter, nil)
		assert.NotNil(err)

		return nil
	})
}

func TestImageQueryRating(t *testing.T) {
	const rating = 3
	ratingCriterion := models.IntCriterionInput{
		Value:    rating,
		Modifier: models.CriterionModifierEquals,
	}

	verifyImagesRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierNotEquals
	verifyImagesRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyImagesRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierLessThan
	verifyImagesRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierIsNull
	verifyImagesRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierNotNull
	verifyImagesRating(t, ratingCriterion)
}

func verifyImagesRating(t *testing.T, ratingCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		imageFilter := models.ImageFilterType{
			Rating: &ratingCriterion,
		}

		images, _, err := sqb.Query(&imageFilter, nil)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		for _, image := range images {
			verifyInt64(t, image.Rating, ratingCriterion)
		}

		return nil
	})
}

func TestImageQueryOCounter(t *testing.T) {
	const oCounter = 1
	oCounterCriterion := models.IntCriterionInput{
		Value:    oCounter,
		Modifier: models.CriterionModifierEquals,
	}

	verifyImagesOCounter(t, oCounterCriterion)

	oCounterCriterion.Modifier = models.CriterionModifierNotEquals
	verifyImagesOCounter(t, oCounterCriterion)

	oCounterCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyImagesOCounter(t, oCounterCriterion)

	oCounterCriterion.Modifier = models.CriterionModifierLessThan
	verifyImagesOCounter(t, oCounterCriterion)
}

func verifyImagesOCounter(t *testing.T, oCounterCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		imageFilter := models.ImageFilterType{
			OCounter: &oCounterCriterion,
		}

		images, _, err := sqb.Query(&imageFilter, nil)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		for _, image := range images {
			verifyInt(t, image.OCounter, oCounterCriterion)
		}

		return nil
	})
}

func TestImageQueryResolution(t *testing.T) {
	verifyImagesResolution(t, models.ResolutionEnumLow)
	verifyImagesResolution(t, models.ResolutionEnumStandard)
	verifyImagesResolution(t, models.ResolutionEnumStandardHd)
	verifyImagesResolution(t, models.ResolutionEnumFullHd)
	verifyImagesResolution(t, models.ResolutionEnumFourK)
	verifyImagesResolution(t, models.ResolutionEnum("unknown"))
}

func verifyImagesResolution(t *testing.T, resolution models.ResolutionEnum) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		imageFilter := models.ImageFilterType{
			Resolution: &models.ResolutionCriterionInput{
				Value:    resolution,
				Modifier: models.CriterionModifierEquals,
			},
		}

		images, _, err := sqb.Query(&imageFilter, nil)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		for _, image := range images {
			verifyImageResolution(t, image.Height, resolution)
		}

		return nil
	})
}

func verifyImageResolution(t *testing.T, height sql.NullInt64, resolution models.ResolutionEnum) {
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

func TestImageQueryIsMissingGalleries(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		isMissing := "galleries"
		imageFilter := models.ImageFilterType{
			IsMissing: &isMissing,
		}

		q := getImageStringValue(imageIdxWithGallery, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		images, _, err := sqb.Query(&imageFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.Len(t, images, 0)

		findFilter.Q = nil
		images, _, err = sqb.Query(&imageFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		// ensure non of the ids equal the one with gallery
		for _, image := range images {
			assert.NotEqual(t, imageIDs[imageIdxWithGallery], image.ID)
		}

		return nil
	})
}

func TestImageQueryIsMissingStudio(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		isMissing := "studio"
		imageFilter := models.ImageFilterType{
			IsMissing: &isMissing,
		}

		q := getImageStringValue(imageIdxWithStudio, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		images, _, err := sqb.Query(&imageFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.Len(t, images, 0)

		findFilter.Q = nil
		images, _, err = sqb.Query(&imageFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		// ensure non of the ids equal the one with studio
		for _, image := range images {
			assert.NotEqual(t, imageIDs[imageIdxWithStudio], image.ID)
		}

		return nil
	})
}

func TestImageQueryIsMissingPerformers(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		isMissing := "performers"
		imageFilter := models.ImageFilterType{
			IsMissing: &isMissing,
		}

		q := getImageStringValue(imageIdxWithPerformer, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		images, _, err := sqb.Query(&imageFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.Len(t, images, 0)

		findFilter.Q = nil
		images, _, err = sqb.Query(&imageFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.True(t, len(images) > 0)

		// ensure non of the ids equal the one with movies
		for _, image := range images {
			assert.NotEqual(t, imageIDs[imageIdxWithPerformer], image.ID)
		}

		return nil
	})
}

func TestImageQueryIsMissingTags(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		isMissing := "tags"
		imageFilter := models.ImageFilterType{
			IsMissing: &isMissing,
		}

		q := getImageStringValue(imageIdxWithTwoTags, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		images, _, err := sqb.Query(&imageFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.Len(t, images, 0)

		findFilter.Q = nil
		images, _, err = sqb.Query(&imageFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.True(t, len(images) > 0)

		return nil
	})
}

func TestImageQueryIsMissingRating(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		isMissing := "rating"
		imageFilter := models.ImageFilterType{
			IsMissing: &isMissing,
		}

		images, _, err := sqb.Query(&imageFilter, nil)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.True(t, len(images) > 0)

		// ensure date is null, empty or "0001-01-01"
		for _, image := range images {
			assert.True(t, !image.Rating.Valid)
		}

		return nil
	})
}

func TestImageQueryGallery(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		galleryCriterion := models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(galleryIDs[galleryIdxWithImage]),
			},
			Modifier: models.CriterionModifierIncludes,
		}

		imageFilter := models.ImageFilterType{
			Galleries: &galleryCriterion,
		}

		images, _, err := sqb.Query(&imageFilter, nil)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.Len(t, images, 1)

		// ensure ids are correct
		for _, image := range images {
			assert.True(t, image.ID == imageIDs[imageIdxWithGallery])
		}

		galleryCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(galleryIDs[galleryIdx1WithImage]),
				strconv.Itoa(galleryIDs[galleryIdx2WithImage]),
			},
			Modifier: models.CriterionModifierIncludesAll,
		}

		images, _, err = sqb.Query(&imageFilter, nil)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.Len(t, images, 1)
		assert.Equal(t, imageIDs[imageIdxWithTwoGalleries], images[0].ID)

		galleryCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(performerIDs[galleryIdx1WithImage]),
			},
			Modifier: models.CriterionModifierExcludes,
		}

		q := getImageStringValue(imageIdxWithTwoGalleries, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		images, _, err = sqb.Query(&imageFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}
		assert.Len(t, images, 0)

		return nil
	})
}

func TestImageQueryPerformers(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		performerCriterion := models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(performerIDs[performerIdxWithImage]),
				strconv.Itoa(performerIDs[performerIdx1WithImage]),
			},
			Modifier: models.CriterionModifierIncludes,
		}

		imageFilter := models.ImageFilterType{
			Performers: &performerCriterion,
		}

		images, _, err := sqb.Query(&imageFilter, nil)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.Len(t, images, 2)

		// ensure ids are correct
		for _, image := range images {
			assert.True(t, image.ID == imageIDs[imageIdxWithPerformer] || image.ID == imageIDs[imageIdxWithTwoPerformers])
		}

		performerCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(performerIDs[performerIdx1WithImage]),
				strconv.Itoa(performerIDs[performerIdx2WithImage]),
			},
			Modifier: models.CriterionModifierIncludesAll,
		}

		images, _, err = sqb.Query(&imageFilter, nil)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.Len(t, images, 1)
		assert.Equal(t, imageIDs[imageIdxWithTwoPerformers], images[0].ID)

		performerCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(performerIDs[performerIdx1WithImage]),
			},
			Modifier: models.CriterionModifierExcludes,
		}

		q := getImageStringValue(imageIdxWithTwoPerformers, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		images, _, err = sqb.Query(&imageFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}
		assert.Len(t, images, 0)

		return nil
	})
}

func TestImageQueryTags(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		tagCriterion := models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdxWithImage]),
				strconv.Itoa(tagIDs[tagIdx1WithImage]),
			},
			Modifier: models.CriterionModifierIncludes,
		}

		imageFilter := models.ImageFilterType{
			Tags: &tagCriterion,
		}

		images, _, err := sqb.Query(&imageFilter, nil)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.Len(t, images, 2)

		// ensure ids are correct
		for _, image := range images {
			assert.True(t, image.ID == imageIDs[imageIdxWithTag] || image.ID == imageIDs[imageIdxWithTwoTags])
		}

		tagCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdx1WithImage]),
				strconv.Itoa(tagIDs[tagIdx2WithImage]),
			},
			Modifier: models.CriterionModifierIncludesAll,
		}

		images, _, err = sqb.Query(&imageFilter, nil)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.Len(t, images, 1)
		assert.Equal(t, imageIDs[imageIdxWithTwoTags], images[0].ID)

		tagCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdx1WithImage]),
			},
			Modifier: models.CriterionModifierExcludes,
		}

		q := getImageStringValue(imageIdxWithTwoTags, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		images, _, err = sqb.Query(&imageFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}
		assert.Len(t, images, 0)

		return nil
	})
}

func TestImageQueryStudio(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		studioCriterion := models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithImage]),
			},
			Modifier: models.CriterionModifierIncludes,
			Depth:    0,
		}

		imageFilter := models.ImageFilterType{
			Studios: &studioCriterion,
		}

		images, _, err := sqb.Query(&imageFilter, nil)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.Len(t, images, 1)

		// ensure id is correct
		assert.Equal(t, imageIDs[imageIdxWithStudio], images[0].ID)

		studioCriterion = models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithImage]),
			},
			Modifier: models.CriterionModifierExcludes,
			Depth:    0,
		}

		q := getImageStringValue(imageIdxWithStudio, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		images, _, err = sqb.Query(&imageFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}
		assert.Len(t, images, 0)

		return nil
	})
}

func TestImageQueryStudioDepth(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		studioCriterion := models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithGrandChild]),
			},
			Modifier: models.CriterionModifierIncludes,
			Depth:    2,
		}

		imageFilter := models.ImageFilterType{
			Studios: &studioCriterion,
		}

		images := queryImages(t, sqb, &imageFilter, nil)
		assert.Len(t, images, 1)

		studioCriterion.Depth = 1

		images = queryImages(t, sqb, &imageFilter, nil)
		assert.Len(t, images, 0)

		studioCriterion.Value = []string{strconv.Itoa(studioIDs[studioIdxWithParentAndChild])}
		images = queryImages(t, sqb, &imageFilter, nil)
		assert.Len(t, images, 1)

		// ensure id is correct
		assert.Equal(t, imageIDs[imageIdxWithGrandChildStudio], images[0].ID)

		studioCriterion = models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithGrandChild]),
			},
			Modifier: models.CriterionModifierExcludes,
			Depth:    2,
		}

		q := getImageStringValue(imageIdxWithGrandChildStudio, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		images = queryImages(t, sqb, &imageFilter, &findFilter)
		assert.Len(t, images, 0)

		studioCriterion.Depth = 1
		images = queryImages(t, sqb, &imageFilter, &findFilter)
		assert.Len(t, images, 1)

		studioCriterion.Value = []string{strconv.Itoa(studioIDs[studioIdxWithParentAndChild])}
		images = queryImages(t, sqb, &imageFilter, &findFilter)
		assert.Len(t, images, 0)

		return nil
	})
}

func queryImages(t *testing.T, sqb models.ImageReader, imageFilter *models.ImageFilterType, findFilter *models.FindFilterType) []*models.Image {
	images, _, err := sqb.Query(imageFilter, findFilter)
	if err != nil {
		t.Errorf("Error querying images: %s", err.Error())
	}

	return images
}

func TestImageQueryPerformerTags(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		tagCriterion := models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdxWithPerformer]),
				strconv.Itoa(tagIDs[tagIdx1WithPerformer]),
			},
			Modifier: models.CriterionModifierIncludes,
		}

		imageFilter := models.ImageFilterType{
			PerformerTags: &tagCriterion,
		}

		images := queryImages(t, sqb, &imageFilter, nil)
		assert.Len(t, images, 2)

		// ensure ids are correct
		for _, image := range images {
			assert.True(t, image.ID == imageIDs[imageIdxWithPerformerTag] || image.ID == imageIDs[imageIdxWithPerformerTwoTags])
		}

		tagCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdx1WithPerformer]),
				strconv.Itoa(tagIDs[tagIdx2WithPerformer]),
			},
			Modifier: models.CriterionModifierIncludesAll,
		}

		images = queryImages(t, sqb, &imageFilter, nil)

		assert.Len(t, images, 1)
		assert.Equal(t, imageIDs[imageIdxWithPerformerTwoTags], images[0].ID)

		tagCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdx1WithPerformer]),
			},
			Modifier: models.CriterionModifierExcludes,
		}

		q := getImageStringValue(imageIdxWithPerformerTwoTags, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		images = queryImages(t, sqb, &imageFilter, &findFilter)
		assert.Len(t, images, 0)

		return nil
	})
}

func TestImageQueryTagCount(t *testing.T) {
	const tagCount = 1
	tagCountCriterion := models.IntCriterionInput{
		Value:    tagCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyImagesTagCount(t, tagCountCriterion)

	tagCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyImagesTagCount(t, tagCountCriterion)

	tagCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyImagesTagCount(t, tagCountCriterion)

	tagCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyImagesTagCount(t, tagCountCriterion)
}

func verifyImagesTagCount(t *testing.T, tagCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		imageFilter := models.ImageFilterType{
			TagCount: &tagCountCriterion,
		}

		images := queryImages(t, sqb, &imageFilter, nil)
		assert.Greater(t, len(images), 0)

		for _, image := range images {
			ids, err := sqb.GetTagIDs(image.ID)
			if err != nil {
				return err
			}
			verifyInt(t, len(ids), tagCountCriterion)
		}

		return nil
	})
}

func TestImageQueryPerformerCount(t *testing.T) {
	const performerCount = 1
	performerCountCriterion := models.IntCriterionInput{
		Value:    performerCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyImagesPerformerCount(t, performerCountCriterion)

	performerCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyImagesPerformerCount(t, performerCountCriterion)

	performerCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyImagesPerformerCount(t, performerCountCriterion)

	performerCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyImagesPerformerCount(t, performerCountCriterion)
}

func verifyImagesPerformerCount(t *testing.T, performerCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Image()
		imageFilter := models.ImageFilterType{
			PerformerCount: &performerCountCriterion,
		}

		images := queryImages(t, sqb, &imageFilter, nil)
		assert.Greater(t, len(images), 0)

		for _, image := range images {
			ids, err := sqb.GetPerformerIDs(image.ID)
			if err != nil {
				return err
			}
			verifyInt(t, len(ids), performerCountCriterion)
		}

		return nil
	})
}

func TestImageQuerySorting(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sort := titleField
		direction := models.SortDirectionEnumAsc
		findFilter := models.FindFilterType{
			Sort:      &sort,
			Direction: &direction,
		}

		sqb := r.Image()
		images, _, err := sqb.Query(nil, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		// images should be in same order as indexes
		firstImage := images[0]
		lastImage := images[len(images)-1]

		assert.Equal(t, imageIDs[0], firstImage.ID)
		assert.Equal(t, imageIDs[len(imageIDs)-1], lastImage.ID)

		// sort in descending order
		direction = models.SortDirectionEnumDesc

		images, _, err = sqb.Query(nil, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}
		firstImage = images[0]
		lastImage = images[len(images)-1]

		assert.Equal(t, imageIDs[len(imageIDs)-1], firstImage.ID)
		assert.Equal(t, imageIDs[0], lastImage.ID)

		return nil
	})
}

func TestImageQueryPagination(t *testing.T) {
	withTxn(func(r models.Repository) error {
		perPage := 1
		findFilter := models.FindFilterType{
			PerPage: &perPage,
		}

		sqb := r.Image()
		images, _, err := sqb.Query(nil, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.Len(t, images, 1)

		firstID := images[0].ID

		page := 2
		findFilter.Page = &page
		images, _, err = sqb.Query(nil, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}

		assert.Len(t, images, 1)
		secondID := images[0].ID
		assert.NotEqual(t, firstID, secondID)

		perPage = 2
		page = 1

		images, _, err = sqb.Query(nil, &findFilter)
		if err != nil {
			t.Errorf("Error querying image: %s", err.Error())
		}
		assert.Len(t, images, 2)
		assert.Equal(t, firstID, images[0].ID)
		assert.Equal(t, secondID, images[1].ID)

		return nil
	})
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

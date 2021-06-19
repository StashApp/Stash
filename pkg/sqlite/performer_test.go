// +build integration

package sqlite_test

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

func TestPerformerFindBySceneID(t *testing.T) {
	withTxn(func(r models.Repository) error {
		pqb := r.Performer()
		sceneID := sceneIDs[sceneIdxWithPerformer]

		performers, err := pqb.FindBySceneID(sceneID)

		if err != nil {
			t.Errorf("Error finding performer: %s", err.Error())
		}

		assert.Equal(t, 1, len(performers))
		performer := performers[0]

		assert.Equal(t, getPerformerStringValue(performerIdxWithScene, "Name"), performer.Name.String)

		performers, err = pqb.FindBySceneID(0)

		if err != nil {
			t.Errorf("Error finding performer: %s", err.Error())
		}

		assert.Equal(t, 0, len(performers))

		return nil
	})
}

func TestPerformerFindByNames(t *testing.T) {
	getNames := func(p []*models.Performer) []string {
		var ret []string
		for _, pp := range p {
			ret = append(ret, pp.Name.String)
		}
		return ret
	}

	withTxn(func(r models.Repository) error {
		var names []string

		pqb := r.Performer()

		names = append(names, performerNames[performerIdxWithScene]) // find performers by names

		performers, err := pqb.FindByNames(names, false)
		if err != nil {
			t.Errorf("Error finding performers: %s", err.Error())
		}
		assert.Len(t, performers, 1)
		assert.Equal(t, performerNames[performerIdxWithScene], performers[0].Name.String)

		performers, err = pqb.FindByNames(names, true) // find performers by names nocase
		if err != nil {
			t.Errorf("Error finding performers: %s", err.Error())
		}
		assert.Len(t, performers, 2) // performerIdxWithScene and performerIdxWithDupName
		assert.Equal(t, strings.ToLower(performerNames[performerIdxWithScene]), strings.ToLower(performers[0].Name.String))
		assert.Equal(t, strings.ToLower(performerNames[performerIdxWithScene]), strings.ToLower(performers[1].Name.String))

		names = append(names, performerNames[performerIdx1WithScene]) // find performers by names ( 2 names )

		performers, err = pqb.FindByNames(names, false)
		if err != nil {
			t.Errorf("Error finding performers: %s", err.Error())
		}
		retNames := getNames(performers)
		assert.Equal(t, names, retNames)

		performers, err = pqb.FindByNames(names, true) // find performers by names ( 2 names nocase)
		if err != nil {
			t.Errorf("Error finding performers: %s", err.Error())
		}
		retNames = getNames(performers)
		assert.Equal(t, []string{
			performerNames[performerIdxWithScene],
			performerNames[performerIdx1WithScene],
			performerNames[performerIdx1WithDupName],
			performerNames[performerIdxWithDupName],
		}, retNames)

		return nil
	})
}

func TestPerformerQueryEthnicityOr(t *testing.T) {
	const performer1Idx = 1
	const performer2Idx = 2

	performer1Eth := getPerformerStringValue(performer1Idx, "Ethnicity")
	performer2Eth := getPerformerStringValue(performer2Idx, "Ethnicity")

	performerFilter := models.PerformerFilterType{
		Ethnicity: &models.StringCriterionInput{
			Value:    performer1Eth,
			Modifier: models.CriterionModifierEquals,
		},
		Or: &models.PerformerFilterType{
			Ethnicity: &models.StringCriterionInput{
				Value:    performer2Eth,
				Modifier: models.CriterionModifierEquals,
			},
		},
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Performer()

		performers := queryPerformers(t, sqb, &performerFilter, nil)

		assert.Len(t, performers, 2)
		assert.Equal(t, performer1Eth, performers[0].Ethnicity.String)
		assert.Equal(t, performer2Eth, performers[1].Ethnicity.String)

		return nil
	})
}

func TestPerformerQueryEthnicityAndRating(t *testing.T) {
	const performerIdx = 1
	performerEth := getPerformerStringValue(performerIdx, "Ethnicity")
	performerRating := getRating(performerIdx)

	performerFilter := models.PerformerFilterType{
		Ethnicity: &models.StringCriterionInput{
			Value:    performerEth,
			Modifier: models.CriterionModifierEquals,
		},
		And: &models.PerformerFilterType{
			Rating: &models.IntCriterionInput{
				Value:    int(performerRating.Int64),
				Modifier: models.CriterionModifierEquals,
			},
		},
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Performer()

		performers := queryPerformers(t, sqb, &performerFilter, nil)

		assert.Len(t, performers, 1)
		assert.Equal(t, performerEth, performers[0].Ethnicity.String)
		assert.Equal(t, performerRating.Int64, performers[0].Rating.Int64)

		return nil
	})
}

func TestPerformerQueryPathNotRating(t *testing.T) {
	const performerIdx = 1

	performerRating := getRating(performerIdx)

	ethCriterion := models.StringCriterionInput{
		Value:    "performer_.*1_Ethnicity",
		Modifier: models.CriterionModifierMatchesRegex,
	}

	ratingCriterion := models.IntCriterionInput{
		Value:    int(performerRating.Int64),
		Modifier: models.CriterionModifierEquals,
	}

	performerFilter := models.PerformerFilterType{
		Ethnicity: &ethCriterion,
		Not: &models.PerformerFilterType{
			Rating: &ratingCriterion,
		},
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Performer()

		performers := queryPerformers(t, sqb, &performerFilter, nil)

		for _, performer := range performers {
			verifyString(t, performer.Ethnicity.String, ethCriterion)
			ratingCriterion.Modifier = models.CriterionModifierNotEquals
			verifyInt64(t, performer.Rating, ratingCriterion)
		}

		return nil
	})
}

func TestPerformerIllegalQuery(t *testing.T) {
	assert := assert.New(t)

	const performerIdx = 1
	subFilter := models.PerformerFilterType{
		Ethnicity: &models.StringCriterionInput{
			Value:    getPerformerStringValue(performerIdx, "Ethnicity"),
			Modifier: models.CriterionModifierEquals,
		},
	}

	performerFilter := &models.PerformerFilterType{
		And: &subFilter,
		Or:  &subFilter,
	}

	withTxn(func(r models.Repository) error {
		sqb := r.Performer()

		_, _, err := sqb.Query(performerFilter, nil)
		assert.NotNil(err)

		performerFilter.Or = nil
		performerFilter.Not = &subFilter
		_, _, err = sqb.Query(performerFilter, nil)
		assert.NotNil(err)

		performerFilter.And = nil
		performerFilter.Or = &subFilter
		_, _, err = sqb.Query(performerFilter, nil)
		assert.NotNil(err)

		return nil
	})
}

func TestPerformerQueryForAutoTag(t *testing.T) {
	withTxn(func(r models.Repository) error {
		tqb := r.Performer()

		name := performerNames[performerIdxWithScene] // find a performer by name

		performers, err := tqb.QueryForAutoTag([]string{name})

		if err != nil {
			t.Errorf("Error finding performers: %s", err.Error())
		}

		assert.Len(t, performers, 2)
		assert.Equal(t, strings.ToLower(performerNames[performerIdxWithScene]), strings.ToLower(performers[0].Name.String))
		assert.Equal(t, strings.ToLower(performerNames[performerIdxWithScene]), strings.ToLower(performers[1].Name.String))

		return nil
	})
}

func TestPerformerUpdatePerformerImage(t *testing.T) {
	if err := withTxn(func(r models.Repository) error {
		qb := r.Performer()

		// create performer to test against
		const name = "TestPerformerUpdatePerformerImage"
		performer := models.Performer{
			Name:     sql.NullString{String: name, Valid: true},
			Checksum: utils.MD5FromString(name),
			Favorite: sql.NullBool{Bool: false, Valid: true},
		}
		created, err := qb.Create(performer)
		if err != nil {
			return fmt.Errorf("Error creating performer: %s", err.Error())
		}

		image := []byte("image")
		err = qb.UpdateImage(created.ID, image)
		if err != nil {
			return fmt.Errorf("Error updating performer image: %s", err.Error())
		}

		// ensure image set
		storedImage, err := qb.GetImage(created.ID)
		if err != nil {
			return fmt.Errorf("Error getting image: %s", err.Error())
		}
		assert.Equal(t, storedImage, image)

		// set nil image
		err = qb.UpdateImage(created.ID, nil)
		if err == nil {
			return fmt.Errorf("Expected error setting nil image")
		}

		return nil
	}); err != nil {
		t.Error(err.Error())
	}
}

func TestPerformerDestroyPerformerImage(t *testing.T) {
	if err := withTxn(func(r models.Repository) error {
		qb := r.Performer()

		// create performer to test against
		const name = "TestPerformerDestroyPerformerImage"
		performer := models.Performer{
			Name:     sql.NullString{String: name, Valid: true},
			Checksum: utils.MD5FromString(name),
			Favorite: sql.NullBool{Bool: false, Valid: true},
		}
		created, err := qb.Create(performer)
		if err != nil {
			return fmt.Errorf("Error creating performer: %s", err.Error())
		}

		image := []byte("image")
		err = qb.UpdateImage(created.ID, image)
		if err != nil {
			return fmt.Errorf("Error updating performer image: %s", err.Error())
		}

		err = qb.DestroyImage(created.ID)
		if err != nil {
			return fmt.Errorf("Error destroying performer image: %s", err.Error())
		}

		// image should be nil
		storedImage, err := qb.GetImage(created.ID)
		if err != nil {
			return fmt.Errorf("Error getting image: %s", err.Error())
		}
		assert.Nil(t, storedImage)

		return nil
	}); err != nil {
		t.Error(err.Error())
	}
}

func TestPerformerQueryAge(t *testing.T) {
	const age = 19
	ageCriterion := models.IntCriterionInput{
		Value:    age,
		Modifier: models.CriterionModifierEquals,
	}

	verifyPerformerAge(t, ageCriterion)

	ageCriterion.Modifier = models.CriterionModifierNotEquals
	verifyPerformerAge(t, ageCriterion)

	ageCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyPerformerAge(t, ageCriterion)

	ageCriterion.Modifier = models.CriterionModifierLessThan
	verifyPerformerAge(t, ageCriterion)
}

func verifyPerformerAge(t *testing.T, ageCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		qb := r.Performer()
		performerFilter := models.PerformerFilterType{
			Age: &ageCriterion,
		}

		performers, _, err := qb.Query(&performerFilter, nil)
		if err != nil {
			t.Errorf("Error querying performer: %s", err.Error())
		}

		now := time.Now()
		for _, performer := range performers {
			cd := now

			if performer.DeathDate.Valid {
				cd, _ = time.Parse("2006-01-02", performer.DeathDate.String)
			}

			bd := performer.Birthdate.String
			d, _ := time.Parse("2006-01-02", bd)
			age := cd.Year() - d.Year()
			if cd.YearDay() < d.YearDay() {
				age = age - 1
			}

			verifyInt(t, age, ageCriterion)
		}

		return nil
	})
}

func TestPerformerQueryCareerLength(t *testing.T) {
	const value = "2005"
	careerLengthCriterion := models.StringCriterionInput{
		Value:    value,
		Modifier: models.CriterionModifierEquals,
	}

	verifyPerformerCareerLength(t, careerLengthCriterion)

	careerLengthCriterion.Modifier = models.CriterionModifierNotEquals
	verifyPerformerCareerLength(t, careerLengthCriterion)

	careerLengthCriterion.Modifier = models.CriterionModifierMatchesRegex
	verifyPerformerCareerLength(t, careerLengthCriterion)

	careerLengthCriterion.Modifier = models.CriterionModifierNotMatchesRegex
	verifyPerformerCareerLength(t, careerLengthCriterion)
}

func verifyPerformerCareerLength(t *testing.T, criterion models.StringCriterionInput) {
	withTxn(func(r models.Repository) error {
		qb := r.Performer()
		performerFilter := models.PerformerFilterType{
			CareerLength: &criterion,
		}

		performers, _, err := qb.Query(&performerFilter, nil)
		if err != nil {
			t.Errorf("Error querying performer: %s", err.Error())
		}

		for _, performer := range performers {
			cl := performer.CareerLength
			verifyNullString(t, cl, criterion)
		}

		return nil
	})
}

func TestPerformerQueryURL(t *testing.T) {
	const sceneIdx = 1
	performerURL := getPerformerStringValue(sceneIdx, urlField)

	urlCriterion := models.StringCriterionInput{
		Value:    performerURL,
		Modifier: models.CriterionModifierEquals,
	}

	filter := models.PerformerFilterType{
		URL: &urlCriterion,
	}

	verifyFn := func(g *models.Performer) {
		t.Helper()
		verifyNullString(t, g.URL, urlCriterion)
	}

	verifyPerformerQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierNotEquals
	verifyPerformerQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierMatchesRegex
	urlCriterion.Value = "performer_.*1_URL"
	verifyPerformerQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierNotMatchesRegex
	verifyPerformerQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierIsNull
	urlCriterion.Value = ""
	verifyPerformerQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierNotNull
	verifyPerformerQuery(t, filter, verifyFn)
}

func verifyPerformerQuery(t *testing.T, filter models.PerformerFilterType, verifyFn func(s *models.Performer)) {
	withTxn(func(r models.Repository) error {
		t.Helper()
		sqb := r.Performer()

		performers := queryPerformers(t, sqb, &filter, nil)

		// assume it should find at least one
		assert.Greater(t, len(performers), 0)

		for _, p := range performers {
			verifyFn(p)
		}

		return nil
	})
}

func queryPerformers(t *testing.T, qb models.PerformerReader, performerFilter *models.PerformerFilterType, findFilter *models.FindFilterType) []*models.Performer {
	performers, _, err := qb.Query(performerFilter, findFilter)
	if err != nil {
		t.Errorf("Error querying performers: %s", err.Error())
	}

	return performers
}

func TestPerformerQueryTags(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Performer()
		tagCriterion := models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdxWithPerformer]),
				strconv.Itoa(tagIDs[tagIdx1WithPerformer]),
			},
			Modifier: models.CriterionModifierIncludes,
			Depth:    0,
		}

		performerFilter := models.PerformerFilterType{
			Tags: &tagCriterion,
		}

		// ensure ids are correct
		performers := queryPerformers(t, sqb, &performerFilter, nil)
		assert.Len(t, performers, 2)
		for _, performer := range performers {
			assert.True(t, performer.ID == performerIDs[performerIdxWithTag] || performer.ID == performerIDs[performerIdxWithTwoTags])
		}

		tagCriterion = models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdx1WithPerformer]),
				strconv.Itoa(tagIDs[tagIdx2WithPerformer]),
			},
			Modifier: models.CriterionModifierIncludesAll,
			Depth:    0,
		}

		performers = queryPerformers(t, sqb, &performerFilter, nil)

		assert.Len(t, performers, 1)
		assert.Equal(t, sceneIDs[performerIdxWithTwoTags], performers[0].ID)

		tagCriterion = models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(tagIDs[tagIdx1WithPerformer]),
			},
			Modifier: models.CriterionModifierExcludes,
			Depth:    0,
		}

		q := getSceneStringValue(performerIdxWithTwoTags, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		performers = queryPerformers(t, sqb, &performerFilter, &findFilter)
		assert.Len(t, performers, 0)

		return nil
	})
}

func TestPerformerQueryTagCount(t *testing.T) {
	const tagCount = 1
	tagCountCriterion := models.IntCriterionInput{
		Value:    tagCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyPerformersTagCount(t, tagCountCriterion)

	tagCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyPerformersTagCount(t, tagCountCriterion)

	tagCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyPerformersTagCount(t, tagCountCriterion)

	tagCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyPerformersTagCount(t, tagCountCriterion)
}

func verifyPerformersTagCount(t *testing.T, tagCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Performer()
		performerFilter := models.PerformerFilterType{
			TagCount: &tagCountCriterion,
		}

		performers := queryPerformers(t, sqb, &performerFilter, nil)
		assert.Greater(t, len(performers), 0)

		for _, performer := range performers {
			ids, err := sqb.GetTagIDs(performer.ID)
			if err != nil {
				return err
			}
			verifyInt(t, len(ids), tagCountCriterion)
		}

		return nil
	})
}

func TestPerformerQuerySceneCount(t *testing.T) {
	const sceneCount = 1
	sceneCountCriterion := models.IntCriterionInput{
		Value:    sceneCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyPerformersSceneCount(t, sceneCountCriterion)

	sceneCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyPerformersSceneCount(t, sceneCountCriterion)

	sceneCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyPerformersSceneCount(t, sceneCountCriterion)

	sceneCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyPerformersSceneCount(t, sceneCountCriterion)
}

func verifyPerformersSceneCount(t *testing.T, sceneCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Performer()
		performerFilter := models.PerformerFilterType{
			SceneCount: &sceneCountCriterion,
		}

		performers := queryPerformers(t, sqb, &performerFilter, nil)
		assert.Greater(t, len(performers), 0)

		for _, performer := range performers {
			ids, err := r.Scene().FindByPerformerID(performer.ID)
			if err != nil {
				return err
			}
			verifyInt(t, len(ids), sceneCountCriterion)
		}

		return nil
	})
}

func TestPerformerQueryImageCount(t *testing.T) {
	const imageCount = 1
	imageCountCriterion := models.IntCriterionInput{
		Value:    imageCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyPerformersImageCount(t, imageCountCriterion)

	imageCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyPerformersImageCount(t, imageCountCriterion)

	imageCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyPerformersImageCount(t, imageCountCriterion)

	imageCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyPerformersImageCount(t, imageCountCriterion)
}

func verifyPerformersImageCount(t *testing.T, imageCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Performer()
		performerFilter := models.PerformerFilterType{
			ImageCount: &imageCountCriterion,
		}

		performers := queryPerformers(t, sqb, &performerFilter, nil)
		assert.Greater(t, len(performers), 0)

		for _, performer := range performers {
			pp := 0

			_, count, err := r.Image().Query(&models.ImageFilterType{
				Performers: &models.MultiCriterionInput{
					Value:    []string{strconv.Itoa(performer.ID)},
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

func TestPerformerQueryGalleryCount(t *testing.T) {
	const galleryCount = 1
	galleryCountCriterion := models.IntCriterionInput{
		Value:    galleryCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyPerformersGalleryCount(t, galleryCountCriterion)

	galleryCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyPerformersGalleryCount(t, galleryCountCriterion)

	galleryCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyPerformersGalleryCount(t, galleryCountCriterion)

	galleryCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyPerformersGalleryCount(t, galleryCountCriterion)
}

func verifyPerformersGalleryCount(t *testing.T, galleryCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Performer()
		performerFilter := models.PerformerFilterType{
			GalleryCount: &galleryCountCriterion,
		}

		performers := queryPerformers(t, sqb, &performerFilter, nil)
		assert.Greater(t, len(performers), 0)

		for _, performer := range performers {
			pp := 0

			_, count, err := r.Gallery().Query(&models.GalleryFilterType{
				Performers: &models.MultiCriterionInput{
					Value:    []string{strconv.Itoa(performer.ID)},
					Modifier: models.CriterionModifierIncludes,
				},
			}, &models.FindFilterType{
				PerPage: &pp,
			})
			if err != nil {
				return err
			}
			verifyInt(t, count, galleryCountCriterion)
		}

		return nil
	})
}

func TestPerformerQueryStudio(t *testing.T) {
	withTxn(func(r models.Repository) error {
		testCases := []struct {
			studioIndex    int
			performerIndex int
		}{
			{studioIndex: studioIdxWithScenePerformer, performerIndex: performerIdxWithSceneStudio},
			{studioIndex: studioIdxWithImagePerformer, performerIndex: performerIdxWithImageStudio},
			{studioIndex: studioIdxWithGalleryPerformer, performerIndex: performerIdxWithGalleryStudio},
		}

		sqb := r.Performer()

		for _, tc := range testCases {
			studioCriterion := models.HierarchicalMultiCriterionInput{
				Value: []string{
					strconv.Itoa(studioIDs[tc.studioIndex]),
				},
				Modifier: models.CriterionModifierIncludes,
			}

			performerFilter := models.PerformerFilterType{
				Studios: &studioCriterion,
			}

			performers := queryPerformers(t, sqb, &performerFilter, nil)

			assert.Len(t, performers, 1)

			// ensure id is correct
			assert.Equal(t, performerIDs[tc.performerIndex], performers[0].ID)

			studioCriterion = models.HierarchicalMultiCriterionInput{
				Value: []string{
					strconv.Itoa(studioIDs[tc.studioIndex]),
				},
				Modifier: models.CriterionModifierExcludes,
			}

			q := getPerformerStringValue(tc.performerIndex, "Name")
			findFilter := models.FindFilterType{
				Q: &q,
			}

			performers = queryPerformers(t, sqb, &performerFilter, &findFilter)
			assert.Len(t, performers, 0)
		}

		return nil
	})
}

func TestPerformerStashIDs(t *testing.T) {
	if err := withTxn(func(r models.Repository) error {
		qb := r.Performer()

		// create performer to test against
		const name = "TestStashIDs"
		performer := models.Performer{
			Name:     sql.NullString{String: name, Valid: true},
			Checksum: utils.MD5FromString(name),
			Favorite: sql.NullBool{Bool: false, Valid: true},
		}
		created, err := qb.Create(performer)
		if err != nil {
			return fmt.Errorf("Error creating performer: %s", err.Error())
		}

		testStashIDReaderWriter(t, qb, created.ID)
		return nil
	}); err != nil {
		t.Error(err.Error())
	}
}
func TestPerformerQueryRating(t *testing.T) {
	const rating = 3
	ratingCriterion := models.IntCriterionInput{
		Value:    rating,
		Modifier: models.CriterionModifierEquals,
	}

	verifyPerformersRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierNotEquals
	verifyPerformersRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyPerformersRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierLessThan
	verifyPerformersRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierIsNull
	verifyPerformersRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierNotNull
	verifyPerformersRating(t, ratingCriterion)
}

func verifyPerformersRating(t *testing.T, ratingCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Performer()
		performerFilter := models.PerformerFilterType{
			Rating: &ratingCriterion,
		}

		performers := queryPerformers(t, sqb, &performerFilter, nil)

		for _, performer := range performers {
			verifyInt64(t, performer.Rating, ratingCriterion)
		}

		return nil
	})
}

func TestPerformerQueryIsMissingRating(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Performer()
		isMissing := "rating"
		performerFilter := models.PerformerFilterType{
			IsMissing: &isMissing,
		}

		performers := queryPerformers(t, sqb, &performerFilter, nil)

		assert.True(t, len(performers) > 0)

		for _, performer := range performers {
			assert.True(t, !performer.Rating.Valid)
		}

		return nil
	})
}

func TestPerformerQueryIsMissingImage(t *testing.T) {
	withTxn(func(r models.Repository) error {
		isMissing := "image"
		performerFilter := &models.PerformerFilterType{
			IsMissing: &isMissing,
		}

		// ensure query does not error
		performers, _, err := r.Performer().Query(performerFilter, nil)
		if err != nil {
			t.Errorf("Error querying performers: %s", err.Error())
		}

		assert.True(t, len(performers) > 0)

		for _, performer := range performers {
			img, err := r.Performer().GetImage(performer.ID)
			if err != nil {
				t.Errorf("error getting performer image: %s", err.Error())
			}
			assert.Nil(t, img)
		}

		return nil
	})
}

func TestPerformerQuerySortScenesCount(t *testing.T) {
	sort := "scenes_count"
	direction := models.SortDirectionEnumDesc
	findFilter := &models.FindFilterType{
		Sort:      &sort,
		Direction: &direction,
	}

	withTxn(func(r models.Repository) error {
		// just ensure it queries without error
		performers, _, err := r.Performer().Query(nil, findFilter)
		if err != nil {
			t.Errorf("Error querying performers: %s", err.Error())
		}

		assert.True(t, len(performers) > 0)

		// first performer should be performerIdxWithTwoScenes
		firstPerformer := performers[0]

		assert.Equal(t, performerIDs[performerIdxWithTwoScenes], firstPerformer.ID)

		// sort in ascending order
		direction = models.SortDirectionEnumAsc

		performers, _, err = r.Performer().Query(nil, findFilter)
		if err != nil {
			t.Errorf("Error querying performers: %s", err.Error())
		}

		assert.True(t, len(performers) > 0)
		lastPerformer := performers[len(performers)-1]

		assert.Equal(t, performerIDs[performerIdxWithTwoScenes], lastPerformer.ID)

		return nil
	})
}

// TODO Update
// TODO Destroy
// TODO Find
// TODO Count
// TODO All
// TODO AllSlim
// TODO Query

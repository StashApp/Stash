// +build integration

package sqlite_test

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestStudioFindByName(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Studio()

		name := studioNames[studioIdxWithScene] // find a studio by name

		studio, err := sqb.FindByName(name, false)

		if err != nil {
			t.Errorf("Error finding studios: %s", err.Error())
		}

		assert.Equal(t, studioNames[studioIdxWithScene], studio.Name.String)

		name = studioNames[studioIdxWithDupName] // find a studio by name nocase

		studio, err = sqb.FindByName(name, true)

		if err != nil {
			t.Errorf("Error finding studios: %s", err.Error())
		}
		// studioIdxWithDupName and studioIdxWithScene should have similar names ( only diff should be Name vs NaMe)
		//studio.Name should match with studioIdxWithScene since its ID is before studioIdxWithDupName
		assert.Equal(t, studioNames[studioIdxWithScene], studio.Name.String)
		//studio.Name should match with studioIdxWithDupName if the check is not case sensitive
		assert.Equal(t, strings.ToLower(studioNames[studioIdxWithDupName]), strings.ToLower(studio.Name.String))

		return nil
	})
}

func TestStudioQueryForAutoTag(t *testing.T) {
	withTxn(func(r models.Repository) error {
		tqb := r.Studio()

		name := studioNames[studioIdxWithScene] // find a studio by name

		studios, err := tqb.QueryForAutoTag([]string{name})

		if err != nil {
			t.Errorf("Error finding studios: %s", err.Error())
		}

		assert.Len(t, studios, 2)
		assert.Equal(t, strings.ToLower(studioNames[studioIdxWithScene]), strings.ToLower(studios[0].Name.String))
		assert.Equal(t, strings.ToLower(studioNames[studioIdxWithScene]), strings.ToLower(studios[1].Name.String))

		return nil
	})
}

func TestStudioQueryParent(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Studio()
		studioCriterion := models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithChildStudio]),
			},
			Modifier: models.CriterionModifierIncludes,
		}

		studioFilter := models.StudioFilterType{
			Parents: &studioCriterion,
		}

		studios, _, err := sqb.Query(&studioFilter, nil)
		if err != nil {
			t.Errorf("Error querying studio: %s", err.Error())
		}

		assert.Len(t, studios, 1)

		// ensure id is correct
		assert.Equal(t, sceneIDs[studioIdxWithParentStudio], studios[0].ID)

		studioCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithChildStudio]),
			},
			Modifier: models.CriterionModifierExcludes,
		}

		q := getStudioStringValue(studioIdxWithParentStudio, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		studios, _, err = sqb.Query(&studioFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying studio: %s", err.Error())
		}
		assert.Len(t, studios, 0)

		return nil
	})
}

func TestStudioDestroyParent(t *testing.T) {
	const parentName = "parent"
	const childName = "child"

	// create parent and child studios
	if err := withTxn(func(r models.Repository) error {
		createdParent, err := createStudio(r.Studio(), parentName, nil)
		if err != nil {
			return fmt.Errorf("Error creating parent studio: %s", err.Error())
		}

		parentID := int64(createdParent.ID)
		createdChild, err := createStudio(r.Studio(), childName, &parentID)
		if err != nil {
			return fmt.Errorf("Error creating child studio: %s", err.Error())
		}

		sqb := r.Studio()

		// destroy the parent
		err = sqb.Destroy(createdParent.ID)
		if err != nil {
			return fmt.Errorf("Error destroying parent studio: %s", err.Error())
		}

		// destroy the child
		err = sqb.Destroy(createdChild.ID)
		if err != nil {
			return fmt.Errorf("Error destroying child studio: %s", err.Error())
		}

		return nil
	}); err != nil {
		t.Error(err.Error())
	}
}

func TestStudioFindChildren(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Studio()

		studios, err := sqb.FindChildren(studioIDs[studioIdxWithChildStudio])

		if err != nil {
			t.Errorf("error calling FindChildren: %s", err.Error())
		}

		assert.Len(t, studios, 1)
		assert.Equal(t, studioIDs[studioIdxWithParentStudio], studios[0].ID)

		studios, err = sqb.FindChildren(0)

		if err != nil {
			t.Errorf("error calling FindChildren: %s", err.Error())
		}

		assert.Len(t, studios, 0)

		return nil
	})
}

func TestStudioUpdateClearParent(t *testing.T) {
	const parentName = "clearParent_parent"
	const childName = "clearParent_child"

	// create parent and child studios
	if err := withTxn(func(r models.Repository) error {
		createdParent, err := createStudio(r.Studio(), parentName, nil)
		if err != nil {
			return fmt.Errorf("Error creating parent studio: %s", err.Error())
		}

		parentID := int64(createdParent.ID)
		createdChild, err := createStudio(r.Studio(), childName, &parentID)
		if err != nil {
			return fmt.Errorf("Error creating child studio: %s", err.Error())
		}

		sqb := r.Studio()

		// clear the parent id from the child
		updatePartial := models.StudioPartial{
			ID:       createdChild.ID,
			ParentID: &sql.NullInt64{Valid: false},
		}

		updatedStudio, err := sqb.Update(updatePartial)

		if err != nil {
			return fmt.Errorf("Error updated studio: %s", err.Error())
		}

		if updatedStudio.ParentID.Valid {
			return errors.New("updated studio has parent ID set")
		}

		return nil
	}); err != nil {
		t.Error(err.Error())
	}
}

func TestStudioUpdateStudioImage(t *testing.T) {
	if err := withTxn(func(r models.Repository) error {
		qb := r.Studio()

		// create performer to test against
		const name = "TestStudioUpdateStudioImage"
		created, err := createStudio(r.Studio(), name, nil)
		if err != nil {
			return fmt.Errorf("Error creating studio: %s", err.Error())
		}

		image := []byte("image")
		err = qb.UpdateImage(created.ID, image)
		if err != nil {
			return fmt.Errorf("Error updating studio image: %s", err.Error())
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

func TestStudioDestroyStudioImage(t *testing.T) {
	if err := withTxn(func(r models.Repository) error {
		qb := r.Studio()

		// create performer to test against
		const name = "TestStudioDestroyStudioImage"
		created, err := createStudio(r.Studio(), name, nil)
		if err != nil {
			return fmt.Errorf("Error creating studio: %s", err.Error())
		}

		image := []byte("image")
		err = qb.UpdateImage(created.ID, image)
		if err != nil {
			return fmt.Errorf("Error updating studio image: %s", err.Error())
		}

		err = qb.DestroyImage(created.ID)
		if err != nil {
			return fmt.Errorf("Error destroying studio image: %s", err.Error())
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

func TestStudioQuerySceneCount(t *testing.T) {
	const sceneCount = 1
	sceneCountCriterion := models.IntCriterionInput{
		Exact:    sceneCount,
		Upper:    sceneCount,
		Lower:    sceneCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyStudiosSceneCount(t, sceneCountCriterion)

	sceneCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyStudiosSceneCount(t, sceneCountCriterion)

	sceneCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyStudiosSceneCount(t, sceneCountCriterion)

	sceneCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyStudiosSceneCount(t, sceneCountCriterion)
}

func verifyStudiosSceneCount(t *testing.T, sceneCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Studio()
		studioFilter := models.StudioFilterType{
			SceneCount: &sceneCountCriterion,
		}

		studios := queryStudio(t, sqb, &studioFilter, nil)
		assert.Greater(t, len(studios), 0)

		for _, studio := range studios {
			sceneCount, err := r.Scene().CountByStudioID(studio.ID)
			if err != nil {
				return err
			}
			verifyInt(t, sceneCount, sceneCountCriterion)
		}

		return nil
	})
}

func TestStudioQueryImageCount(t *testing.T) {
	const imageCount = 1
	imageCountCriterion := models.IntCriterionInput{
		Exact:    imageCount,
		Upper:    imageCount,
		Lower:    imageCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyStudiosImageCount(t, imageCountCriterion)

	imageCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyStudiosImageCount(t, imageCountCriterion)

	imageCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyStudiosImageCount(t, imageCountCriterion)

	imageCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyStudiosImageCount(t, imageCountCriterion)
}

func verifyStudiosImageCount(t *testing.T, imageCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Studio()
		studioFilter := models.StudioFilterType{
			ImageCount: &imageCountCriterion,
		}

		studios := queryStudio(t, sqb, &studioFilter, nil)
		assert.Greater(t, len(studios), 0)

		for _, studio := range studios {
			pp := 0

			_, count, err := r.Image().Query(&models.ImageFilterType{
				Studios: &models.HierarchicalMultiCriterionInput{
					Value:    []string{strconv.Itoa(studio.ID)},
					Modifier: models.CriterionModifierIncludes,
					Depth:    0,
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

func TestStudioQueryGalleryCount(t *testing.T) {
	const galleryCount = 1
	galleryCountCriterion := models.IntCriterionInput{
		Exact:    galleryCount,
		Upper:    galleryCount,
		Lower:    galleryCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyStudiosGalleryCount(t, galleryCountCriterion)

	galleryCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyStudiosGalleryCount(t, galleryCountCriterion)

	galleryCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyStudiosGalleryCount(t, galleryCountCriterion)

	galleryCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyStudiosGalleryCount(t, galleryCountCriterion)
}

func verifyStudiosGalleryCount(t *testing.T, galleryCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Studio()
		studioFilter := models.StudioFilterType{
			GalleryCount: &galleryCountCriterion,
		}

		studios := queryStudio(t, sqb, &studioFilter, nil)
		assert.Greater(t, len(studios), 0)

		for _, studio := range studios {
			pp := 0

			_, count, err := r.Gallery().Query(&models.GalleryFilterType{
				Studios: &models.HierarchicalMultiCriterionInput{
					Value:    []string{strconv.Itoa(studio.ID)},
					Modifier: models.CriterionModifierIncludes,
					Depth:    0,
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

func TestStudioStashIDs(t *testing.T) {
	if err := withTxn(func(r models.Repository) error {
		qb := r.Studio()

		// create studio to test against
		const name = "TestStudioStashIDs"
		created, err := createStudio(r.Studio(), name, nil)
		if err != nil {
			return fmt.Errorf("Error creating studio: %s", err.Error())
		}

		testStashIDReaderWriter(t, qb, created.ID)
		return nil
	}); err != nil {
		t.Error(err.Error())
	}
}

func TestStudioQueryURL(t *testing.T) {
	const sceneIdx = 1
	studioURL := getStudioStringValue(sceneIdx, urlField)

	urlCriterion := models.StringCriterionInput{
		Value:    studioURL,
		Modifier: models.CriterionModifierEquals,
	}

	filter := models.StudioFilterType{
		URL: &urlCriterion,
	}

	verifyFn := func(g *models.Studio) {
		t.Helper()
		verifyNullString(t, g.URL, urlCriterion)
	}

	verifyStudioQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierNotEquals
	verifyStudioQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierMatchesRegex
	urlCriterion.Value = "studio_.*1_URL"
	verifyStudioQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierNotMatchesRegex
	verifyStudioQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierIsNull
	urlCriterion.Value = ""
	verifyStudioQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierNotNull
	verifyStudioQuery(t, filter, verifyFn)
}

func TestStudioQueryRating(t *testing.T) {
	const rating = 3
	ratingCriterion := models.IntCriterionInput{
		Exact:    rating,
		Upper:    rating,
		Lower:    rating,
		Modifier: models.CriterionModifierEquals,
	}

	verifyStudiosRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierNotEquals
	verifyStudiosRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyStudiosRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierLessThan
	verifyStudiosRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierIsNull
	verifyStudiosRating(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierNotNull
	verifyStudiosRating(t, ratingCriterion)
}

func verifyStudioQuery(t *testing.T, filter models.StudioFilterType, verifyFn func(s *models.Studio)) {
	withTxn(func(r models.Repository) error {
		t.Helper()
		sqb := r.Studio()

		studios := queryStudio(t, sqb, &filter, nil)

		// assume it should find at least one
		assert.Greater(t, len(studios), 0)

		for _, studio := range studios {
			verifyFn(studio)
		}

		return nil
	})
}

func verifyStudiosRating(t *testing.T, ratingCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		sqb := r.Studio()
		studioFilter := models.StudioFilterType{
			Rating: &ratingCriterion,
		}

		studios, _, err := sqb.Query(&studioFilter, nil)

		if err != nil {
			t.Errorf("Error querying studio: %s", err.Error())
		}

		for _, studio := range studios {
			verifyInt64(t, studio.Rating, ratingCriterion)
		}

		return nil
	})
}

func TestStudioQueryIsMissingRating(t *testing.T) {
	withTxn(func(r models.Repository) error {
		sqb := r.Studio()
		isMissing := "rating"
		studioFilter := models.StudioFilterType{
			IsMissing: &isMissing,
		}

		studios, _, err := sqb.Query(&studioFilter, nil)

		if err != nil {
			t.Errorf("Error querying studio: %s", err.Error())
		}

		assert.True(t, len(studios) > 0)

		for _, studio := range studios {
			assert.True(t, !studio.Rating.Valid)
		}

		return nil
	})
}

func queryStudio(t *testing.T, sqb models.StudioReader, studioFilter *models.StudioFilterType, findFilter *models.FindFilterType) []*models.Studio {
	studios, _, err := sqb.Query(studioFilter, findFilter)
	if err != nil {
		t.Errorf("Error querying studio: %s", err.Error())
	}

	return studios
}

// TODO Create
// TODO Update
// TODO Destroy
// TODO Find
// TODO FindBySceneID
// TODO Count
// TODO All
// TODO AllSlim
// TODO Query

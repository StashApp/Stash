// +build integration

package sqlite_test

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestMarkerFindBySceneMarkerID(t *testing.T) {
	withTxn(func(r models.Repository) error {
		tqb := r.Tag()

		markerID := markerIDs[markerIdxWithScene]

		tags, err := tqb.FindBySceneMarkerID(markerID)

		if err != nil {
			t.Errorf("Error finding tags: %s", err.Error())
		}

		assert.Len(t, tags, 1)
		assert.Equal(t, tagIDs[tagIdxWithMarker], tags[0].ID)

		tags, err = tqb.FindBySceneMarkerID(0)

		if err != nil {
			t.Errorf("Error finding tags: %s", err.Error())
		}

		assert.Len(t, tags, 0)

		return nil
	})
}

func TestTagFindByName(t *testing.T) {
	withTxn(func(r models.Repository) error {
		tqb := r.Tag()

		name := tagNames[tagIdxWithScene] // find a tag by name

		tag, err := tqb.FindByName(name, false)

		if err != nil {
			t.Errorf("Error finding tags: %s", err.Error())
		}

		assert.Equal(t, tagNames[tagIdxWithScene], tag.Name)

		name = tagNames[tagIdxWithDupName] // find a tag by name nocase

		tag, err = tqb.FindByName(name, true)

		if err != nil {
			t.Errorf("Error finding tags: %s", err.Error())
		}
		// tagIdxWithDupName and tagIdxWithScene should have similar names ( only diff should be Name vs NaMe)
		//tag.Name should match with tagIdxWithScene since its ID is before tagIdxWithDupName
		assert.Equal(t, tagNames[tagIdxWithScene], tag.Name)
		//tag.Name should match with tagIdxWithDupName if the check is not case sensitive
		assert.Equal(t, strings.ToLower(tagNames[tagIdxWithDupName]), strings.ToLower(tag.Name))

		return nil
	})
}

func TestTagQueryForAutoTag(t *testing.T) {
	withTxn(func(r models.Repository) error {
		tqb := r.Tag()

		name := tagNames[tagIdxWithScene] // find a tag by name

		tags, err := tqb.QueryForAutoTag([]string{name})

		if err != nil {
			t.Errorf("Error finding tags: %s", err.Error())
		}

		assert.Len(t, tags, 2)
		lcName := tagNames[tagIdxWithScene]
		assert.Equal(t, strings.ToLower(lcName), strings.ToLower(tags[0].Name))
		assert.Equal(t, strings.ToLower(lcName), strings.ToLower(tags[1].Name))

		// find by alias
		name = getTagStringValue(tagIdxWithScene, "Alias")
		tags, err = tqb.QueryForAutoTag([]string{name})

		if err != nil {
			t.Errorf("Error finding tags: %s", err.Error())
		}

		assert.Len(t, tags, 1)
		assert.Equal(t, tagIDs[tagIdxWithScene], tags[0].ID)

		return nil
	})
}

func TestTagFindByNames(t *testing.T) {
	var names []string

	withTxn(func(r models.Repository) error {
		tqb := r.Tag()

		names = append(names, tagNames[tagIdxWithScene]) // find tags by names

		tags, err := tqb.FindByNames(names, false)
		if err != nil {
			t.Errorf("Error finding tags: %s", err.Error())
		}
		assert.Len(t, tags, 1)
		assert.Equal(t, tagNames[tagIdxWithScene], tags[0].Name)

		tags, err = tqb.FindByNames(names, true) // find tags by names nocase
		if err != nil {
			t.Errorf("Error finding tags: %s", err.Error())
		}
		assert.Len(t, tags, 2) // tagIdxWithScene and tagIdxWithDupName
		assert.Equal(t, strings.ToLower(tagNames[tagIdxWithScene]), strings.ToLower(tags[0].Name))
		assert.Equal(t, strings.ToLower(tagNames[tagIdxWithScene]), strings.ToLower(tags[1].Name))

		names = append(names, tagNames[tagIdx1WithScene]) // find tags by names ( 2 names )

		tags, err = tqb.FindByNames(names, false)
		if err != nil {
			t.Errorf("Error finding tags: %s", err.Error())
		}
		assert.Len(t, tags, 2) // tagIdxWithScene and tagIdx1WithScene
		assert.Equal(t, tagNames[tagIdxWithScene], tags[0].Name)
		assert.Equal(t, tagNames[tagIdx1WithScene], tags[1].Name)

		tags, err = tqb.FindByNames(names, true) // find tags by names ( 2 names nocase)
		if err != nil {
			t.Errorf("Error finding tags: %s", err.Error())
		}
		assert.Len(t, tags, 4) // tagIdxWithScene and tagIdxWithDupName , tagIdx1WithScene and tagIdx1WithDupName
		assert.Equal(t, tagNames[tagIdxWithScene], tags[0].Name)
		assert.Equal(t, tagNames[tagIdx1WithScene], tags[1].Name)
		assert.Equal(t, tagNames[tagIdx1WithDupName], tags[2].Name)
		assert.Equal(t, tagNames[tagIdxWithDupName], tags[3].Name)

		return nil
	})
}

func TestTagQueryName(t *testing.T) {
	const tagIdx = 1
	tagName := getSceneStringValue(tagIdx, "Name")

	nameCriterion := &models.StringCriterionInput{
		Value:    tagName,
		Modifier: models.CriterionModifierEquals,
	}

	tagFilter := &models.TagFilterType{
		Name: nameCriterion,
	}

	verifyFn := func(tag *models.Tag, r models.Repository) {
		verifyString(t, tag.Name, *nameCriterion)
	}

	verifyTagQuery(t, tagFilter, nil, verifyFn)

	nameCriterion.Modifier = models.CriterionModifierNotEquals
	verifyTagQuery(t, tagFilter, nil, verifyFn)

	nameCriterion.Modifier = models.CriterionModifierMatchesRegex
	nameCriterion.Value = "tag_.*1_Name"
	verifyTagQuery(t, tagFilter, nil, verifyFn)

	nameCriterion.Modifier = models.CriterionModifierNotMatchesRegex
	verifyTagQuery(t, tagFilter, nil, verifyFn)
}

func TestTagQueryAlias(t *testing.T) {
	const tagIdx = 1
	tagName := getSceneStringValue(tagIdx, "Alias")

	aliasCriterion := &models.StringCriterionInput{
		Value:    tagName,
		Modifier: models.CriterionModifierEquals,
	}

	tagFilter := &models.TagFilterType{
		Aliases: aliasCriterion,
	}

	verifyFn := func(tag *models.Tag, r models.Repository) {
		aliases, err := r.Tag().GetAliases(tag.ID)
		if err != nil {
			t.Errorf("Error querying tags: %s", err.Error())
		}

		var alias string
		if len(aliases) > 0 {
			alias = aliases[0]
		}

		verifyString(t, alias, *aliasCriterion)
	}

	verifyTagQuery(t, tagFilter, nil, verifyFn)

	aliasCriterion.Modifier = models.CriterionModifierNotEquals
	verifyTagQuery(t, tagFilter, nil, verifyFn)

	aliasCriterion.Modifier = models.CriterionModifierMatchesRegex
	aliasCriterion.Value = "tag_.*1_Alias"
	verifyTagQuery(t, tagFilter, nil, verifyFn)

	aliasCriterion.Modifier = models.CriterionModifierNotMatchesRegex
	verifyTagQuery(t, tagFilter, nil, verifyFn)
}

func verifyTagQuery(t *testing.T, tagFilter *models.TagFilterType, findFilter *models.FindFilterType, verifyFn func(t *models.Tag, r models.Repository)) {
	withTxn(func(r models.Repository) error {
		sqb := r.Tag()

		tags := queryTags(t, sqb, tagFilter, findFilter)

		for _, tag := range tags {
			verifyFn(tag, r)
		}

		return nil
	})
}

func queryTags(t *testing.T, qb models.TagReader, tagFilter *models.TagFilterType, findFilter *models.FindFilterType) []*models.Tag {
	t.Helper()
	tags, _, err := qb.Query(tagFilter, findFilter)
	if err != nil {
		t.Errorf("Error querying tags: %s", err.Error())
	}

	return tags
}

func TestTagQueryIsMissingImage(t *testing.T) {
	withTxn(func(r models.Repository) error {
		qb := r.Tag()
		isMissing := "image"
		tagFilter := models.TagFilterType{
			IsMissing: &isMissing,
		}

		q := getTagStringValue(tagIdxWithCoverImage, "name")
		findFilter := models.FindFilterType{
			Q: &q,
		}

		tags, _, err := qb.Query(&tagFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying tag: %s", err.Error())
		}

		assert.Len(t, tags, 0)

		findFilter.Q = nil
		tags, _, err = qb.Query(&tagFilter, &findFilter)
		if err != nil {
			t.Errorf("Error querying tag: %s", err.Error())
		}

		// ensure non of the ids equal the one with image
		for _, tag := range tags {
			assert.NotEqual(t, tagIDs[tagIdxWithCoverImage], tag.ID)
		}

		return nil
	})
}

func TestTagQuerySceneCount(t *testing.T) {
	countCriterion := models.IntCriterionInput{
		Exact:    1,
		Modifier: models.CriterionModifierEquals,
	}

	verifyTagSceneCount(t, countCriterion)

	countCriterion.Modifier = models.CriterionModifierNotEquals
	verifyTagSceneCount(t, countCriterion)

	countCriterion.Modifier = models.CriterionModifierLessThan
	verifyTagSceneCount(t, countCriterion)

	countCriterion.Exact = 0
	countCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyTagSceneCount(t, countCriterion)
}

func verifyTagSceneCount(t *testing.T, sceneCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		qb := r.Tag()
		tagFilter := models.TagFilterType{
			SceneCount: &sceneCountCriterion,
		}

		tags, _, err := qb.Query(&tagFilter, nil)
		if err != nil {
			t.Errorf("Error querying tag: %s", err.Error())
		}

		for _, tag := range tags {
			verifyInt64(t, sql.NullInt64{
				Int64: int64(getTagSceneCount(tag.ID)),
				Valid: true,
			}, sceneCountCriterion)
		}

		return nil
	})
}

// disabled due to performance issues

// func TestTagQueryMarkerCount(t *testing.T) {
// 	countCriterion := models.IntCriterionInput{
// 		Exact:    1,
// 		Modifier: models.CriterionModifierEquals,
// 	}

// 	verifyTagMarkerCount(t, countCriterion)

// 	countCriterion.Modifier = models.CriterionModifierNotEquals
// 	verifyTagMarkerCount(t, countCriterion)

// 	countCriterion.Modifier = models.CriterionModifierLessThan
// 	verifyTagMarkerCount(t, countCriterion)

// 	countCriterion.Value = 0
// 	countCriterion.Modifier = models.CriterionModifierGreaterThan
// 	verifyTagMarkerCount(t, countCriterion)
// }

func verifyTagMarkerCount(t *testing.T, markerCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		qb := r.Tag()
		tagFilter := models.TagFilterType{
			MarkerCount: &markerCountCriterion,
		}

		tags, _, err := qb.Query(&tagFilter, nil)
		if err != nil {
			t.Errorf("Error querying tag: %s", err.Error())
		}

		for _, tag := range tags {
			verifyInt64(t, sql.NullInt64{
				Int64: int64(getTagMarkerCount(tag.ID)),
				Valid: true,
			}, markerCountCriterion)
		}

		return nil
	})
}

func TestTagQueryImageCount(t *testing.T) {
	countCriterion := models.IntCriterionInput{
		Exact:    1,
		Modifier: models.CriterionModifierEquals,
	}

	verifyTagImageCount(t, countCriterion)

	countCriterion.Modifier = models.CriterionModifierNotEquals
	verifyTagImageCount(t, countCriterion)

	countCriterion.Modifier = models.CriterionModifierLessThan
	verifyTagImageCount(t, countCriterion)

	countCriterion.Exact = 0
	countCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyTagImageCount(t, countCriterion)
}

func verifyTagImageCount(t *testing.T, imageCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		qb := r.Tag()
		tagFilter := models.TagFilterType{
			ImageCount: &imageCountCriterion,
		}

		tags, _, err := qb.Query(&tagFilter, nil)
		if err != nil {
			t.Errorf("Error querying tag: %s", err.Error())
		}

		for _, tag := range tags {
			verifyInt64(t, sql.NullInt64{
				Int64: int64(getTagImageCount(tag.ID)),
				Valid: true,
			}, imageCountCriterion)
		}

		return nil
	})
}

func TestTagQueryGalleryCount(t *testing.T) {
	countCriterion := models.IntCriterionInput{
		Exact:    1,
		Modifier: models.CriterionModifierEquals,
	}

	verifyTagGalleryCount(t, countCriterion)

	countCriterion.Modifier = models.CriterionModifierNotEquals
	verifyTagGalleryCount(t, countCriterion)

	countCriterion.Modifier = models.CriterionModifierLessThan
	verifyTagGalleryCount(t, countCriterion)

	countCriterion.Exact = 0
	countCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyTagGalleryCount(t, countCriterion)
}

func verifyTagGalleryCount(t *testing.T, imageCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		qb := r.Tag()
		tagFilter := models.TagFilterType{
			GalleryCount: &imageCountCriterion,
		}

		tags, _, err := qb.Query(&tagFilter, nil)
		if err != nil {
			t.Errorf("Error querying tag: %s", err.Error())
		}

		for _, tag := range tags {
			verifyInt64(t, sql.NullInt64{
				Int64: int64(getTagGalleryCount(tag.ID)),
				Valid: true,
			}, imageCountCriterion)
		}

		return nil
	})
}

func TestTagQueryPerformerCount(t *testing.T) {
	countCriterion := models.IntCriterionInput{
		Exact:    1,
		Modifier: models.CriterionModifierEquals,
	}

	verifyTagPerformerCount(t, countCriterion)

	countCriterion.Modifier = models.CriterionModifierNotEquals
	verifyTagPerformerCount(t, countCriterion)

	countCriterion.Modifier = models.CriterionModifierLessThan
	verifyTagPerformerCount(t, countCriterion)

	countCriterion.Exact = 0
	countCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyTagPerformerCount(t, countCriterion)
}

func verifyTagPerformerCount(t *testing.T, imageCountCriterion models.IntCriterionInput) {
	withTxn(func(r models.Repository) error {
		qb := r.Tag()
		tagFilter := models.TagFilterType{
			PerformerCount: &imageCountCriterion,
		}

		tags, _, err := qb.Query(&tagFilter, nil)
		if err != nil {
			t.Errorf("Error querying tag: %s", err.Error())
		}

		for _, tag := range tags {
			verifyInt64(t, sql.NullInt64{
				Int64: int64(getTagPerformerCount(tag.ID)),
				Valid: true,
			}, imageCountCriterion)
		}

		return nil
	})
}

func TestTagUpdateTagImage(t *testing.T) {
	if err := withTxn(func(r models.Repository) error {
		qb := r.Tag()

		// create tag to test against
		const name = "TestTagUpdateTagImage"
		tag := models.Tag{
			Name: name,
		}
		created, err := qb.Create(tag)
		if err != nil {
			return fmt.Errorf("Error creating tag: %s", err.Error())
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

func TestTagDestroyTagImage(t *testing.T) {
	if err := withTxn(func(r models.Repository) error {
		qb := r.Tag()

		// create performer to test against
		const name = "TestTagDestroyTagImage"
		tag := models.Tag{
			Name: name,
		}
		created, err := qb.Create(tag)
		if err != nil {
			return fmt.Errorf("Error creating tag: %s", err.Error())
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

func TestTagUpdateAlias(t *testing.T) {
	if err := withTxn(func(r models.Repository) error {
		qb := r.Tag()

		// create tag to test against
		const name = "TestTagUpdateAlias"
		tag := models.Tag{
			Name: name,
		}
		created, err := qb.Create(tag)
		if err != nil {
			return fmt.Errorf("Error creating tag: %s", err.Error())
		}

		aliases := []string{"alias1", "alias2"}
		err = qb.UpdateAliases(created.ID, aliases)
		if err != nil {
			return fmt.Errorf("Error updating tag aliases: %s", err.Error())
		}

		// ensure aliases set
		storedAliases, err := qb.GetAliases(created.ID)
		if err != nil {
			return fmt.Errorf("Error getting aliases: %s", err.Error())
		}
		assert.Equal(t, aliases, storedAliases)

		return nil
	}); err != nil {
		t.Error(err.Error())
	}
}

func TestTagMerge(t *testing.T) {
	assert := assert.New(t)

	// merge tests - perform these in a transaction that we'll rollback
	if err := withRollbackTxn(func(r models.Repository) error {
		qb := r.Tag()

		// try merging into same tag
		err := qb.Merge([]int{tagIDs[tagIdx1WithScene]}, tagIDs[tagIdx1WithScene])
		assert.NotNil(err)

		// merge everything into tagIdxWithScene
		srcIdxs := []int{
			tagIdx1WithScene,
			tagIdx2WithScene,
			tagIdxWithPrimaryMarker,
			tagIdxWithMarker,
			tagIdxWithCoverImage,
			tagIdxWithImage,
			tagIdx1WithImage,
			tagIdx2WithImage,
			tagIdxWithPerformer,
			tagIdx1WithPerformer,
			tagIdx2WithPerformer,
			tagIdxWithGallery,
			tagIdx1WithGallery,
			tagIdx2WithGallery,
		}
		var srcIDs []int
		for _, idx := range srcIdxs {
			srcIDs = append(srcIDs, tagIDs[idx])
		}

		destID := tagIDs[tagIdxWithScene]
		if err = qb.Merge(srcIDs, destID); err != nil {
			return err
		}

		// ensure other tags are deleted
		for _, tagId := range srcIDs {
			t, err := qb.Find(tagId)
			if err != nil {
				return err
			}

			assert.Nil(t)
		}

		// ensure aliases are set on the destination
		destAliases, err := qb.GetAliases(destID)
		if err != nil {
			return err
		}
		for _, tagIdx := range srcIdxs {
			assert.Contains(destAliases, getTagStringValue(tagIdx, "Name"))
		}

		// ensure scene points to new tag
		sceneTagIDs, err := r.Scene().GetTagIDs(sceneIDs[sceneIdxWithTwoTags])
		if err != nil {
			return err
		}

		assert.Contains(sceneTagIDs, destID)

		// ensure marker points to new tag
		marker, err := r.SceneMarker().Find(markerIDs[markerIdxWithScene])
		if err != nil {
			return err
		}

		assert.Equal(destID, marker.PrimaryTagID)

		markerTagIDs, err := r.SceneMarker().GetTagIDs(marker.ID)
		if err != nil {
			return err
		}

		assert.Contains(markerTagIDs, destID)

		// ensure image points to new tag
		imageTagIDs, err := r.Image().GetTagIDs(imageIDs[imageIdxWithTwoTags])
		if err != nil {
			return err
		}

		assert.Contains(imageTagIDs, destID)

		// ensure gallery points to new tag
		galleryTagIDs, err := r.Gallery().GetTagIDs(galleryIDs[galleryIdxWithTwoTags])
		if err != nil {
			return err
		}

		assert.Contains(galleryTagIDs, destID)

		// ensure performer points to new tag
		performerTagIDs, err := r.Gallery().GetTagIDs(performerIDs[performerIdxWithTwoTags])
		if err != nil {
			return err
		}

		assert.Contains(performerTagIDs, destID)

		return nil
	}); err != nil {
		t.Error(err.Error())
	}
}

// TODO Create
// TODO Update
// TODO Destroy
// TODO Find
// TODO FindBySceneID
// TODO FindBySceneMarkerID
// TODO Count
// TODO All
// TODO AllSlim
// TODO Query

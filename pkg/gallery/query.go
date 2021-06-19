package gallery

import (
	"strconv"

	"github.com/stashapp/stash/pkg/models"
)

func CountByPerformerID(r models.GalleryReader, id int) (int, error) {
	filter := &models.GalleryFilterType{
		Performers: &models.MultiCriterionInput{
			Value:    []string{strconv.Itoa(id)},
			Modifier: models.CriterionModifierIncludes,
		},
	}

	return r.QueryCount(filter, nil)
}

func CountByStudioID(r models.GalleryReader, id int) (int, error) {
	filter := &models.GalleryFilterType{
		Studios: &models.HierarchicalMultiCriterionInput{
			Value:    []string{strconv.Itoa(id)},
			Modifier: models.CriterionModifierIncludes,
			Depth:    0,
		},
	}

	return r.QueryCount(filter, nil)
}

func CountByTagID(r models.GalleryReader, id int) (int, error) {
	filter := &models.GalleryFilterType{
		Tags: &models.HierarchicalMultiCriterionInput{
			Value:    []string{strconv.Itoa(id)},
			Modifier: models.CriterionModifierIncludes,
			Depth:    0,
		},
	}

	return r.QueryCount(filter, nil)
}

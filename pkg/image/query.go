package image

import (
	"strconv"

	"github.com/stashapp/stash/pkg/models"
)

func CountByPerformerID(r models.ImageReader, id int) (int, error) {
	filter := &models.ImageFilterType{
		Performers: &models.MultiCriterionInput{
			Value:    []string{strconv.Itoa(id)},
			Modifier: models.CriterionModifierIncludes,
		},
	}

	return r.QueryCount(filter, nil)
}

func CountByStudioID(r models.ImageReader, id int) (int, error) {
	filter := &models.ImageFilterType{
		Studios: &models.HierarchicalMultiCriterionInput{
			Value:    []string{strconv.Itoa(id)},
			Modifier: models.CriterionModifierIncludes,
			Depth:    0,
		},
	}

	return r.QueryCount(filter, nil)
}

func CountByTagID(r models.ImageReader, id int) (int, error) {
	filter := &models.ImageFilterType{
		Tags: &models.HierarchicalMultiCriterionInput{
			Value:    []string{strconv.Itoa(id)},
			Modifier: models.CriterionModifierIncludes,
			Depth:    0,
		},
	}

	return r.QueryCount(filter, nil)
}

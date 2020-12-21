package api

import (
	"context"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
)

func (r *mutationResolver) AddSceneError(ctx context.Context, input models.SceneErrorInput) (*models.SceneError, error) {
	var err error
	sceneID := -1

	if input.SceneID != nil {
		sceneID, err = strconv.Atoi(*input.SceneID)
		if err != nil {
			sceneID = -1
		}
	}

	relatedSceneID := -1
	if input.RelatedSceneID != nil {
		relatedSceneID, err = strconv.Atoi(*input.RelatedSceneID)
		if err != nil {
			relatedSceneID = -1
		}
	}

	recurring := ""
	if input.Recurring != nil {
		recurring = *input.Recurring
	}

	details := ""
	if input.Details != nil {
		details = *input.Details
	}

	return models.PushFullSceneError(sceneID, input.ErrorType, recurring, details, relatedSceneID)
}

func (r *mutationResolver) ClearRecurringSceneErrors(ctx context.Context, recurringType string) (bool, error) {
	qb := models.NewSceneErrorQueryBuilder()
	err := qb.ClearRecurringErrors(recurringType)
	if err != nil {
		return false, err
	}
	return true, nil
}

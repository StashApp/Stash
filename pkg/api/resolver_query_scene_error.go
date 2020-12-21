package api

import (
	"context"

	"github.com/stashapp/stash/pkg/models"
)

func (r *queryResolver) GetSceneErrors(ctx context.Context) ([]*models.SceneError, error) {
	qb := models.NewSceneErrorQueryBuilder()
	sceneErrors, err := qb.All()

	if err != nil {
		return nil, err
	}

	return sceneErrors, nil
}

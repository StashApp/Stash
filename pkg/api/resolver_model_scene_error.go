package api

import (
	"context"

	"github.com/stashapp/stash/pkg/models"
)

func (r *sceneErrorResolver) Scene(ctx context.Context, obj *models.SceneError) (*models.Scene, error) {
	if !obj.SceneID.Valid {
		return nil, nil
	}
	sceneID := int(obj.SceneID.Int64)
	qb := models.NewSceneQueryBuilder()
	scene, err := qb.Find(sceneID)
	return scene, err
}

func (r *sceneErrorResolver) RelatedScene(ctx context.Context, obj *models.SceneError) (*models.Scene, error) {
	if !obj.RelatedSceneID.Valid {
		return nil, nil
	}
	sceneID := int(obj.RelatedSceneID.Int64)
	qb := models.NewSceneQueryBuilder()
	scene, err := qb.Find(sceneID)
	return scene, err
}

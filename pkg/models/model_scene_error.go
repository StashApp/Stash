package models

import (
	"database/sql"
)

type SceneError struct {
	SceneID        sql.NullInt64 `db:"scene_id" json:"scene_id"`
	Recurring      string        `db:"recurring" json:"recurring"`
	ErrorType      string        `db:"error_type" json:"error_type"`
	Details        string        `db:"details" json:"details"`
	RelatedSceneID sql.NullInt64 `db:"related_scene_id" json:"related_scene_id"`
}

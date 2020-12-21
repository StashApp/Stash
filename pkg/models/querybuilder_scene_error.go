package models

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/stashapp/stash/pkg/database"
)

const sceneErrorTable = "scene_errors"

type SceneErrorQueryBuilder struct{}

func NewSceneErrorQueryBuilder() SceneErrorQueryBuilder {
	return SceneErrorQueryBuilder{}
}

func (qb *SceneErrorQueryBuilder) Create(newSceneError SceneError) (*SceneError, error) {
	_, err := database.DB.NamedExec(
		`INSERT INTO scene_errors (scene_id, error_type, recurring, details, related_scene_id)
				VALUES (:scene_id, :error_type, :recurring, :details, :related_scene_id)
		`,
		newSceneError,
	)
	if err != nil {
		return nil, err
	}

	return &newSceneError, nil
}

func (qb *SceneErrorQueryBuilder) ClearRecurringErrors(recurringType string) error {
	_, err := database.DB.Exec("DELETE FROM scene_errors WHERE recurring = ?", recurringType)
	if err != nil {
		return err
	}
	return nil
}

func (qb *SceneErrorQueryBuilder) ClearErrors(sceneID int, errorType string) error {
	_, err := database.DB.Exec("DELETE FROM scene_errors WHERE scene_errors.scene_id = ? AND error_type = ?", sceneID, errorType)
	if err != nil {
		return err
	}
	return nil
}

func (qb *SceneErrorQueryBuilder) All() ([]*SceneError, error) {
	return qb.querySceneErrors(selectAll("scene_errors"), nil)
}

func (qb *SceneErrorQueryBuilder) querySceneErrors(query string, args []interface{}) ([]*SceneError, error) {
	var rows *sqlx.Rows
	var err error
	rows, err = database.DB.Queryx(query, args...)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	defer rows.Close()

	sceneErrors := make([]*SceneError, 0)
	for rows.Next() {
		sceneError := SceneError{}
		if err := rows.StructScan(&sceneError); err != nil {
			return nil, err
		}
		sceneErrors = append(sceneErrors, &sceneError)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sceneErrors, nil
}

func SetSceneError(sceneID int, errorType string, recurring string, details string) (*SceneError, error) {
	err := RemoveSceneError(sceneID, errorType)
	if err != nil {
		return nil, err
	}
	return PushFullSceneError(sceneID, errorType, recurring, details, -1)
}

func PushSceneError(sceneID int, errorType string, recurring string, details string) (*SceneError, error) {
	return PushFullSceneError(sceneID, errorType, recurring, details, -1)
}

func PushFullSceneError(sceneID int, errorType string, recurring string, details string, relatedSceneID int) (*SceneError, error) {
	truncatedDetails := details
	if len(truncatedDetails) > 254 {
		truncatedDetails = details[:254]
	}

	id := sql.NullInt64{
		Valid: sceneID != -1,
		Int64: int64(sceneID),
	}

	relatedID := sql.NullInt64{
		Valid: relatedSceneID != -1,
		Int64: int64(relatedSceneID),
	}

	sceneError := SceneError{
		SceneID:        id,
		ErrorType:      errorType,
		Recurring:      recurring,
		Details:        truncatedDetails,
		RelatedSceneID: relatedID,
	}

	qb := NewSceneErrorQueryBuilder()
	return qb.Create(sceneError)
}

func RemoveSceneError(sceneID int, errorType string) error {
	qb := NewSceneErrorQueryBuilder()
	return qb.ClearErrors(sceneID, errorType)
}

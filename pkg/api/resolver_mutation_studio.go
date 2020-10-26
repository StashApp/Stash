package api

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/stashapp/stash/pkg/database"
	"github.com/stashapp/stash/pkg/manager"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

func (r *mutationResolver) StudioCreate(ctx context.Context, input models.StudioCreateInput) (*models.Studio, error) {
	// generate checksum from studio name rather than image
	checksum := utils.MD5FromString(input.Name)

	var imageData []byte
	var err error

	// Process the base 64 encoded image string
	if input.Image != nil {
		_, imageData, err = utils.ProcessBase64Image(*input.Image)
		if err != nil {
			return nil, err
		}
	}

	// Populate a new studio from the input
	currentTime := time.Now()
	newStudio := models.Studio{
		Checksum:  checksum,
		Name:      sql.NullString{String: input.Name, Valid: true},
		CreatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
		UpdatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
	}
	if input.URL != nil {
		newStudio.URL = sql.NullString{String: *input.URL, Valid: true}
	}
	if input.ParentID != nil {
		parentID, _ := strconv.ParseInt(*input.ParentID, 10, 64)
		newStudio.ParentID = sql.NullInt64{Int64: parentID, Valid: true}
	}

	// Start the transaction and save the studio
	tx := database.DB.MustBeginTx(ctx, nil)
	qb := models.NewStudioQueryBuilder()
	jqb := models.NewJoinsQueryBuilder()

	studio, err := qb.Create(newStudio, tx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	// update image table
	if len(imageData) > 0 {
		if err := qb.UpdateStudioImage(studio.ID, imageData, tx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}

	// Save the stash_ids
	if input.StashIds != nil {
		var stashIDJoins []models.StashID
		for _, stashID := range input.StashIds {
			newJoin := models.StashID{
				StashID:  stashID.StashID,
				Endpoint: stashID.Endpoint,
			}
			stashIDJoins = append(stashIDJoins, newJoin)
		}
		if err := jqb.UpdateStudioStashIDs(studio.ID, stashIDJoins, tx); err != nil {
			return nil, err
		}
	}

	// Commit
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return studio, nil
}

func (r *mutationResolver) StudioUpdate(ctx context.Context, input models.StudioUpdateInput) (*models.Studio, error) {
	// Populate studio from the input
	studioID, _ := strconv.Atoi(input.ID)

	updatedStudio := models.StudioPartial{
		ID:        studioID,
		UpdatedAt: &models.SQLiteTimestamp{Timestamp: time.Now()},
	}

	var imageData []byte
	imageIncluded := wasFieldIncluded(ctx, "image")
	if input.Image != nil {
		var err error
		_, imageData, err = utils.ProcessBase64Image(*input.Image)
		if err != nil {
			return nil, err
		}
	}
	if input.Name != nil {
		// generate checksum from studio name rather than image
		checksum := utils.MD5FromString(*input.Name)
		updatedStudio.Name = &sql.NullString{String: *input.Name, Valid: true}
		updatedStudio.Checksum = &checksum
	}
	if input.URL != nil {
		updatedStudio.URL = &sql.NullString{String: *input.URL, Valid: true}
	}

	if input.ParentID != nil {
		parentID, _ := strconv.ParseInt(*input.ParentID, 10, 64)
		updatedStudio.ParentID = &sql.NullInt64{Int64: parentID, Valid: true}
	} else {
		// parent studio must be nullable
		updatedStudio.ParentID = &sql.NullInt64{Valid: false}
	}

	// Start the transaction and save the studio
	tx := database.DB.MustBeginTx(ctx, nil)
	qb := models.NewStudioQueryBuilder()
	jqb := models.NewJoinsQueryBuilder()

	if err := manager.ValidateModifyStudio(updatedStudio, tx); err != nil {
		tx.Rollback()
		return nil, err
	}

	studio, err := qb.Update(updatedStudio, tx)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// update image table
	if len(imageData) > 0 {
		if err := qb.UpdateStudioImage(studio.ID, imageData, tx); err != nil {
			tx.Rollback()
			return nil, err
		}
	} else if imageIncluded {
		// must be unsetting
		if err := qb.DestroyStudioImage(studio.ID, tx); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Save the stash_ids
	if input.StashIds != nil {
		var stashIDJoins []models.StashID
		for _, stashID := range input.StashIds {
			newJoin := models.StashID{
				StashID:  stashID.StashID,
				Endpoint: stashID.Endpoint,
			}
			stashIDJoins = append(stashIDJoins, newJoin)
		}
		if err := jqb.UpdateStudioStashIDs(studioID, stashIDJoins, tx); err != nil {
			return nil, err
		}
	}

	// Commit
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return studio, nil
}

func (r *mutationResolver) StudioDestroy(ctx context.Context, input models.StudioDestroyInput) (bool, error) {
	qb := models.NewStudioQueryBuilder()
	tx := database.DB.MustBeginTx(ctx, nil)
	if err := qb.Destroy(input.ID, tx); err != nil {
		_ = tx.Rollback()
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

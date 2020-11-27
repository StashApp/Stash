package api

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/stashapp/stash/pkg/database"
	"github.com/stashapp/stash/pkg/manager"
	"github.com/stashapp/stash/pkg/models"
)

func (r *mutationResolver) ImageUpdate(ctx context.Context, input models.ImageUpdateInput) (*models.Image, error) {
	// Start the transaction and save the image
	tx := database.DB.MustBeginTx(ctx, nil)

	ret, err := r.imageUpdate(input, tx)

	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	// Commit
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) ImagesUpdate(ctx context.Context, input []*models.ImageUpdateInput) ([]*models.Image, error) {
	// Start the transaction and save the image
	tx := database.DB.MustBeginTx(ctx, nil)

	var ret []*models.Image

	for _, image := range input {
		thisImage, err := r.imageUpdate(*image, tx)
		ret = append(ret, thisImage)

		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}

	// Commit
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) imageUpdate(input models.ImageUpdateInput, tx *sqlx.Tx) (*models.Image, error) {
	// Populate image from the input
	imageID, _ := strconv.Atoi(input.ID)

	updatedTime := time.Now()
	updatedImage := models.ImagePartial{
		ID:        imageID,
		UpdatedAt: &models.SQLiteTimestamp{Timestamp: updatedTime},
	}
	if input.Title != nil {
		updatedImage.Title = &sql.NullString{String: *input.Title, Valid: true}
	}

	if input.Rating != nil {
		updatedImage.Rating = &sql.NullInt64{Int64: int64(*input.Rating), Valid: true}
	} else {
		// rating must be nullable
		updatedImage.Rating = &sql.NullInt64{Valid: false}
	}

	if input.StudioID != nil {
		studioID, _ := strconv.ParseInt(*input.StudioID, 10, 64)
		updatedImage.StudioID = &sql.NullInt64{Int64: studioID, Valid: true}
	} else {
		// studio must be nullable
		updatedImage.StudioID = &sql.NullInt64{Valid: false}
	}

	qb := models.NewImageQueryBuilder()
	jqb := models.NewJoinsQueryBuilder()
	image, err := qb.Update(updatedImage, tx)
	if err != nil {
		return nil, err
	}

	// don't set the galleries directly. Use add/remove gallery images interface instead

	// Save the performers
	var performerJoins []models.PerformersImages
	for _, pid := range input.PerformerIds {
		performerID, _ := strconv.Atoi(pid)
		performerJoin := models.PerformersImages{
			PerformerID: performerID,
			ImageID:     imageID,
		}
		performerJoins = append(performerJoins, performerJoin)
	}
	if err := jqb.UpdatePerformersImages(imageID, performerJoins, tx); err != nil {
		return nil, err
	}

	// Save the tags
	var tagJoins []models.ImagesTags
	for _, tid := range input.TagIds {
		tagID, _ := strconv.Atoi(tid)
		tagJoin := models.ImagesTags{
			ImageID: imageID,
			TagID:   tagID,
		}
		tagJoins = append(tagJoins, tagJoin)
	}
	if err := jqb.UpdateImagesTags(imageID, tagJoins, tx); err != nil {
		return nil, err
	}

	return image, nil
}

func (r *mutationResolver) BulkImageUpdate(ctx context.Context, input models.BulkImageUpdateInput) ([]*models.Image, error) {
	// Populate image from the input
	updatedTime := time.Now()

	// Start the transaction and save the image marker
	tx := database.DB.MustBeginTx(ctx, nil)
	qb := models.NewImageQueryBuilder()
	jqb := models.NewJoinsQueryBuilder()

	updatedImage := models.ImagePartial{
		UpdatedAt: &models.SQLiteTimestamp{Timestamp: updatedTime},
	}
	if input.Title != nil {
		updatedImage.Title = &sql.NullString{String: *input.Title, Valid: true}
	}
	if input.Rating != nil {
		// a rating of 0 means unset the rating
		if *input.Rating == 0 {
			updatedImage.Rating = &sql.NullInt64{Int64: 0, Valid: false}
		} else {
			updatedImage.Rating = &sql.NullInt64{Int64: int64(*input.Rating), Valid: true}
		}
	}
	if input.StudioID != nil {
		// empty string means unset the studio
		if *input.StudioID == "" {
			updatedImage.StudioID = &sql.NullInt64{Int64: 0, Valid: false}
		} else {
			studioID, _ := strconv.ParseInt(*input.StudioID, 10, 64)
			updatedImage.StudioID = &sql.NullInt64{Int64: studioID, Valid: true}
		}
	}

	ret := []*models.Image{}

	for _, imageIDStr := range input.Ids {
		imageID, _ := strconv.Atoi(imageIDStr)
		updatedImage.ID = imageID

		image, err := qb.Update(updatedImage, tx)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}

		ret = append(ret, image)

		// Save the galleries
		if wasFieldIncluded(ctx, "gallery_ids") {
			galleryIDs, err := adjustImageGalleryIDs(tx, imageID, *input.GalleryIds)
			if err != nil {
				_ = tx.Rollback()
				return nil, err
			}

			var galleryJoins []models.GalleriesImages
			for _, gid := range galleryIDs {
				galleryJoin := models.GalleriesImages{
					GalleryID: gid,
					ImageID:   imageID,
				}
				galleryJoins = append(galleryJoins, galleryJoin)
			}
			if err := jqb.UpdateGalleriesImages(imageID, galleryJoins, tx); err != nil {
				return nil, err
			}
		}

		// Save the performers
		if wasFieldIncluded(ctx, "performer_ids") {
			performerIDs, err := adjustImagePerformerIDs(tx, imageID, *input.PerformerIds)
			if err != nil {
				_ = tx.Rollback()
				return nil, err
			}

			var performerJoins []models.PerformersImages
			for _, performerID := range performerIDs {
				performerJoin := models.PerformersImages{
					PerformerID: performerID,
					ImageID:     imageID,
				}
				performerJoins = append(performerJoins, performerJoin)
			}
			if err := jqb.UpdatePerformersImages(imageID, performerJoins, tx); err != nil {
				_ = tx.Rollback()
				return nil, err
			}
		}

		// Save the tags
		if wasFieldIncluded(ctx, "tag_ids") {
			tagIDs, err := adjustImageTagIDs(tx, imageID, *input.TagIds)
			if err != nil {
				_ = tx.Rollback()
				return nil, err
			}

			var tagJoins []models.ImagesTags
			for _, tagID := range tagIDs {
				tagJoin := models.ImagesTags{
					ImageID: imageID,
					TagID:   tagID,
				}
				tagJoins = append(tagJoins, tagJoin)
			}
			if err := jqb.UpdateImagesTags(imageID, tagJoins, tx); err != nil {
				_ = tx.Rollback()
				return nil, err
			}
		}
	}

	// Commit
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return ret, nil
}

func adjustImageGalleryIDs(tx *sqlx.Tx, imageID int, ids models.BulkUpdateIds) ([]int, error) {
	var ret []int

	jqb := models.NewJoinsQueryBuilder()
	if ids.Mode == models.BulkUpdateIDModeAdd || ids.Mode == models.BulkUpdateIDModeRemove {
		// adding to the joins
		galleryJoins, err := jqb.GetImageGalleries(imageID, tx)

		if err != nil {
			return nil, err
		}

		for _, join := range galleryJoins {
			ret = append(ret, join.GalleryID)
		}
	}

	return adjustIDs(ret, ids), nil
}

func adjustImagePerformerIDs(tx *sqlx.Tx, imageID int, ids models.BulkUpdateIds) ([]int, error) {
	var ret []int

	jqb := models.NewJoinsQueryBuilder()
	if ids.Mode == models.BulkUpdateIDModeAdd || ids.Mode == models.BulkUpdateIDModeRemove {
		// adding to the joins
		performerJoins, err := jqb.GetImagePerformers(imageID, tx)

		if err != nil {
			return nil, err
		}

		for _, join := range performerJoins {
			ret = append(ret, join.PerformerID)
		}
	}

	return adjustIDs(ret, ids), nil
}

func adjustImageTagIDs(tx *sqlx.Tx, imageID int, ids models.BulkUpdateIds) ([]int, error) {
	var ret []int

	jqb := models.NewJoinsQueryBuilder()
	if ids.Mode == models.BulkUpdateIDModeAdd || ids.Mode == models.BulkUpdateIDModeRemove {
		// adding to the joins
		tagJoins, err := jqb.GetImageTags(imageID, tx)

		if err != nil {
			return nil, err
		}

		for _, join := range tagJoins {
			ret = append(ret, join.TagID)
		}
	}

	return adjustIDs(ret, ids), nil
}

func (r *mutationResolver) ImageDestroy(ctx context.Context, input models.ImageDestroyInput) (bool, error) {
	qb := models.NewImageQueryBuilder()
	tx := database.DB.MustBeginTx(ctx, nil)

	imageID, _ := strconv.Atoi(input.ID)
	image, err := qb.Find(imageID)
	err = qb.Destroy(imageID, tx)

	if err != nil {
		tx.Rollback()
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}

	// if delete generated is true, then delete the generated files
	// for the image
	if input.DeleteGenerated != nil && *input.DeleteGenerated {
		manager.DeleteGeneratedImageFiles(image)
	}

	// if delete file is true, then delete the file as well
	// if it fails, just log a message
	if input.DeleteFile != nil && *input.DeleteFile {
		manager.DeleteImageFile(image)
	}

	return true, nil
}

func (r *mutationResolver) ImagesDestroy(ctx context.Context, input models.ImagesDestroyInput) (bool, error) {
	qb := models.NewImageQueryBuilder()
	tx := database.DB.MustBeginTx(ctx, nil)

	var images []*models.Image
	for _, id := range input.Ids {
		imageID, _ := strconv.Atoi(id)

		image, err := qb.Find(imageID)
		if image != nil {
			images = append(images, image)
		}
		err = qb.Destroy(imageID, tx)

		if err != nil {
			tx.Rollback()
			return false, err
		}
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}

	for _, image := range images {
		// if delete generated is true, then delete the generated files
		// for the image
		if input.DeleteGenerated != nil && *input.DeleteGenerated {
			manager.DeleteGeneratedImageFiles(image)
		}

		// if delete file is true, then delete the file as well
		// if it fails, just log a message
		if input.DeleteFile != nil && *input.DeleteFile {
			manager.DeleteImageFile(image)
		}
	}

	return true, nil
}

func (r *mutationResolver) ImageIncrementO(ctx context.Context, id string) (int, error) {
	imageID, _ := strconv.Atoi(id)

	tx := database.DB.MustBeginTx(ctx, nil)
	qb := models.NewImageQueryBuilder()

	newVal, err := qb.IncrementOCounter(imageID, tx)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}

	// Commit
	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return newVal, nil
}

func (r *mutationResolver) ImageDecrementO(ctx context.Context, id string) (int, error) {
	imageID, _ := strconv.Atoi(id)

	tx := database.DB.MustBeginTx(ctx, nil)
	qb := models.NewImageQueryBuilder()

	newVal, err := qb.DecrementOCounter(imageID, tx)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}

	// Commit
	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return newVal, nil
}

func (r *mutationResolver) ImageResetO(ctx context.Context, id string) (int, error) {
	imageID, _ := strconv.Atoi(id)

	tx := database.DB.MustBeginTx(ctx, nil)
	qb := models.NewImageQueryBuilder()

	newVal, err := qb.ResetOCounter(imageID, tx)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}

	// Commit
	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return newVal, nil
}

package api

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin"
	"github.com/stashapp/stash/pkg/tag"
	"github.com/stashapp/stash/pkg/utils"
)

func (r *mutationResolver) getTag(ctx context.Context, id int) (ret *models.Tag, err error) {
	if err := r.withReadTxn(ctx, func(repo models.ReaderRepository) error {
		ret, err = repo.Tag().Find(id)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) TagCreate(ctx context.Context, input models.TagCreateInput) (*models.Tag, error) {
	// Populate a new tag from the input
	currentTime := time.Now()
	newTag := models.Tag{
		Name:      input.Name,
		CreatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
		UpdatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
	}

	var imageData []byte
	var err error

	if input.Image != nil {
		imageData, err = utils.ProcessImageInput(*input.Image)

		if err != nil {
			return nil, err
		}
	}

	// Start the transaction and save the t
	var t *models.Tag
	if err := r.withTxn(ctx, func(repo models.Repository) error {
		qb := repo.Tag()

		// ensure name is unique
		if err := tag.EnsureTagNameUnique(0, newTag.Name, qb); err != nil {
			return err
		}

		t, err = qb.Create(newTag)
		if err != nil {
			return err
		}

		// update image table
		if len(imageData) > 0 {
			if err := qb.UpdateImage(t.ID, imageData); err != nil {
				return err
			}
		}

		if len(input.Aliases) > 0 {
			if err := tag.EnsureAliasesUnique(t.ID, input.Aliases, qb); err != nil {
				return err
			}

			if err := qb.UpdateAliases(t.ID, input.Aliases); err != nil {
				return err
			}
		}

		if input.ParentIds != nil && len(input.ParentIds) > 0 {
			ids, err := utils.StringSliceToIntSlice(input.ParentIds)
			if err != nil {
				return err
			}

			if err := qb.UpdateParentTags(t.ID, ids); err != nil {
				return err
			}
		}

		if input.ChildIds != nil && len(input.ChildIds) > 0 {
			ids, err := utils.StringSliceToIntSlice(input.ChildIds)
			if err != nil {
				return err
			}

			if err := qb.UpdateChildTags(t.ID, ids); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	r.hookExecutor.ExecutePostHooks(ctx, t.ID, plugin.TagCreatePost, input, nil)
	return r.getTag(ctx, t.ID)
}

func (r *mutationResolver) TagUpdate(ctx context.Context, input models.TagUpdateInput) (*models.Tag, error) {
	// Populate tag from the input
	tagID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, err
	}

	var imageData []byte

	translator := changesetTranslator{
		inputMap: getUpdateInputMap(ctx),
	}

	imageIncluded := translator.hasField("image")
	if input.Image != nil {
		imageData, err = utils.ProcessImageInput(*input.Image)

		if err != nil {
			return nil, err
		}
	}

	// Start the transaction and save the tag
	var t *models.Tag
	if err := r.withTxn(ctx, func(repo models.Repository) error {
		qb := repo.Tag()

		// ensure name is unique
		t, err = qb.Find(tagID)
		if err != nil {
			return err
		}

		if t == nil {
			return fmt.Errorf("Tag with ID %d not found", tagID)
		}

		updatedTag := models.TagPartial{
			ID:        tagID,
			UpdatedAt: &models.SQLiteTimestamp{Timestamp: time.Now()},
		}

		if input.Name != nil && t.Name != *input.Name {
			if err := tag.EnsureTagNameUnique(tagID, *input.Name, qb); err != nil {
				return err
			}

			updatedTag.Name = input.Name
		}

		t, err = qb.Update(updatedTag)
		if err != nil {
			return err
		}

		// update image table
		if len(imageData) > 0 {
			if err := qb.UpdateImage(tagID, imageData); err != nil {
				return err
			}
		} else if imageIncluded {
			// must be unsetting
			if err := qb.DestroyImage(tagID); err != nil {
				return err
			}
		}

		if translator.hasField("aliases") {
			if err := tag.EnsureAliasesUnique(tagID, input.Aliases, qb); err != nil {
				return err
			}

			if err := qb.UpdateAliases(tagID, input.Aliases); err != nil {
				return err
			}
		}

		var parentIDs []int
		var childIDs []int

		if translator.hasField("parent_ids") {
			parentIDs, err = utils.StringSliceToIntSlice(input.ParentIds)
			if err != nil {
				return err
			}
		}

		if translator.hasField("child_ids") {
			childIDs, err = utils.StringSliceToIntSlice(input.ChildIds)
			if err != nil {
				return err
			}
		}

		if parentIDs != nil || childIDs != nil {
			if err := tag.EnsureUniqueHierarchy(tagID, parentIDs, childIDs, qb); err != nil {
				return err
			}
		}

		if parentIDs != nil {
			if err := qb.UpdateParentTags(tagID, parentIDs); err != nil {
				return err
			}
		}

		if childIDs != nil {
			if err := qb.UpdateChildTags(tagID, childIDs); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	r.hookExecutor.ExecutePostHooks(ctx, t.ID, plugin.TagUpdatePost, input, translator.getFields())
	return r.getTag(ctx, t.ID)
}

func (r *mutationResolver) TagDestroy(ctx context.Context, input models.TagDestroyInput) (bool, error) {
	tagID, err := strconv.Atoi(input.ID)
	if err != nil {
		return false, err
	}

	if err := r.withTxn(ctx, func(repo models.Repository) error {
		return repo.Tag().Destroy(tagID)
	}); err != nil {
		return false, err
	}

	r.hookExecutor.ExecutePostHooks(ctx, tagID, plugin.TagDestroyPost, input, nil)

	return true, nil
}

func (r *mutationResolver) TagsDestroy(ctx context.Context, tagIDs []string) (bool, error) {
	ids, err := utils.StringSliceToIntSlice(tagIDs)
	if err != nil {
		return false, err
	}

	if err := r.withTxn(ctx, func(repo models.Repository) error {
		qb := repo.Tag()
		for _, id := range ids {
			if err := qb.Destroy(id); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return false, err
	}

	for _, id := range ids {
		r.hookExecutor.ExecutePostHooks(ctx, id, plugin.TagDestroyPost, tagIDs, nil)
	}

	return true, nil
}

func (r *mutationResolver) TagsMerge(ctx context.Context, input models.TagsMergeInput) (*models.Tag, error) {
	source, err := utils.StringSliceToIntSlice(input.Source)
	if err != nil {
		return nil, err
	}

	destination, err := strconv.Atoi(input.Destination)
	if err != nil {
		return nil, err
	}

	if len(source) == 0 {
		return nil, nil
	}

	var t *models.Tag
	if err := r.withTxn(ctx, func(repo models.Repository) error {
		qb := repo.Tag()

		var err error
		t, err = qb.Find(destination)
		if err != nil {
			return err
		}

		if t == nil {
			return fmt.Errorf("Tag with ID %d not found", destination)
		}

		if err = qb.Merge(source, destination); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return t, nil
}

package api

import (
	"context"
	"time"

	"github.com/stashapp/stash/pkg/api/urlbuilders"
	"github.com/stashapp/stash/pkg/gallery"
	"github.com/stashapp/stash/pkg/image"
	"github.com/stashapp/stash/pkg/models"
)

func (r *tagResolver) Parents(ctx context.Context, obj *models.Tag) (ret []*models.Tag, err error) {
	if err := r.withReadTxn(ctx, func(repo models.ReaderRepository) error {
		ret, err = repo.Tag().FindByChildTagID(obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *tagResolver) Children(ctx context.Context, obj *models.Tag) (ret []*models.Tag, err error) {
	if err := r.withReadTxn(ctx, func(repo models.ReaderRepository) error {
		ret, err = repo.Tag().FindByParentTagID(obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *tagResolver) Aliases(ctx context.Context, obj *models.Tag) (ret []string, err error) {
	if err := r.withReadTxn(ctx, func(repo models.ReaderRepository) error {
		ret, err = repo.Tag().GetAliases(obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, err
}

func (r *tagResolver) SceneCount(ctx context.Context, obj *models.Tag) (ret *int, err error) {
	var count int
	if err := r.withReadTxn(ctx, func(repo models.ReaderRepository) error {
		count, err = repo.Scene().CountByTagID(obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	return &count, err
}

func (r *tagResolver) SceneMarkerCount(ctx context.Context, obj *models.Tag) (ret *int, err error) {
	var count int
	if err := r.withReadTxn(ctx, func(repo models.ReaderRepository) error {
		count, err = repo.SceneMarker().CountByTagID(obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	return &count, err
}

func (r *tagResolver) ImageCount(ctx context.Context, obj *models.Tag) (ret *int, err error) {
	var res int
	if err := r.withReadTxn(ctx, func(repo models.ReaderRepository) error {
		res, err = image.CountByTagID(repo.Image(), obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	return &res, nil
}

func (r *tagResolver) GalleryCount(ctx context.Context, obj *models.Tag) (ret *int, err error) {
	var res int
	if err := r.withReadTxn(ctx, func(repo models.ReaderRepository) error {
		res, err = gallery.CountByTagID(repo.Gallery(), obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	return &res, nil
}

func (r *tagResolver) PerformerCount(ctx context.Context, obj *models.Tag) (ret *int, err error) {
	var count int
	if err := r.withReadTxn(ctx, func(repo models.ReaderRepository) error {
		count, err = repo.Performer().CountByTagID(obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	return &count, err
}

func (r *tagResolver) ImagePath(ctx context.Context, obj *models.Tag) (*string, error) {
	baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
	imagePath := urlbuilders.NewTagURLBuilder(baseURL, obj).GetTagImageURL()
	return &imagePath, nil
}

func (r *tagResolver) CreatedAt(ctx context.Context, obj *models.Tag) (*time.Time, error) {
	return &obj.CreatedAt.Timestamp, nil
}

func (r *tagResolver) UpdatedAt(ctx context.Context, obj *models.Tag) (*time.Time, error) {
	return &obj.UpdatedAt.Timestamp, nil
}

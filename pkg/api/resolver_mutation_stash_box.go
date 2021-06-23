package api

import (
	"context"
	"fmt"
	"strconv"

	"github.com/stashapp/stash/pkg/manager"
	"github.com/stashapp/stash/pkg/manager/config"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scraper/stashbox"
)

func (r *mutationResolver) SubmitStashBoxFingerprints(ctx context.Context, input models.StashBoxFingerprintSubmissionInput) (bool, error) {
	boxes := config.GetInstance().GetStashBoxes()

	if input.StashBoxIndex < 0 || input.StashBoxIndex >= len(boxes) {
		return false, fmt.Errorf("invalid stash_box_index %d", input.StashBoxIndex)
	}

	client := stashbox.NewClient(*boxes[input.StashBoxIndex], r.txnManager)

	return client.SubmitStashBoxFingerprints(input.SceneIds, boxes[input.StashBoxIndex].Endpoint)
}

func (r *mutationResolver) StashBoxBatchPerformerTag(ctx context.Context, input models.StashBoxBatchPerformerTagInput) (string, error) {
	jobID := manager.GetInstance().StashBoxBatchPerformerTag(ctx, input)
	return strconv.Itoa(jobID), nil
}

func (r *mutationResolver) StashBoxBatchSceneTag(ctx context.Context, input models.StashBoxBatchSceneTagInput) (string, error) {
	jobID := manager.GetInstance().StashBoxBatchSceneTag(ctx, input)
	return strconv.Itoa(jobID), nil
}

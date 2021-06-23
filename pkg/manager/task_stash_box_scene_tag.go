package manager

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scraper/stashbox"
	"github.com/stashapp/stash/pkg/utils"
)

type StashBoxSceneTagTask struct {
	txnManager        models.TransactionManager
	box               *models.StashBox
	scene             *models.Scene
	refresh           bool
	excludedFields    []string
	phashDistance     int
	setOrganized      bool
	tagStrategy       models.TagStrategy
	createTags        bool
	tagMalePerformers bool
}

func (t *StashBoxSceneTagTask) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	t.stashBoxSceneTag()
}

func (t *StashBoxSceneTagTask) Description() string {
	name := "Unknown"
	if t.scene != nil && t.scene.Title.Valid {
		name = t.scene.Title.String
	}

	return fmt.Sprintf("Tagging scene '%s' from stash-box", name)
}

func (t *StashBoxSceneTagTask) stashBoxSceneTag() {
	var scenes []*models.ScrapedScene
	var err error

	client := stashbox.NewClient(*t.box, t.txnManager)

	if t.refresh {
		var sceneID string
		t.txnManager.WithReadTxn(context.TODO(), func(r models.ReaderRepository) error {
			stashids, _ := r.Scene().GetStashIDs(t.scene.ID)
			for _, id := range stashids {
				if id.Endpoint == t.box.Endpoint {
					sceneID = id.StashID
				}
			}
			return nil
		})
		if sceneID != "" {
			scene, err := client.FindStashBoxSceneByID(sceneID)
			if err == nil && scene != nil {
				scenes = append(scenes, scene)
			}
		}
	} else {
		scenes, err = client.FindStashBoxScenesByFingerprints([]int{
			t.scene.ID,
		})
	}

	if err != nil {
		logger.Errorf("Error fetching scene data from stash-box: %s", err.Error())
		return
	}

	excluded := map[string]bool{}
	for _, field := range t.excludedFields {
		excluded[field] = true
	}

	if len(scenes) > 1 {
		logger.Errorf("Multiple (%d) results found for scene %d, skipping.", len(scenes), t.scene.ID)
		return
	}

	if scenes != nil {
		updatedTime := time.Now()

		scene := scenes[0]

		if t.scene != nil {
			partial := models.ScenePartial{
				ID:        t.scene.ID,
				UpdatedAt: &models.SQLiteTimestamp{Timestamp: updatedTime},
			}

			if !excluded["title"] {
				value := getNullString(scene.Title)
				partial.Title = &value
			}
			if !excluded["date"] {
				value := getDate(scene.Date)
				partial.Date = &value
			}
			// TODO
			//if scene.Urls.!= nil && !excluded["url"] {
			//value := getNullString(scene.Title)
			//partial.URL = &value
			//}
			if scene.Details != nil && *scene.Details != "" && !excluded["details"] {
				value := getNullString(scene.Details)
				partial.Details = &value
			}
			if t.setOrganized {
				organized := true
				partial.Organized = &organized
			}
			if !excluded["studio"] {
				if scene.Studio.ID != nil {
					studioID, err := strconv.Atoi(*scene.Studio.ID)
					if err != nil {
						partial.StudioID = &sql.NullInt64{Int64: int64(studioID), Valid: true}
					}
				} else {
					err := t.txnManager.WithTxn(context.TODO(), func(r models.Repository) error {
						studio := scrapedToStudioInput(scene.Studio)
						res, err := r.Studio().Create(studio)
						if err == nil && res != nil {
							logger.Tracef("Created studio %s with id %d", res.Name.String, res.ID)
							err := r.Studio().UpdateStashIDs(res.ID, []models.StashID{{
								StashID:  *scene.Studio.RemoteSiteID,
								Endpoint: t.box.Endpoint,
							}})
							if err != nil {
								return err
							}
							partial.StudioID = &sql.NullInt64{Int64: int64(res.ID), Valid: true}

							if scene.Studio.Image != nil {
								image, err := utils.ReadImageFromURL(*scene.Studio.Image)
								if err != nil {
									return err
								}
								err = r.Studio().UpdateImage(res.ID, image)
								if err != nil {
									return err
								}
							}
						}
						return nil
					})
					if err != nil {
						logger.Errorf("Error resolving studio: %s", err.Error())
						return
					}
				}
			}

			err := t.txnManager.WithTxn(context.TODO(), func(r models.Repository) error {
				_, err := r.Scene().Update(partial)

				if !t.refresh {
					err = r.Scene().UpdateStashIDs(t.scene.ID, []models.StashID{{
						Endpoint: t.box.Endpoint,
						StashID:  *scene.RemoteSiteID,
					}})
					if err != nil {
						return err
					}
				}

				if !excluded["covers"] && scene.Image != nil {
					err = r.Scene().UpdateCover(t.scene.ID, []byte(*scene.Image))
					if err != nil {
						return err
					}
				}

				if !excluded["performers"] {
					var performerIDs []int
					for _, performer := range scene.Performers {
						if performer.ID != nil {
							performerID, err := strconv.Atoi(*performer.ID)
							if err == nil {
								performerIDs = append(performerIDs, performerID)
							}
						} else if performer.Gender == nil || *performer.Gender != "MALE" || t.tagMalePerformers {
							performerInput := scrapedToPerformerInput(performer)
							res, err := r.Performer().Create(performerInput)
							if err != nil {
								return err
							}
							logger.Infof("Created performer: %s", performer.Name)

							err = r.Performer().UpdateStashIDs(res.ID, []models.StashID{{
								Endpoint: t.box.Endpoint,
								StashID:  *performer.RemoteSiteID,
							}})
							if err != nil {
								return err
							}

							if len(performer.Images) > 0 && !excluded["image"] {
								image, err := utils.ReadImageFromURL(performer.Images[0])
								if err != nil {
									return err
								}
								err = r.Performer().UpdateImage(res.ID, image)
								if err != nil {
									return err
								}
							}

							performerIDs = append(performerIDs, res.ID)
						}
					}

					err = r.Scene().UpdatePerformers(t.scene.ID, performerIDs)
					if err != nil {
						return err
					}
				}

				if t.tagStrategy != models.TagStrategyIgnore {
					var tags []int
					if t.tagStrategy == models.TagStrategyMerge {
						tags, err = r.Scene().GetTagIDs(t.scene.ID)
						if err != nil {
							return err
						}
					}

					for _, tag := range scene.Tags {
						var id *int
						if tag.ID != nil {
							parsedID, err := strconv.Atoi(*tag.ID)
							if err != nil {
								return err
							}
							id = &parsedID
						}

						if id == nil && t.createTags {
							now := time.Now()
							tag, err := r.Tag().Create(models.Tag{
								Name:      tag.Name,
								CreatedAt: models.SQLiteTimestamp{Timestamp: now},
								UpdatedAt: models.SQLiteTimestamp{Timestamp: now},
							})
							if err != nil {
								return err
							}
							logger.Infof("Created tag: %s", tag.Name)
							id = &tag.ID
						}

						if id != nil {
							exists := false
							for _, existingTag := range tags {
								if existingTag == *id {
									exists = true
								}
							}
							if !exists {
								tags = append(tags, *id)
							}
						}
					}

					err = r.Scene().UpdateTags(t.scene.ID, tags)
				}

				if err == nil {
					name := strconv.Itoa(t.scene.ID)
					if scene.Title != nil {
						name = *scene.Title
					}
					logger.Infof("Updated scene %s", name)
				}
				return err
			})
			if err != nil {
				logger.Errorf("Unable to update scene: %s", err.Error())
			}
		}
	} else if !t.refresh {
		logger.Infof("No match found for scene %d", t.scene.ID)
	}
}

func scrapedToStudioInput(studio *models.ScrapedSceneStudio) models.Studio {
	currentTime := time.Now()
	ret := models.Studio{
		Name:      sql.NullString{String: studio.Name, Valid: true},
		Checksum:  utils.MD5FromString(studio.Name),
		CreatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
		UpdatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
	}

	if studio.URL != nil {
		ret.URL = sql.NullString{String: *studio.URL, Valid: true}
	}

	return ret
}

func scrapedToPerformerInput(performer *models.ScrapedScenePerformer) models.Performer {
	currentTime := time.Now()
	ret := models.Performer{
		Name:      sql.NullString{String: performer.Name, Valid: true},
		Checksum:  utils.MD5FromString(performer.Name),
		CreatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
		UpdatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
		Favorite:  sql.NullBool{Bool: false, Valid: true},
	}
	if performer.Birthdate != nil {
		ret.Birthdate = models.SQLiteDate{String: *performer.Birthdate, Valid: true}
	}
	if performer.DeathDate != nil {
		ret.DeathDate = models.SQLiteDate{String: *performer.DeathDate, Valid: true}
	}
	if performer.Gender != nil {
		ret.Gender = sql.NullString{String: *performer.Gender, Valid: true}
	}
	if performer.Ethnicity != nil {
		ret.Ethnicity = sql.NullString{String: *performer.Ethnicity, Valid: true}
	}
	if performer.Country != nil {
		ret.Country = sql.NullString{String: *performer.Country, Valid: true}
	}
	if performer.EyeColor != nil {
		ret.EyeColor = sql.NullString{String: *performer.EyeColor, Valid: true}
	}
	if performer.HairColor != nil {
		ret.HairColor = sql.NullString{String: *performer.HairColor, Valid: true}
	}
	if performer.Height != nil {
		ret.Height = sql.NullString{String: *performer.Height, Valid: true}
	}
	if performer.Measurements != nil {
		ret.Measurements = sql.NullString{String: *performer.Measurements, Valid: true}
	}
	if performer.FakeTits != nil {
		ret.FakeTits = sql.NullString{String: *performer.FakeTits, Valid: true}
	}
	if performer.CareerLength != nil {
		ret.CareerLength = sql.NullString{String: *performer.CareerLength, Valid: true}
	}
	if performer.Tattoos != nil {
		ret.Tattoos = sql.NullString{String: *performer.Tattoos, Valid: true}
	}
	if performer.Piercings != nil {
		ret.Piercings = sql.NullString{String: *performer.Piercings, Valid: true}
	}
	if performer.Aliases != nil {
		ret.Aliases = sql.NullString{String: *performer.Aliases, Valid: true}
	}
	if performer.Twitter != nil {
		ret.Twitter = sql.NullString{String: *performer.Twitter, Valid: true}
	}
	if performer.Instagram != nil {
		ret.Instagram = sql.NullString{String: *performer.Instagram, Valid: true}
	}

	return ret
}

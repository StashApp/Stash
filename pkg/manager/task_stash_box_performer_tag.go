package manager

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scraper/stashbox"
	"github.com/stashapp/stash/pkg/utils"
)

type StashBoxPerformerTagTask struct {
	txnManager      models.TransactionManager
	box             *models.StashBox
	name            *string
	performer       *models.Performer
	refresh         bool
	excluded_fields []string
}

func (t *StashBoxPerformerTagTask) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	t.stashBoxPerformerTag()
}

func (t *StashBoxPerformerTagTask) Description() string {
	var name string
	if t.name != nil {
		name = *t.name
	} else if t.performer != nil {
		name = t.performer.Name.String
	}

	return fmt.Sprintf("Tagging performer %s from stash-box", name)
}

func (t *StashBoxPerformerTagTask) stashBoxPerformerTag() {
	var performer *models.ScrapedScenePerformer
	var err error

	client := stashbox.NewClient(*t.box, t.txnManager)

	if t.refresh {
		var performerID string
		t.txnManager.WithReadTxn(context.TODO(), func(r models.ReaderRepository) error {
			stashids, _ := r.Performer().GetStashIDs(t.performer.ID)
			for _, id := range stashids {
				if id.Endpoint == t.box.Endpoint {
					performerID = id.StashID
				}
			}
			return nil
		})
		if performerID != "" {
			performer, err = client.FindStashBoxPerformerByID(performerID)
		}
	} else {
		var name string
		if t.name != nil {
			name = *t.name
		} else {
			name = t.performer.Name.String
		}
		performer, err = client.FindStashBoxPerformerByName(name)
	}

	if err != nil {
		logger.Errorf("Error fetching performer data from stash-box: %s", err.Error())
		return
	}

	excluded := map[string]bool{}
	for _, field := range t.excluded_fields {
		excluded[field] = true
	}

	if performer != nil {
		updatedTime := time.Now()

		if t.performer != nil {
			partial := models.PerformerPartial{
				ID:        t.performer.ID,
				UpdatedAt: &models.SQLiteTimestamp{Timestamp: updatedTime},
			}

			if performer.Aliases != nil && !excluded["aliases"] {
				value := getNullString(performer.Aliases)
				partial.Aliases = &value
			}
			if performer.Birthdate != nil && *performer.Birthdate != "" && !excluded["birthdate"] {
				value := getDate(performer.Birthdate)
				partial.Birthdate = &value
			}
			if performer.CareerLength != nil && !excluded["career_length"] {
				value := getNullString(performer.CareerLength)
				partial.CareerLength = &value
			}
			if performer.Country != nil && !excluded["country"] {
				value := getNullString(performer.Country)
				partial.Country = &value
			}
			if performer.Ethnicity != nil && !excluded["ethnicity"] {
				value := getNullString(performer.Ethnicity)
				partial.Ethnicity = &value
			}
			if performer.EyeColor != nil && !excluded["eye_color"] {
				value := getNullString(performer.EyeColor)
				partial.EyeColor = &value
			}
			if performer.FakeTits != nil && !excluded["fake_tits"] {
				value := getNullString(performer.FakeTits)
				partial.FakeTits = &value
			}
			if performer.Gender != nil && !excluded["gender"] {
				value := getNullString(performer.Gender)
				partial.Gender = &value
			}
			if performer.Height != nil && !excluded["height"] {
				value := getNullString(performer.Height)
				partial.Height = &value
			}
			if performer.Instagram != nil && !excluded["instagram"] {
				value := getNullString(performer.Instagram)
				partial.Instagram = &value
			}
			if performer.Measurements != nil && !excluded["measurements"] {
				value := getNullString(performer.Measurements)
				partial.Measurements = &value
			}
			if excluded["name"] {
				value := sql.NullString{String: performer.Name, Valid: true}
				partial.Name = &value
			}
			if performer.Piercings != nil && !excluded["piercings"] {
				value := getNullString(performer.Piercings)
				partial.Piercings = &value
			}
			if performer.Tattoos != nil && !excluded["tattoos"] {
				value := getNullString(performer.Tattoos)
				partial.Tattoos = &value
			}
			if performer.Twitter != nil && !excluded["twitter"] {
				value := getNullString(performer.Tattoos)
				partial.Twitter = &value
			}
			if performer.URL != nil && !excluded["url"] {
				value := getNullString(performer.URL)
				partial.URL = &value
			}

			t.txnManager.WithTxn(context.TODO(), func(r models.Repository) error {
				_, err := r.Performer().Update(partial)

				if !t.refresh {
					err = r.Performer().UpdateStashIDs(t.performer.ID, []models.StashID{
						{
							Endpoint: t.box.Endpoint,
							StashID:  *performer.RemoteSiteID,
						},
					})
					if err != nil {
						return err
					}
				}

				if len(performer.Images) > 0 && !excluded["image"] {
					image, err := utils.ReadImageFromURL(performer.Images[0])
					if err != nil {
						return err
					}
					err = r.Performer().UpdateImage(t.performer.ID, image)
					if err != nil {
						return err
					}
				}

				if err == nil {
					logger.Infof("Updated performer %s", performer.Name)
				}
				return err
			})
		} else if t.name != nil {
			err := t.txnManager.WithTxn(context.TODO(), func(r models.Repository) error {
				return CreateScrapedPerformer(performer, t.box.Endpoint, r)
			})
			if err != nil {
				logger.Errorf("Failed to save performer %s: %s", *t.name, err.Error())
			} else {
				logger.Infof("Saved performer %s", *t.name)
			}
		}
	} else {
		var name string
		if t.name != nil {
			name = *t.name
		} else if t.performer != nil {
			name = t.performer.Name.String
		}
		logger.Infof("No match found for %s", name)
	}
}

func getDate(val *string) models.SQLiteDate {
	if val == nil {
		return models.SQLiteDate{Valid: false}
	} else {
		return models.SQLiteDate{String: *val, Valid: false}
	}
}

func getNullString(val *string) sql.NullString {
	if val == nil {
		return sql.NullString{Valid: false}
	} else {
		return sql.NullString{String: *val, Valid: true}
	}
}

func CreateScrapedPerformer(performer *models.ScrapedScenePerformer, endpoint string, r models.Repository) error {
	currentTime := time.Now()
	newPerformer := models.Performer{
		Aliases:      getNullString(performer.Aliases),
		Birthdate:    getDate(performer.Birthdate),
		CareerLength: getNullString(performer.CareerLength),
		Checksum:     utils.MD5FromString(performer.Name),
		Country:      getNullString(performer.Country),
		CreatedAt:    models.SQLiteTimestamp{Timestamp: currentTime},
		Ethnicity:    getNullString(performer.Ethnicity),
		EyeColor:     getNullString(performer.EyeColor),
		FakeTits:     getNullString(performer.FakeTits),
		Favorite:     sql.NullBool{Bool: false, Valid: true},
		Gender:       getNullString(performer.Gender),
		Height:       getNullString(performer.Height),
		Instagram:    getNullString(performer.Instagram),
		Measurements: getNullString(performer.Measurements),
		Name:         sql.NullString{String: performer.Name, Valid: true},
		Piercings:    getNullString(performer.Piercings),
		Tattoos:      getNullString(performer.Tattoos),
		Twitter:      getNullString(performer.Twitter),
		URL:          getNullString(performer.URL),
		UpdatedAt:    models.SQLiteTimestamp{Timestamp: currentTime},
	}

	createdPerformer, err := r.Performer().Create(newPerformer)
	if err != nil {
		return err
	}

	err = r.Performer().UpdateStashIDs(createdPerformer.ID, []models.StashID{
		{
			Endpoint: endpoint,
			StashID:  *performer.RemoteSiteID,
		},
	})
	if err != nil {
		return err
	}

	if len(performer.Images) > 0 {
		image, err := utils.ReadImageFromURL(performer.Images[0])
		if err != nil {
			return err
		}
		err = r.Performer().UpdateImage(createdPerformer.ID, image)
	}

	return nil
}

package performer

import (
	"fmt"

	"github.com/stashapp/stash/pkg/manager/jsonschema"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

// ToJSON converts a Performer object into its JSON equivalent.
func ToJSON(reader models.PerformerReader, performer *models.Performer) (*jsonschema.Performer, error) {
	newPerformerJSON := jsonschema.Performer{
		CreatedAt: models.JSONTime{Time: performer.CreatedAt.Timestamp},
		UpdatedAt: models.JSONTime{Time: performer.UpdatedAt.Timestamp},
	}

	if performer.Name.Valid {
		newPerformerJSON.Name = performer.Name.String
	}
	if performer.Gender.Valid {
		newPerformerJSON.Gender = performer.Gender.String
	}
	if performer.URL.Valid {
		newPerformerJSON.URL = performer.URL.String
	}
	if performer.Birthdate.Valid {
		newPerformerJSON.Birthdate = utils.GetYMDFromDatabaseDate(performer.Birthdate.String)
	}
	if performer.Ethnicity.Valid {
		newPerformerJSON.Ethnicity = performer.Ethnicity.String
	}
	if performer.Country.Valid {
		newPerformerJSON.Country = performer.Country.String
	}
	if performer.EyeColor.Valid {
		newPerformerJSON.EyeColor = performer.EyeColor.String
	}
	if performer.Height.Valid {
		newPerformerJSON.Height = performer.Height.String
	}
	if performer.Measurements.Valid {
		newPerformerJSON.Measurements = performer.Measurements.String
	}
	if performer.FakeTits.Valid {
		newPerformerJSON.FakeTits = performer.FakeTits.String
	}
	if performer.CareerLength.Valid {
		newPerformerJSON.CareerLength = performer.CareerLength.String
	}
	if performer.Tattoos.Valid {
		newPerformerJSON.Tattoos = performer.Tattoos.String
	}
	if performer.Piercings.Valid {
		newPerformerJSON.Piercings = performer.Piercings.String
	}
	if performer.Aliases.Valid {
		newPerformerJSON.Aliases = performer.Aliases.String
	}
	if performer.Twitter.Valid {
		newPerformerJSON.Twitter = performer.Twitter.String
	}
	if performer.Instagram.Valid {
		newPerformerJSON.Instagram = performer.Instagram.String
	}
	if performer.Favorite.Valid {
		newPerformerJSON.Favorite = performer.Favorite.Bool
	}

	image, err := reader.GetPerformerImage(performer.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting performers image: %s", err.Error())
	}

	if len(image) > 0 {
		newPerformerJSON.Image = utils.GetBase64StringFromData(image)
	}

	return &newPerformerJSON, nil
}

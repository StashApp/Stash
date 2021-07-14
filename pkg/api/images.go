package api

import (
	"math/rand"
	"strings"

	"github.com/gobuffalo/packr/v2"
	"github.com/stashapp/stash/pkg/utils"
)

var performerBox *packr.Box
var performerBoxMale *packr.Box
var performerBoxCustom *packr.Box

func initialiseImages() {
	performerBox = packr.New("Performer Box", "../../static/performer")
	performerBoxMale = packr.New("Male Performer Box", "../../static/performer_male")
}

func getRandomPerformerImage(gender string) ([]byte, error) {
	var box *packr.Box
	switch strings.ToUpper(gender) {
	case "FEMALE":
		box = performerBox
	case "MALE":
		box = performerBoxMale
	default:
		box = performerBox

	}
	imageFiles := box.List()
	index := rand.Intn(len(imageFiles))
	return box.Find(imageFiles[index])
}

func getRandomPerformerImageUsingName(name, gender, customPath string) ([]byte, error) {
	var box *packr.Box

	// If we have a custom path, we should return a new box in the given path.
	// TODO: Should we check the validity of a custom path?
	if customPath != "" {
		if performerBoxCustom != nil {
			box = performerBoxCustom
		} else {
			// We need to set performerBoxCustom at runtime, as this is a custom path, and store it in a pointer.
			performerBoxCustom = packr.New("Custom Performer Box", customPath)
			box = performerBoxCustom
		}
	} else {
		
		switch strings.ToUpper(gender) {
		case "FEMALE":
			box = performerBox
		case "MALE":
			box = performerBoxMale
		default:
			box = performerBoxMale
		}
	}

	imageFiles := box.List()
	index := utils.IntFromString(name) % uint64(len(imageFiles))
	return box.Find(imageFiles[index])
}

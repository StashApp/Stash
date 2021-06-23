package models

import (
	"database/sql"
)

type ScrapedItem struct {
	ID              int             `db:"id" json:"id"`
	Title           sql.NullString  `db:"title" json:"title"`
	Description     sql.NullString  `db:"description" json:"description"`
	URL             sql.NullString  `db:"url" json:"url"`
	Date            SQLiteDate      `db:"date" json:"date"`
	Rating          sql.NullString  `db:"rating" json:"rating"`
	Tags            sql.NullString  `db:"tags" json:"tags"`
	Models          sql.NullString  `db:"models" json:"models"`
	Episode         sql.NullInt64   `db:"episode" json:"episode"`
	GalleryFilename sql.NullString  `db:"gallery_filename" json:"gallery_filename"`
	GalleryURL      sql.NullString  `db:"gallery_url" json:"gallery_url"`
	VideoFilename   sql.NullString  `db:"video_filename" json:"video_filename"`
	VideoURL        sql.NullString  `db:"video_url" json:"video_url"`
	StudioID        sql.NullInt64   `db:"studio_id,omitempty" json:"studio_id"`
	CreatedAt       SQLiteTimestamp `db:"created_at" json:"created_at"`
	UpdatedAt       SQLiteTimestamp `db:"updated_at" json:"updated_at"`
}

type ScrapedPerformer struct {
	Name         *string            `graphql:"name" json:"name"`
	Gender       *string            `graphql:"gender" json:"gender"`
	URL          *string            `graphql:"url" json:"url"`
	Twitter      *string            `graphql:"twitter" json:"twitter"`
	Instagram    *string            `graphql:"instagram" json:"instagram"`
	Birthdate    *string            `graphql:"birthdate" json:"birthdate"`
	Ethnicity    *string            `graphql:"ethnicity" json:"ethnicity"`
	Country      *string            `graphql:"country" json:"country"`
	EyeColor     *string            `graphql:"eye_color" json:"eye_color"`
	Height       *string            `graphql:"height" json:"height"`
	Measurements *string            `graphql:"measurements" json:"measurements"`
	FakeTits     *string            `graphql:"fake_tits" json:"fake_tits"`
	CareerLength *string            `graphql:"career_length" json:"career_length"`
	Tattoos      *string            `graphql:"tattoos" json:"tattoos"`
	Piercings    *string            `graphql:"piercings" json:"piercings"`
	Aliases      *string            `graphql:"aliases" json:"aliases"`
	Tags         []*ScrapedSceneTag `graphql:"tags" json:"tags"`
	Image        *string            `graphql:"image" json:"image"`
	Details      *string            `graphql:"details" json:"details"`
	DeathDate    *string            `graphql:"death_date" json:"death_date"`
	HairColor    *string            `graphql:"hair_color" json:"hair_color"`
	Weight       *string            `graphql:"weight" json:"weight"`
}

// this type has no Image field
type ScrapedPerformerStash struct {
	Name         *string            `graphql:"name" json:"name"`
	Gender       *string            `graphql:"gender" json:"gender"`
	URL          *string            `graphql:"url" json:"url"`
	Twitter      *string            `graphql:"twitter" json:"twitter"`
	Instagram    *string            `graphql:"instagram" json:"instagram"`
	Birthdate    *string            `graphql:"birthdate" json:"birthdate"`
	Ethnicity    *string            `graphql:"ethnicity" json:"ethnicity"`
	Country      *string            `graphql:"country" json:"country"`
	EyeColor     *string            `graphql:"eye_color" json:"eye_color"`
	Height       *string            `graphql:"height" json:"height"`
	Measurements *string            `graphql:"measurements" json:"measurements"`
	FakeTits     *string            `graphql:"fake_tits" json:"fake_tits"`
	CareerLength *string            `graphql:"career_length" json:"career_length"`
	Tattoos      *string            `graphql:"tattoos" json:"tattoos"`
	Piercings    *string            `graphql:"piercings" json:"piercings"`
	Aliases      *string            `graphql:"aliases" json:"aliases"`
	Tags         []*ScrapedSceneTag `graphql:"tags" json:"tags"`
	Details      *string            `graphql:"details" json:"details"`
	DeathDate    *string            `graphql:"death_date" json:"death_date"`
	HairColor    *string            `graphql:"hair_color" json:"hair_color"`
	Weight       *string            `graphql:"weight" json:"weight"`
}

type ScrapedScene struct {
	Title        *string                  `graphql:"title" json:"title"`
	Details      *string                  `graphql:"details" json:"details"`
	URL          *string                  `graphql:"url" json:"url"`
	Date         *string                  `graphql:"date" json:"date"`
	Image        *string                  `graphql:"image" json:"image"`
	RemoteSiteID *string                  `graphql:"remote_site_id" json:"remote_site_id"`
	Duration     *int                     `graphql:"duration" json:"duration"`
	File         *SceneFileType           `graphql:"file" json:"file"`
	Fingerprints []*StashBoxFingerprint   `graphql:"fingerprints" json:"fingerprints"`
	Studio       *ScrapedSceneStudio      `graphql:"studio" json:"studio"`
	Movies       []*ScrapedSceneMovie     `graphql:"movies" json:"movies"`
	Tags         []*ScrapedSceneTag       `graphql:"tags" json:"tags"`
	Performers   []*ScrapedScenePerformer `graphql:"performers" json:"performers"`
}

// stash doesn't return image, and we need id
type ScrapedSceneStash struct {
	ID         string                   `graphql:"id" json:"id"`
	Title      *string                  `graphql:"title" json:"title"`
	Details    *string                  `graphql:"details" json:"details"`
	URL        *string                  `graphql:"url" json:"url"`
	Date       *string                  `graphql:"date" json:"date"`
	File       *SceneFileType           `graphql:"file" json:"file"`
	Studio     *ScrapedSceneStudio      `graphql:"studio" json:"studio"`
	Tags       []*ScrapedSceneTag       `graphql:"tags" json:"tags"`
	Performers []*ScrapedScenePerformer `graphql:"performers" json:"performers"`
}

type ScrapedGalleryStash struct {
	ID         string                   `graphql:"id" json:"id"`
	Title      *string                  `graphql:"title" json:"title"`
	Details    *string                  `graphql:"details" json:"details"`
	URL        *string                  `graphql:"url" json:"url"`
	Date       *string                  `graphql:"date" json:"date"`
	File       *SceneFileType           `graphql:"file" json:"file"`
	Studio     *ScrapedSceneStudio      `graphql:"studio" json:"studio"`
	Tags       []*ScrapedSceneTag       `graphql:"tags" json:"tags"`
	Performers []*ScrapedScenePerformer `graphql:"performers" json:"performers"`
}

type ScrapedScenePerformer struct {
	// Set if performer matched
	ID           *string            `graphql:"id" json:"id"`
	Name         string             `graphql:"name" json:"name"`
	Gender       *string            `graphql:"gender" json:"gender"`
	URL          *string            `graphql:"url" json:"url"`
	Twitter      *string            `graphql:"twitter" json:"twitter"`
	Instagram    *string            `graphql:"instagram" json:"instagram"`
	Birthdate    *string            `graphql:"birthdate" json:"birthdate"`
	Ethnicity    *string            `graphql:"ethnicity" json:"ethnicity"`
	Country      *string            `graphql:"country" json:"country"`
	EyeColor     *string            `graphql:"eye_color" json:"eye_color"`
	Height       *string            `graphql:"height" json:"height"`
	Measurements *string            `graphql:"measurements" json:"measurements"`
	FakeTits     *string            `graphql:"fake_tits" json:"fake_tits"`
	CareerLength *string            `graphql:"career_length" json:"career_length"`
	Tattoos      *string            `graphql:"tattoos" json:"tattoos"`
	Piercings    *string            `graphql:"piercings" json:"piercings"`
	Aliases      *string            `graphql:"aliases" json:"aliases"`
	Tags         []*ScrapedSceneTag `graphql:"tags" json:"tags"`
	RemoteSiteID *string            `graphql:"remote_site_id" json:"remote_site_id"`
	Images       []string           `graphql:"images" json:"images"`
	Details      *string            `graphql:"details" json:"details"`
	DeathDate    *string            `graphql:"death_date" json:"death_date"`
	HairColor    *string            `graphql:"hair_color" json:"hair_color"`
	Weight       *string            `graphql:"weight" json:"weight"`
}

type ScrapedSceneStudio struct {
	// Set if studio matched
	ID           *string `graphql:"id" json:"id"`
	Name         string  `graphql:"name" json:"name"`
	URL          *string `graphql:"url" json:"url"`
	RemoteSiteID *string `graphql:"remote_site_id" json:"remote_site_id"`
	Image        *string `graphql:"image" json:"image"`
}

type ScrapedSceneMovie struct {
	// Set if movie matched
	ID       *string `graphql:"id" json:"id"`
	Name     string  `graphql:"name" json:"name"`
	Aliases  string  `graphql:"aliases" json:"aliases"`
	Duration string  `graphql:"duration" json:"duration"`
	Date     string  `graphql:"date" json:"date"`
	Rating   string  `graphql:"rating" json:"rating"`
	Director string  `graphql:"director" json:"director"`
	Synopsis string  `graphql:"synopsis" json:"synopsis"`
	URL      *string `graphql:"url" json:"url"`
}

type ScrapedSceneTag struct {
	// Set if tag matched
	ID   *string `graphql:"stored_id" json:"stored_id"`
	Name string  `graphql:"name" json:"name"`
}

type ScrapedMovie struct {
	Name       *string             `graphql:"name" json:"name"`
	Aliases    *string             `graphql:"aliases" json:"aliases"`
	Duration   *string             `graphql:"duration" json:"duration"`
	Date       *string             `graphql:"date" json:"date"`
	Rating     *string             `graphql:"rating" json:"rating"`
	Director   *string             `graphql:"director" json:"director"`
	Studio     *ScrapedMovieStudio `graphql:"studio" json:"studio"`
	Synopsis   *string             `graphql:"synopsis" json:"synopsis"`
	URL        *string             `graphql:"url" json:"url"`
	FrontImage *string             `graphql:"front_image" json:"front_image"`
	BackImage  *string             `graphql:"back_image" json:"back_image"`
}

type ScrapedMovieStudio struct {
	// Set if studio matched
	ID   *string `graphql:"id" json:"id"`
	Name string  `graphql:"name" json:"name"`
	URL  *string `graphql:"url" json:"url"`
}

type ScrapedItems []*ScrapedItem

func (s *ScrapedItems) Append(o interface{}) {
	*s = append(*s, o.(*ScrapedItem))
}

func (s *ScrapedItems) New() interface{} {
	return &ScrapedItem{}
}

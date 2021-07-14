// +build integration

package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stashapp/stash/pkg/database"
	"github.com/stashapp/stash/pkg/gallery"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scene"
	"github.com/stashapp/stash/pkg/sqlite"
	"github.com/stashapp/stash/pkg/utils"
)

const (
	spacedSceneTitle = "zzz yyy xxx"
)

const (
	sceneIdxWithMovie = iota
	sceneIdxWithGallery
	sceneIdxWithPerformer
	sceneIdx1WithPerformer
	sceneIdx2WithPerformer
	sceneIdxWithTwoPerformers
	sceneIdxWithTag
	sceneIdxWithTwoTags
	sceneIdxWithStudio
	sceneIdx1WithStudio
	sceneIdx2WithStudio
	sceneIdxWithMarker
	sceneIdxWithPerformerTag
	sceneIdxWithPerformerTwoTags
	sceneIdxWithSpacedName
	sceneIdxWithStudioPerformer
	sceneIdxWithGrandChildStudio
	// new indexes above
	lastSceneIdx

	totalScenes = lastSceneIdx + 3
)

const (
	imageIdxWithGallery = iota
	imageIdx1WithGallery
	imageIdx2WithGallery
	imageIdxWithTwoGalleries
	imageIdxWithPerformer
	imageIdx1WithPerformer
	imageIdx2WithPerformer
	imageIdxWithTwoPerformers
	imageIdxWithTag
	imageIdxWithTwoTags
	imageIdxWithStudio
	imageIdx1WithStudio
	imageIdx2WithStudio
	imageIdxWithStudioPerformer
	imageIdxInZip // TODO - not implemented
	imageIdxWithPerformerTag
	imageIdxWithPerformerTwoTags
	imageIdxWithGrandChildStudio
	// new indexes above
	totalImages
)

const (
	performerIdxWithScene = iota
	performerIdx1WithScene
	performerIdx2WithScene
	performerIdxWithTwoScenes
	performerIdxWithImage
	performerIdxWithTwoImages
	performerIdx1WithImage
	performerIdx2WithImage
	performerIdxWithTag
	performerIdxWithTwoTags
	performerIdxWithGallery
	performerIdxWithTwoGalleries
	performerIdx1WithGallery
	performerIdx2WithGallery
	performerIdxWithSceneStudio
	performerIdxWithImageStudio
	performerIdxWithGalleryStudio
	// new indexes above
	// performers with dup names start from the end
	performerIdx1WithDupName
	performerIdxWithDupName

	performersNameCase   = performerIdx1WithDupName
	performersNameNoCase = 2
)

const (
	movieIdxWithScene = iota
	movieIdxWithStudio
	// movies with dup names start from the end
	// create 10 more basic movies (can remove this if we add more indexes)
	movieIdxWithDupName = movieIdxWithStudio + 10

	moviesNameCase   = movieIdxWithDupName
	moviesNameNoCase = 1
)

const (
	galleryIdxWithScene = iota
	galleryIdxWithImage
	galleryIdx1WithImage
	galleryIdx2WithImage
	galleryIdxWithTwoImages
	galleryIdxWithPerformer
	galleryIdx1WithPerformer
	galleryIdx2WithPerformer
	galleryIdxWithTwoPerformers
	galleryIdxWithTag
	galleryIdxWithTwoTags
	galleryIdxWithStudio
	galleryIdx1WithStudio
	galleryIdx2WithStudio
	galleryIdxWithPerformerTag
	galleryIdxWithPerformerTwoTags
	galleryIdxWithStudioPerformer
	galleryIdxWithGrandChildStudio
	// new indexes above
	lastGalleryIdx

	totalGalleries = lastGalleryIdx + 1
)

const (
	tagIdxWithScene = iota
	tagIdx1WithScene
	tagIdx2WithScene
	tagIdxWithPrimaryMarker
	tagIdxWithMarker
	tagIdxWithCoverImage
	tagIdxWithImage
	tagIdx1WithImage
	tagIdx2WithImage
	tagIdxWithPerformer
	tagIdx1WithPerformer
	tagIdx2WithPerformer
	tagIdxWithGallery
	tagIdx1WithGallery
	tagIdx2WithGallery
	// new indexes above
	// tags with dup names start from the end
	tagIdx1WithDupName
	tagIdxWithDupName

	tagsNameNoCase = 2
	tagsNameCase   = tagIdx1WithDupName
)

const (
	studioIdxWithScene = iota
	studioIdxWithTwoScenes
	studioIdxWithMovie
	studioIdxWithChildStudio
	studioIdxWithParentStudio
	studioIdxWithImage
	studioIdxWithTwoImages
	studioIdxWithGallery
	studioIdxWithTwoGalleries
	studioIdxWithScenePerformer
	studioIdxWithImagePerformer
	studioIdxWithGalleryPerformer
	studioIdxWithGrandChild
	studioIdxWithParentAndChild
	studioIdxWithGrandParent
	// new indexes above
	// studios with dup names start from the end
	studioIdxWithDupName

	studiosNameCase   = studioIdxWithDupName
	studiosNameNoCase = 1
)

const (
	markerIdxWithScene = iota
)

const (
	savedFilterIdxDefaultScene = iota
	savedFilterIdxDefaultImage
	savedFilterIdxScene
	savedFilterIdxImage

	// new indexes above
	totalSavedFilters
)

const (
	pathField            = "Path"
	checksumField        = "Checksum"
	titleField           = "Title"
	urlField             = "URL"
	zipPath              = "zipPath.zip"
	firstSavedFilterName = "firstSavedFilterName"
)

var (
	sceneIDs       []int
	imageIDs       []int
	performerIDs   []int
	movieIDs       []int
	galleryIDs     []int
	tagIDs         []int
	studioIDs      []int
	markerIDs      []int
	savedFilterIDs []int

	tagNames       []string
	studioNames    []string
	movieNames     []string
	performerNames []string
)

type idAssociation struct {
	first  int
	second int
}

var (
	sceneTagLinks = [][2]int{
		{sceneIdxWithTag, tagIdxWithScene},
		{sceneIdxWithTwoTags, tagIdx1WithScene},
		{sceneIdxWithTwoTags, tagIdx2WithScene},
	}

	scenePerformerLinks = [][2]int{
		{sceneIdxWithPerformer, performerIdxWithScene},
		{sceneIdxWithTwoPerformers, performerIdx1WithScene},
		{sceneIdxWithTwoPerformers, performerIdx2WithScene},
		{sceneIdxWithPerformerTag, performerIdxWithTag},
		{sceneIdxWithPerformerTwoTags, performerIdxWithTwoTags},
		{sceneIdx1WithPerformer, performerIdxWithTwoScenes},
		{sceneIdx2WithPerformer, performerIdxWithTwoScenes},
		{sceneIdxWithStudioPerformer, performerIdxWithSceneStudio},
	}

	sceneGalleryLinks = [][2]int{
		{sceneIdxWithGallery, galleryIdxWithScene},
	}

	sceneMovieLinks = [][2]int{
		{sceneIdxWithMovie, movieIdxWithScene},
	}

	sceneStudioLinks = [][2]int{
		{sceneIdxWithStudio, studioIdxWithScene},
		{sceneIdx1WithStudio, studioIdxWithTwoScenes},
		{sceneIdx2WithStudio, studioIdxWithTwoScenes},
		{sceneIdxWithStudioPerformer, studioIdxWithScenePerformer},
		{sceneIdxWithGrandChildStudio, studioIdxWithGrandParent},
	}
)

var (
	imageGalleryLinks = [][2]int{
		{imageIdxWithGallery, galleryIdxWithImage},
		{imageIdx1WithGallery, galleryIdxWithTwoImages},
		{imageIdx2WithGallery, galleryIdxWithTwoImages},
		{imageIdxWithTwoGalleries, galleryIdx1WithImage},
		{imageIdxWithTwoGalleries, galleryIdx2WithImage},
	}
	imageStudioLinks = [][2]int{
		{imageIdxWithStudio, studioIdxWithImage},
		{imageIdx1WithStudio, studioIdxWithTwoImages},
		{imageIdx2WithStudio, studioIdxWithTwoImages},
		{imageIdxWithStudioPerformer, studioIdxWithImagePerformer},
		{imageIdxWithGrandChildStudio, studioIdxWithGrandParent},
	}
	imageTagLinks = [][2]int{
		{imageIdxWithTag, tagIdxWithImage},
		{imageIdxWithTwoTags, tagIdx1WithImage},
		{imageIdxWithTwoTags, tagIdx2WithImage},
	}
	imagePerformerLinks = [][2]int{
		{imageIdxWithPerformer, performerIdxWithImage},
		{imageIdxWithTwoPerformers, performerIdx1WithImage},
		{imageIdxWithTwoPerformers, performerIdx2WithImage},
		{imageIdxWithPerformerTag, performerIdxWithTag},
		{imageIdxWithPerformerTwoTags, performerIdxWithTwoTags},
		{imageIdx1WithPerformer, performerIdxWithTwoImages},
		{imageIdx2WithPerformer, performerIdxWithTwoImages},
		{imageIdxWithStudioPerformer, performerIdxWithImageStudio},
	}
)

var (
	galleryPerformerLinks = [][2]int{
		{galleryIdxWithPerformer, performerIdxWithGallery},
		{galleryIdxWithTwoPerformers, performerIdx1WithGallery},
		{galleryIdxWithTwoPerformers, performerIdx2WithGallery},
		{galleryIdxWithPerformerTag, performerIdxWithTag},
		{galleryIdxWithPerformerTwoTags, performerIdxWithTwoTags},
		{galleryIdx1WithPerformer, performerIdxWithTwoGalleries},
		{galleryIdx2WithPerformer, performerIdxWithTwoGalleries},
		{galleryIdxWithStudioPerformer, performerIdxWithGalleryStudio},
	}

	galleryStudioLinks = [][2]int{
		{galleryIdxWithStudio, studioIdxWithGallery},
		{galleryIdx1WithStudio, studioIdxWithTwoGalleries},
		{galleryIdx2WithStudio, studioIdxWithTwoGalleries},
		{galleryIdxWithStudioPerformer, studioIdxWithGalleryPerformer},
		{galleryIdxWithGrandChildStudio, studioIdxWithGrandParent},
	}

	galleryTagLinks = [][2]int{
		{galleryIdxWithTag, tagIdxWithGallery},
		{galleryIdxWithTwoTags, tagIdx1WithGallery},
		{galleryIdxWithTwoTags, tagIdx2WithGallery},
	}
)

var (
	movieStudioLinks = [][2]int{
		{movieIdxWithStudio, studioIdxWithMovie},
	}
)

var (
	studioParentLinks = [][2]int{
		{studioIdxWithChildStudio, studioIdxWithParentStudio},
		{studioIdxWithGrandChild, studioIdxWithParentAndChild},
		{studioIdxWithParentAndChild, studioIdxWithGrandParent},
	}
)

var (
	performerTagLinks = [][2]int{
		{performerIdxWithTag, tagIdxWithPerformer},
		{performerIdxWithTwoTags, tagIdx1WithPerformer},
		{performerIdxWithTwoTags, tagIdx2WithPerformer},
	}
)

func TestMain(m *testing.M) {
	ret := runTests(m)
	os.Exit(ret)
}

func withTxn(f func(r models.Repository) error) error {
	t := sqlite.NewTransactionManager()
	return t.WithTxn(context.TODO(), f)
}

func withRollbackTxn(f func(r models.Repository) error) error {
	var ret error
	withTxn(func(repo models.Repository) error {
		ret = f(repo)
		return errors.New("fake error for rollback")
	})

	return ret
}

func testTeardown(databaseFile string) {
	err := database.DB.Close()

	if err != nil {
		panic(err)
	}

	err = os.Remove(databaseFile)
	if err != nil {
		panic(err)
	}
}

func runTests(m *testing.M) int {
	// create the database file
	f, err := ioutil.TempFile("", "*.sqlite")
	if err != nil {
		panic(fmt.Sprintf("Could not create temporary file: %s", err.Error()))
	}

	f.Close()
	databaseFile := f.Name()
	database.Initialize(databaseFile)

	// defer close and delete the database
	defer testTeardown(databaseFile)

	err = populateDB()
	if err != nil {
		panic(fmt.Sprintf("Could not populate database: %s", err.Error()))
	} else {
		// run the tests
		return m.Run()
	}
}

func populateDB() error {
	if err := withTxn(func(r models.Repository) error {
		if err := createScenes(r.Scene(), totalScenes); err != nil {
			return fmt.Errorf("error creating scenes: %s", err.Error())
		}

		if err := createImages(r.Image(), totalImages); err != nil {
			return fmt.Errorf("error creating images: %s", err.Error())
		}

		if err := createGalleries(r.Gallery(), totalGalleries); err != nil {
			return fmt.Errorf("error creating galleries: %s", err.Error())
		}

		if err := createMovies(r.Movie(), moviesNameCase, moviesNameNoCase); err != nil {
			return fmt.Errorf("error creating movies: %s", err.Error())
		}

		if err := createPerformers(r.Performer(), performersNameCase, performersNameNoCase); err != nil {
			return fmt.Errorf("error creating performers: %s", err.Error())
		}

		if err := createTags(r.Tag(), tagsNameCase, tagsNameNoCase); err != nil {
			return fmt.Errorf("error creating tags: %s", err.Error())
		}

		if err := addTagImage(r.Tag(), tagIdxWithCoverImage); err != nil {
			return fmt.Errorf("error adding tag image: %s", err.Error())
		}

		if err := createStudios(r.Studio(), studiosNameCase, studiosNameNoCase); err != nil {
			return fmt.Errorf("error creating studios: %s", err.Error())
		}

		if err := createSavedFilters(r.SavedFilter(), totalSavedFilters); err != nil {
			return fmt.Errorf("error creating saved filters: %s", err.Error())
		}

		if err := linkPerformerTags(r.Performer()); err != nil {
			return fmt.Errorf("error linking performer tags: %s", err.Error())
		}

		if err := linkSceneGalleries(r.Scene()); err != nil {
			return fmt.Errorf("error linking scenes to galleries: %s", err.Error())
		}

		if err := linkSceneMovies(r.Scene()); err != nil {
			return fmt.Errorf("error linking scenes to movies: %s", err.Error())
		}

		if err := linkScenePerformers(r.Scene()); err != nil {
			return fmt.Errorf("error linking scene performers: %s", err.Error())
		}

		if err := linkSceneTags(r.Scene()); err != nil {
			return fmt.Errorf("error linking scene tags: %s", err.Error())
		}

		if err := linkSceneStudios(r.Scene()); err != nil {
			return fmt.Errorf("error linking scene studios: %s", err.Error())
		}

		if err := linkImageGalleries(r.Gallery()); err != nil {
			return fmt.Errorf("error linking gallery images: %s", err.Error())
		}

		if err := linkImagePerformers(r.Image()); err != nil {
			return fmt.Errorf("error linking image performers: %s", err.Error())
		}

		if err := linkImageTags(r.Image()); err != nil {
			return fmt.Errorf("error linking image tags: %s", err.Error())
		}

		if err := linkImageStudios(r.Image()); err != nil {
			return fmt.Errorf("error linking image studio: %s", err.Error())
		}

		if err := linkMovieStudios(r.Movie()); err != nil {
			return fmt.Errorf("error linking movie studios: %s", err.Error())
		}

		if err := linkStudiosParent(r.Studio()); err != nil {
			return fmt.Errorf("error linking studios parent: %s", err.Error())
		}

		if err := linkGalleryPerformers(r.Gallery()); err != nil {
			return fmt.Errorf("error linking gallery performers: %s", err.Error())
		}

		if err := linkGalleryTags(r.Gallery()); err != nil {
			return fmt.Errorf("error linking gallery tags: %s", err.Error())
		}

		if err := linkGalleryStudios(r.Gallery()); err != nil {
			return fmt.Errorf("error linking gallery studios: %s", err.Error())
		}

		if err := createMarker(r.SceneMarker(), sceneIdxWithMarker, tagIdxWithPrimaryMarker, []int{tagIdxWithMarker}); err != nil {
			return fmt.Errorf("error creating scene marker: %s", err.Error())
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func getPrefixedStringValue(prefix string, index int, field string) string {
	return fmt.Sprintf("%s_%04d_%s", prefix, index, field)
}

func getPrefixedNullStringValue(prefix string, index int, field string) sql.NullString {
	if index > 0 && index%5 == 0 {
		return sql.NullString{}
	}
	if index > 0 && index%6 == 0 {
		return sql.NullString{
			String: "",
			Valid:  true,
		}
	}
	return sql.NullString{
		String: getPrefixedStringValue(prefix, index, field),
		Valid:  true,
	}
}

func getSceneStringValue(index int, field string) string {
	return getPrefixedStringValue("scene", index, field)
}

func getSceneNullStringValue(index int, field string) sql.NullString {
	return getPrefixedNullStringValue("scene", index, field)
}

func getSceneTitle(index int) string {
	switch index {
	case sceneIdxWithSpacedName:
		return spacedSceneTitle
	default:
		return getSceneStringValue(index, titleField)
	}
}

func getRating(index int) sql.NullInt64 {
	rating := index % 6
	return sql.NullInt64{Int64: int64(rating), Valid: rating > 0}
}

func getOCounter(index int) int {
	return index % 3
}

func getSceneDuration(index int) sql.NullFloat64 {
	duration := index % 4
	duration = duration * 100

	return sql.NullFloat64{
		Float64: float64(duration) + 0.432,
		Valid:   duration != 0,
	}
}

func getHeight(index int) sql.NullInt64 {
	heights := []int64{0, 200, 240, 300, 480, 700, 720, 800, 1080, 1500, 2160, 3000}
	height := heights[index%len(heights)]
	return sql.NullInt64{
		Int64: height,
		Valid: height != 0,
	}
}

func getWidth(index int) sql.NullInt64 {
	height := getHeight(index)
	return sql.NullInt64{
		Int64: height.Int64 * 2,
		Valid: height.Valid,
	}
}

func getSceneDate(index int) models.SQLiteDate {
	dates := []string{"null", "", "0001-01-01", "2001-02-03"}
	date := dates[index%len(dates)]
	return models.SQLiteDate{
		String: date,
		Valid:  date != "null",
	}
}

func createScenes(sqb models.SceneReaderWriter, n int) error {
	for i := 0; i < n; i++ {
		scene := models.Scene{
			Path:     getSceneStringValue(i, pathField),
			Title:    sql.NullString{String: getSceneTitle(i), Valid: true},
			Checksum: sql.NullString{String: getSceneStringValue(i, checksumField), Valid: true},
			Details:  sql.NullString{String: getSceneStringValue(i, "Details"), Valid: true},
			URL:      getSceneNullStringValue(i, urlField),
			Rating:   getRating(i),
			OCounter: getOCounter(i),
			Duration: getSceneDuration(i),
			Height:   getHeight(i),
			Width:    getWidth(i),
			Date:     getSceneDate(i),
		}

		created, err := sqb.Create(scene)

		if err != nil {
			return fmt.Errorf("Error creating scene %v+: %s", scene, err.Error())
		}

		sceneIDs = append(sceneIDs, created.ID)
	}

	return nil
}

func getImageStringValue(index int, field string) string {
	return fmt.Sprintf("image_%04d_%s", index, field)
}

func getImagePath(index int) string {
	// TODO - currently not working
	// if index == imageIdxInZip {
	// 	return image.ZipFilename(zipPath, "image_0001_Path")
	// }

	return getImageStringValue(index, pathField)
}

func createImages(qb models.ImageReaderWriter, n int) error {
	for i := 0; i < n; i++ {
		image := models.Image{
			Path:     getImagePath(i),
			Title:    sql.NullString{String: getImageStringValue(i, titleField), Valid: true},
			Checksum: getImageStringValue(i, checksumField),
			Rating:   getRating(i),
			OCounter: getOCounter(i),
			Height:   getHeight(i),
			Width:    getWidth(i),
		}

		created, err := qb.Create(image)

		if err != nil {
			return fmt.Errorf("Error creating image %v+: %s", image, err.Error())
		}

		imageIDs = append(imageIDs, created.ID)
	}

	return nil
}

func getGalleryStringValue(index int, field string) string {
	return getPrefixedStringValue("gallery", index, field)
}

func getGalleryNullStringValue(index int, field string) sql.NullString {
	return getPrefixedNullStringValue("gallery", index, field)
}

func createGalleries(gqb models.GalleryReaderWriter, n int) error {
	for i := 0; i < n; i++ {
		gallery := models.Gallery{
			Path:     models.NullString(getGalleryStringValue(i, pathField)),
			URL:      getGalleryNullStringValue(i, urlField),
			Checksum: getGalleryStringValue(i, checksumField),
			Rating:   getRating(i),
		}

		created, err := gqb.Create(gallery)

		if err != nil {
			return fmt.Errorf("Error creating gallery %v+: %s", gallery, err.Error())
		}

		galleryIDs = append(galleryIDs, created.ID)
	}

	return nil
}

func getMovieStringValue(index int, field string) string {
	return getPrefixedStringValue("movie", index, field)
}

func getMovieNullStringValue(index int, field string) sql.NullString {
	return getPrefixedNullStringValue("movie", index, field)
}

// createMoviees creates n movies with plain Name and o movies with camel cased NaMe included
func createMovies(mqb models.MovieReaderWriter, n int, o int) error {
	const namePlain = "Name"
	const nameNoCase = "NaMe"

	for i := 0; i < n+o; i++ {
		index := i
		name := namePlain

		if i >= n { // i<n tags get normal names
			name = nameNoCase       // i>=n movies get dup names if case is not checked
			index = n + o - (i + 1) // for the name to be the same the number (index) must be the same also
		} // so count backwards to 0 as needed
		// movies [ i ] and [ n + o - i - 1  ] should have similar names with only the Name!=NaMe part different

		name = getMovieStringValue(index, name)
		movie := models.Movie{
			Name:     sql.NullString{String: name, Valid: true},
			URL:      getMovieNullStringValue(index, urlField),
			Checksum: utils.MD5FromString(name),
		}

		created, err := mqb.Create(movie)

		if err != nil {
			return fmt.Errorf("Error creating movie [%d] %v+: %s", i, movie, err.Error())
		}

		movieIDs = append(movieIDs, created.ID)
		movieNames = append(movieNames, created.Name.String)
	}

	return nil
}

func getPerformerStringValue(index int, field string) string {
	return getPrefixedStringValue("performer", index, field)
}

func getPerformerNullStringValue(index int, field string) sql.NullString {
	return getPrefixedNullStringValue("performer", index, field)
}

func getPerformerBoolValue(index int) bool {
	index = index % 2
	return index == 1
}

func getPerformerBirthdate(index int) string {
	const minAge = 18
	birthdate := time.Now()
	birthdate = birthdate.AddDate(-minAge-index, -1, -1)
	return birthdate.Format("2006-01-02")
}

func getPerformerDeathDate(index int) models.SQLiteDate {
	if index != 5 {
		return models.SQLiteDate{}
	}

	deathDate := time.Now()
	deathDate = deathDate.AddDate(-index+1, -1, -1)
	return models.SQLiteDate{
		String: deathDate.Format("2006-01-02"),
		Valid:  true,
	}
}

func getPerformerCareerLength(index int) *string {
	if index%5 == 0 {
		return nil
	}

	ret := fmt.Sprintf("20%2d", index)
	return &ret
}

// createPerformers creates n performers with plain Name and o performers with camel cased NaMe included
func createPerformers(pqb models.PerformerReaderWriter, n int, o int) error {
	const namePlain = "Name"
	const nameNoCase = "NaMe"

	name := namePlain

	for i := 0; i < n+o; i++ {
		index := i

		if i >= n { // i<n tags get normal names
			name = nameNoCase       // i>=n performers get dup names if case is not checked
			index = n + o - (i + 1) // for the name to be the same the number (index) must be the same also
		} // so count backwards to 0 as needed
		// performers [ i ] and [ n + o - i - 1  ] should have similar names with only the Name!=NaMe part different

		performer := models.Performer{
			Name:     sql.NullString{String: getPerformerStringValue(index, name), Valid: true},
			Checksum: getPerformerStringValue(i, checksumField),
			URL:      getPerformerNullStringValue(i, urlField),
			Favorite: sql.NullBool{Bool: getPerformerBoolValue(i), Valid: true},
			Birthdate: models.SQLiteDate{
				String: getPerformerBirthdate(i),
				Valid:  true,
			},
			DeathDate: getPerformerDeathDate(i),
			Details:   sql.NullString{String: getPerformerStringValue(i, "Details"), Valid: true},
			Ethnicity: sql.NullString{String: getPerformerStringValue(i, "Ethnicity"), Valid: true},
			Rating:    getRating(i),
		}

		careerLength := getPerformerCareerLength(i)
		if careerLength != nil {
			performer.CareerLength = models.NullString(*careerLength)
		}

		created, err := pqb.Create(performer)

		if err != nil {
			return fmt.Errorf("Error creating performer %v+: %s", performer, err.Error())
		}

		performerIDs = append(performerIDs, created.ID)
		performerNames = append(performerNames, created.Name.String)
	}

	return nil
}

func getTagStringValue(index int, field string) string {
	return "tag_" + strconv.FormatInt(int64(index), 10) + "_" + field
}

func getTagSceneCount(id int) int {
	if id == tagIDs[tagIdx1WithScene] || id == tagIDs[tagIdx2WithScene] || id == tagIDs[tagIdxWithScene] {
		return 1
	}

	return 0
}

func getTagMarkerCount(id int) int {
	if id == tagIDs[tagIdxWithMarker] || id == tagIDs[tagIdxWithPrimaryMarker] {
		return 1
	}

	return 0
}

func getTagImageCount(id int) int {
	if id == tagIDs[tagIdx1WithImage] || id == tagIDs[tagIdx2WithImage] || id == tagIDs[tagIdxWithImage] {
		return 1
	}

	return 0
}

func getTagGalleryCount(id int) int {
	if id == tagIDs[tagIdx1WithGallery] || id == tagIDs[tagIdx2WithGallery] || id == tagIDs[tagIdxWithGallery] {
		return 1
	}

	return 0
}

func getTagPerformerCount(id int) int {
	if id == tagIDs[tagIdx1WithPerformer] || id == tagIDs[tagIdx2WithPerformer] || id == tagIDs[tagIdxWithPerformer] {
		return 1
	}

	return 0
}

//createTags creates n tags with plain Name and o tags with camel cased NaMe included
func createTags(tqb models.TagReaderWriter, n int, o int) error {
	const namePlain = "Name"
	const nameNoCase = "NaMe"

	name := namePlain

	for i := 0; i < n+o; i++ {
		index := i

		if i >= n { // i<n tags get normal names
			name = nameNoCase       // i>=n tags get dup names if case is not checked
			index = n + o - (i + 1) // for the name to be the same the number (index) must be the same also
		} // so count backwards to 0 as needed
		// tags [ i ] and [ n + o - i - 1  ] should have similar names with only the Name!=NaMe part different

		tag := models.Tag{
			Name: getTagStringValue(index, name),
		}

		created, err := tqb.Create(tag)

		if err != nil {
			return fmt.Errorf("Error creating tag %v+: %s", tag, err.Error())
		}

		// add alias
		alias := getTagStringValue(i, "Alias")
		if err := tqb.UpdateAliases(created.ID, []string{alias}); err != nil {
			return fmt.Errorf("error setting tag alias: %s", err.Error())
		}

		tagIDs = append(tagIDs, created.ID)
		tagNames = append(tagNames, created.Name)
	}

	return nil
}

func getStudioStringValue(index int, field string) string {
	return getPrefixedStringValue("studio", index, field)
}

func getStudioNullStringValue(index int, field string) sql.NullString {
	return getPrefixedNullStringValue("studio", index, field)
}

func createStudio(sqb models.StudioReaderWriter, name string, parentID *int64) (*models.Studio, error) {
	studio := models.Studio{
		Name:     sql.NullString{String: name, Valid: true},
		Checksum: utils.MD5FromString(name),
	}

	if parentID != nil {
		studio.ParentID = sql.NullInt64{Int64: *parentID, Valid: true}
	}

	return createStudioFromModel(sqb, studio)
}

func createStudioFromModel(sqb models.StudioReaderWriter, studio models.Studio) (*models.Studio, error) {
	created, err := sqb.Create(studio)

	if err != nil {
		return nil, fmt.Errorf("Error creating studio %v+: %s", studio, err.Error())
	}

	return created, nil
}

// createStudios creates n studios with plain Name and o studios with camel cased NaMe included
func createStudios(sqb models.StudioReaderWriter, n int, o int) error {
	const namePlain = "Name"
	const nameNoCase = "NaMe"

	for i := 0; i < n+o; i++ {
		index := i
		name := namePlain

		if i >= n { // i<n studios get normal names
			name = nameNoCase       // i>=n studios get dup names if case is not checked
			index = n + o - (i + 1) // for the name to be the same the number (index) must be the same also
		} // so count backwards to 0 as needed
		// studios [ i ] and [ n + o - i - 1  ] should have similar names with only the Name!=NaMe part different

		name = getStudioStringValue(index, name)
		studio := models.Studio{
			Name:     sql.NullString{String: name, Valid: true},
			Checksum: utils.MD5FromString(name),
			URL:      getStudioNullStringValue(index, urlField),
		}
		created, err := createStudioFromModel(sqb, studio)

		if err != nil {
			return err
		}

		studioIDs = append(studioIDs, created.ID)
		studioNames = append(studioNames, created.Name.String)
	}

	return nil
}

func createMarker(mqb models.SceneMarkerReaderWriter, sceneIdx, primaryTagIdx int, tagIdxs []int) error {
	marker := models.SceneMarker{
		SceneID:      sql.NullInt64{Int64: int64(sceneIDs[sceneIdx]), Valid: true},
		PrimaryTagID: tagIDs[primaryTagIdx],
	}

	created, err := mqb.Create(marker)

	if err != nil {
		return fmt.Errorf("Error creating marker %v+: %s", marker, err.Error())
	}

	markerIDs = append(markerIDs, created.ID)

	newTagIDs := []int{}

	for _, tagIdx := range tagIdxs {
		newTagIDs = append(newTagIDs, tagIDs[tagIdx])
	}

	if err := mqb.UpdateTags(created.ID, newTagIDs); err != nil {
		return fmt.Errorf("Error creating marker/tag join: %s", err.Error())
	}

	return nil
}

func getSavedFilterMode(index int) models.FilterMode {
	switch index {
	case savedFilterIdxScene, savedFilterIdxDefaultScene:
		return models.FilterModeScenes
	case savedFilterIdxImage, savedFilterIdxDefaultImage:
		return models.FilterModeImages
	default:
		return models.FilterModeScenes
	}
}

func getSavedFilterName(index int) string {
	if index <= savedFilterIdxDefaultImage {
		// empty string for default filters
		return ""
	}

	if index <= savedFilterIdxImage {
		// use the same name for the first two - should be possible
		return firstSavedFilterName
	}

	return getPrefixedStringValue("savedFilter", index, "Name")
}

func createSavedFilters(qb models.SavedFilterReaderWriter, n int) error {
	for i := 0; i < n; i++ {
		savedFilter := models.SavedFilter{
			Mode:   getSavedFilterMode(i),
			Name:   getSavedFilterName(i),
			Filter: getPrefixedStringValue("savedFilter", i, "Filter"),
		}

		created, err := qb.Create(savedFilter)

		if err != nil {
			return fmt.Errorf("Error creating saved filter %v+: %s", savedFilter, err.Error())
		}

		savedFilterIDs = append(savedFilterIDs, created.ID)
	}

	return nil
}

func doLinks(links [][2]int, fn func(idx1, idx2 int) error) error {
	for _, l := range links {
		if err := fn(l[0], l[1]); err != nil {
			return err
		}
	}

	return nil
}

func linkPerformerTags(qb models.PerformerReaderWriter) error {
	return doLinks(performerTagLinks, func(performerIndex, tagIndex int) error {
		performerID := performerIDs[performerIndex]
		tagID := tagIDs[tagIndex]
		tagIDs, err := qb.GetTagIDs(performerID)
		if err != nil {
			return err
		}

		tagIDs = utils.IntAppendUnique(tagIDs, tagID)

		return qb.UpdateTags(performerID, tagIDs)
	})
}

func linkSceneMovies(qb models.SceneReaderWriter) error {
	return doLinks(sceneMovieLinks, func(sceneIndex, movieIndex int) error {
		sceneID := sceneIDs[sceneIndex]
		movies, err := qb.GetMovies(sceneID)
		if err != nil {
			return err
		}

		movies = append(movies, models.MoviesScenes{
			MovieID: movieIDs[movieIndex],
			SceneID: sceneID,
		})
		return qb.UpdateMovies(sceneID, movies)
	})
}

func linkScenePerformers(qb models.SceneReaderWriter) error {
	return doLinks(scenePerformerLinks, func(sceneIndex, performerIndex int) error {
		_, err := scene.AddPerformer(qb, sceneIDs[sceneIndex], performerIDs[performerIndex])
		return err
	})
}

func linkSceneGalleries(qb models.SceneReaderWriter) error {
	return doLinks(sceneGalleryLinks, func(sceneIndex, galleryIndex int) error {
		_, err := scene.AddGallery(qb, sceneIDs[sceneIndex], galleryIDs[galleryIndex])
		return err
	})
}

func linkSceneTags(qb models.SceneReaderWriter) error {
	return doLinks(sceneTagLinks, func(sceneIndex, tagIndex int) error {
		_, err := scene.AddTag(qb, sceneIDs[sceneIndex], tagIDs[tagIndex])
		return err
	})
}

func linkSceneStudios(sqb models.SceneWriter) error {
	return doLinks(sceneStudioLinks, func(sceneIndex, studioIndex int) error {
		scene := models.ScenePartial{
			ID:       sceneIDs[sceneIndex],
			StudioID: &sql.NullInt64{Int64: int64(studioIDs[studioIndex]), Valid: true},
		}
		_, err := sqb.Update(scene)

		return err
	})
}

func linkImageGalleries(gqb models.GalleryReaderWriter) error {
	return doLinks(imageGalleryLinks, func(imageIndex, galleryIndex int) error {
		return gallery.AddImage(gqb, galleryIDs[galleryIndex], imageIDs[imageIndex])
	})
}

func linkImageTags(iqb models.ImageReaderWriter) error {
	return doLinks(imageTagLinks, func(imageIndex, tagIndex int) error {
		imageID := imageIDs[imageIndex]
		tags, err := iqb.GetTagIDs(imageID)
		if err != nil {
			return err
		}

		tags = append(tags, tagIDs[tagIndex])

		return iqb.UpdateTags(imageID, tags)
	})
}

func linkImageStudios(qb models.ImageWriter) error {
	return doLinks(imageStudioLinks, func(imageIndex, studioIndex int) error {
		image := models.ImagePartial{
			ID:       imageIDs[imageIndex],
			StudioID: &sql.NullInt64{Int64: int64(studioIDs[studioIndex]), Valid: true},
		}
		_, err := qb.Update(image)

		return err
	})
}

func linkImagePerformers(qb models.ImageReaderWriter) error {
	return doLinks(imagePerformerLinks, func(imageIndex, performerIndex int) error {
		imageID := imageIDs[imageIndex]
		performers, err := qb.GetPerformerIDs(imageID)
		if err != nil {
			return err
		}

		performers = append(performers, performerIDs[performerIndex])

		return qb.UpdatePerformers(imageID, performers)
	})
}

func linkGalleryPerformers(qb models.GalleryReaderWriter) error {
	return doLinks(galleryPerformerLinks, func(galleryIndex, performerIndex int) error {
		galleryID := galleryIDs[galleryIndex]
		performers, err := qb.GetPerformerIDs(galleryID)
		if err != nil {
			return err
		}

		performers = append(performers, performerIDs[performerIndex])

		return qb.UpdatePerformers(galleryID, performers)
	})
}

func linkGalleryStudios(qb models.GalleryReaderWriter) error {
	return doLinks(galleryStudioLinks, func(galleryIndex, studioIndex int) error {
		gallery := models.GalleryPartial{
			ID:       galleryIDs[galleryIndex],
			StudioID: &sql.NullInt64{Int64: int64(studioIDs[studioIndex]), Valid: true},
		}
		_, err := qb.UpdatePartial(gallery)

		return err
	})
}

func linkGalleryTags(qb models.GalleryReaderWriter) error {
	return doLinks(galleryTagLinks, func(galleryIndex, tagIndex int) error {
		galleryID := galleryIDs[galleryIndex]
		tags, err := qb.GetTagIDs(galleryID)
		if err != nil {
			return err
		}

		tags = append(tags, tagIDs[tagIndex])

		return qb.UpdateTags(galleryID, tags)
	})
}

func linkMovieStudios(mqb models.MovieWriter) error {
	return doLinks(movieStudioLinks, func(movieIndex, studioIndex int) error {
		movie := models.MoviePartial{
			ID:       movieIDs[movieIndex],
			StudioID: &sql.NullInt64{Int64: int64(studioIDs[studioIndex]), Valid: true},
		}
		_, err := mqb.Update(movie)

		return err
	})
}

func linkStudiosParent(qb models.StudioWriter) error {
	return doLinks(studioParentLinks, func(parentIndex, childIndex int) error {
		studio := models.StudioPartial{
			ID:       studioIDs[childIndex],
			ParentID: &sql.NullInt64{Int64: int64(studioIDs[parentIndex]), Valid: true},
		}
		_, err := qb.Update(studio)

		return err
	})
}

func addTagImage(qb models.TagWriter, tagIndex int) error {
	return qb.UpdateImage(tagIDs[tagIndex], models.DefaultTagImage)
}

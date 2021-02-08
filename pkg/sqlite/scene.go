package sqlite

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/stashapp/stash/pkg/models"
)

const sceneTable = "scenes"
const sceneIDColumn = "scene_id"
const performersScenesTable = "performers_scenes"
const scenesTagsTable = "scenes_tags"
const scenesGalleriesTable = "scenes_galleries"
const moviesScenesTable = "movies_scenes"

var scenesForPerformerQuery = selectAll(sceneTable) + `
LEFT JOIN performers_scenes as performers_join on performers_join.scene_id = scenes.id
WHERE performers_join.performer_id = ?
GROUP BY scenes.id
`

var countScenesForPerformerQuery = `
SELECT performer_id FROM performers_scenes as performers_join
WHERE performer_id = ?
GROUP BY scene_id
`

var scenesForStudioQuery = selectAll(sceneTable) + `
JOIN studios ON studios.id = scenes.studio_id
WHERE studios.id = ?
GROUP BY scenes.id
`
var scenesForMovieQuery = selectAll(sceneTable) + `
LEFT JOIN movies_scenes as movies_join on movies_join.scene_id = scenes.id
WHERE movies_join.movie_id = ?
GROUP BY scenes.id
`

var countScenesForTagQuery = `
SELECT tag_id AS id FROM scenes_tags
WHERE scenes_tags.tag_id = ?
GROUP BY scenes_tags.scene_id
`

var scenesForGalleryQuery = selectAll(sceneTable) + `
LEFT JOIN scenes_galleries as galleries_join on galleries_join.scene_id = scenes.id
WHERE galleries_join.gallery_id = ?
GROUP BY scenes.id
`

var countScenesForMissingChecksumQuery = `
SELECT id FROM scenes
WHERE scenes.checksum is null
`

var countScenesForMissingOSHashQuery = `
SELECT id FROM scenes
WHERE scenes.oshash is null
`

type sceneQueryBuilder struct {
	repository
}

func NewSceneReaderWriter(tx dbi) *sceneQueryBuilder {
	return &sceneQueryBuilder{
		repository{
			tx:        tx,
			tableName: sceneTable,
			idColumn:  idColumn,
		},
	}
}

func (qb *sceneQueryBuilder) Create(newObject models.Scene) (*models.Scene, error) {
	var ret models.Scene
	if err := qb.insertObject(newObject, &ret); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (qb *sceneQueryBuilder) Update(updatedObject models.ScenePartial) (*models.Scene, error) {
	const partial = true
	if err := qb.update(updatedObject.ID, updatedObject, partial); err != nil {
		return nil, err
	}

	return qb.find(updatedObject.ID)
}

func (qb *sceneQueryBuilder) UpdateFull(updatedObject models.Scene) (*models.Scene, error) {
	const partial = false
	if err := qb.update(updatedObject.ID, updatedObject, partial); err != nil {
		return nil, err
	}

	return qb.find(updatedObject.ID)
}

func (qb *sceneQueryBuilder) UpdateFileModTime(id int, modTime models.NullSQLiteTimestamp) error {
	return qb.updateMap(id, map[string]interface{}{
		"file_mod_time": modTime,
	})
}

func (qb *sceneQueryBuilder) IncrementOCounter(id int) (int, error) {
	_, err := qb.tx.Exec(
		`UPDATE scenes SET o_counter = o_counter + 1 WHERE scenes.id = ?`,
		id,
	)
	if err != nil {
		return 0, err
	}

	scene, err := qb.find(id)
	if err != nil {
		return 0, err
	}

	return scene.OCounter, nil
}

func (qb *sceneQueryBuilder) DecrementOCounter(id int) (int, error) {
	_, err := qb.tx.Exec(
		`UPDATE scenes SET o_counter = o_counter - 1 WHERE scenes.id = ? and scenes.o_counter > 0`,
		id,
	)
	if err != nil {
		return 0, err
	}

	scene, err := qb.find(id)
	if err != nil {
		return 0, err
	}

	return scene.OCounter, nil
}

func (qb *sceneQueryBuilder) ResetOCounter(id int) (int, error) {
	_, err := qb.tx.Exec(
		`UPDATE scenes SET o_counter = 0 WHERE scenes.id = ?`,
		id,
	)
	if err != nil {
		return 0, err
	}

	scene, err := qb.find(id)
	if err != nil {
		return 0, err
	}

	return scene.OCounter, nil
}

func (qb *sceneQueryBuilder) Destroy(id int) error {
	// delete all related table rows
	// TODO - this should be handled by a delete cascade
	if err := qb.performersRepository().destroy([]int{id}); err != nil {
		return err
	}

	// scene markers should be handled prior to calling destroy
	// galleries should be handled prior to calling destroy

	return qb.destroyExisting([]int{id})
}

func (qb *sceneQueryBuilder) Find(id int) (*models.Scene, error) {
	return qb.find(id)
}

func (qb *sceneQueryBuilder) FindMany(ids []int) ([]*models.Scene, error) {
	var scenes []*models.Scene
	for _, id := range ids {
		scene, err := qb.Find(id)
		if err != nil {
			return nil, err
		}

		if scene == nil {
			return nil, fmt.Errorf("scene with id %d not found", id)
		}

		scenes = append(scenes, scene)
	}

	return scenes, nil
}

func (qb *sceneQueryBuilder) find(id int) (*models.Scene, error) {
	var ret models.Scene
	if err := qb.get(id, &ret); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ret, nil
}

func (qb *sceneQueryBuilder) FindByChecksum(checksum string) (*models.Scene, error) {
	query := "SELECT * FROM scenes WHERE checksum = ? LIMIT 1"
	args := []interface{}{checksum}
	return qb.queryScene(query, args)
}

func (qb *sceneQueryBuilder) FindByOSHash(oshash string) (*models.Scene, error) {
	query := "SELECT * FROM scenes WHERE oshash = ? LIMIT 1"
	args := []interface{}{oshash}
	return qb.queryScene(query, args)
}

func (qb *sceneQueryBuilder) FindByPath(path string) (*models.Scene, error) {
	query := selectAll(sceneTable) + "WHERE path = ? LIMIT 1"
	args := []interface{}{path}
	return qb.queryScene(query, args)
}

func (qb *sceneQueryBuilder) FindByPerformerID(performerID int) ([]*models.Scene, error) {
	args := []interface{}{performerID}
	return qb.queryScenes(scenesForPerformerQuery, args)
}

func (qb *sceneQueryBuilder) FindByGalleryID(galleryID int) ([]*models.Scene, error) {
	args := []interface{}{galleryID}
	return qb.queryScenes(scenesForGalleryQuery, args)
}

func (qb *sceneQueryBuilder) CountByPerformerID(performerID int) (int, error) {
	args := []interface{}{performerID}
	return qb.runCountQuery(qb.buildCountQuery(countScenesForPerformerQuery), args)
}

func (qb *sceneQueryBuilder) FindByMovieID(movieID int) ([]*models.Scene, error) {
	args := []interface{}{movieID}
	return qb.queryScenes(scenesForMovieQuery, args)
}

func (qb *sceneQueryBuilder) CountByMovieID(movieID int) (int, error) {
	args := []interface{}{movieID}
	return qb.runCountQuery(qb.buildCountQuery(scenesForMovieQuery), args)
}

func (qb *sceneQueryBuilder) Count() (int, error) {
	return qb.runCountQuery(qb.buildCountQuery("SELECT scenes.id FROM scenes"), nil)
}

func (qb *sceneQueryBuilder) Size() (float64, error) {
	return qb.runSumQuery("SELECT SUM(cast(size as double)) as sum FROM scenes", nil)
}

func (qb *sceneQueryBuilder) CountByStudioID(studioID int) (int, error) {
	args := []interface{}{studioID}
	return qb.runCountQuery(qb.buildCountQuery(scenesForStudioQuery), args)
}

func (qb *sceneQueryBuilder) CountByTagID(tagID int) (int, error) {
	args := []interface{}{tagID}
	return qb.runCountQuery(qb.buildCountQuery(countScenesForTagQuery), args)
}

// CountMissingChecksum returns the number of scenes missing a checksum value.
func (qb *sceneQueryBuilder) CountMissingChecksum() (int, error) {
	return qb.runCountQuery(qb.buildCountQuery(countScenesForMissingChecksumQuery), []interface{}{})
}

// CountMissingOSHash returns the number of scenes missing an oshash value.
func (qb *sceneQueryBuilder) CountMissingOSHash() (int, error) {
	return qb.runCountQuery(qb.buildCountQuery(countScenesForMissingOSHashQuery), []interface{}{})
}

func (qb *sceneQueryBuilder) Wall(q *string) ([]*models.Scene, error) {
	s := ""
	if q != nil {
		s = *q
	}
	query := selectAll(sceneTable) + "WHERE scenes.details LIKE '%" + s + "%' ORDER BY RANDOM() LIMIT 80"
	return qb.queryScenes(query, nil)
}

func (qb *sceneQueryBuilder) All() ([]*models.Scene, error) {
	return qb.queryScenes(selectAll(sceneTable)+qb.getSceneSort(nil), nil)
}

// QueryForAutoTag queries for scenes whose paths match the provided regex and
// are optionally within the provided path. Excludes organized scenes.
// TODO - this should be replaced with Query once it can perform multiple
// filters on the same field.
func (qb *sceneQueryBuilder) QueryForAutoTag(regex string, pathPrefixes []string) ([]*models.Scene, error) {
	var args []interface{}
	body := selectDistinctIDs("scenes") + ` WHERE 
	scenes.path regexp ? AND 
	scenes.organized = 0`

	args = append(args, "(?i)"+regex)

	var pathClauses []string
	for _, p := range pathPrefixes {
		pathClauses = append(pathClauses, "scenes.path like ?")

		sep := string(filepath.Separator)
		if !strings.HasSuffix(p, sep) {
			p = p + sep
		}
		args = append(args, p+"%")
	}

	if len(pathClauses) > 0 {
		body += " AND (" + strings.Join(pathClauses, " OR ") + ")"
	}

	idsResult, err := qb.runIdsQuery(body, args)

	if err != nil {
		return nil, err
	}

	var scenes []*models.Scene
	for _, id := range idsResult {
		scene, err := qb.Find(id)

		if err != nil {
			return nil, err
		}

		scenes = append(scenes, scene)
	}

	return scenes, nil
}

func (qb *sceneQueryBuilder) makeFilter(sceneFilter *models.SceneFilterType) *filterBuilder {
	query := &filterBuilder{}

	query.handleCriterionFunc(stringCriterionHandler(sceneFilter.Path, "scenes.path"))
	query.handleCriterionFunc(intCriterionHandler(sceneFilter.Rating, "scenes.rating"))
	query.handleCriterionFunc(intCriterionHandler(sceneFilter.OCounter, "scenes.o_counter"))
	query.handleCriterionFunc(boolCriterionHandler(sceneFilter.Organized, "scenes.organized"))
	query.handleCriterionFunc(durationCriterionHandler(sceneFilter.Duration, "scenes.duration"))
	query.handleCriterionFunc(resolutionCriterionHandler(sceneFilter.Resolution, "scenes.height", "scenes.width"))
	query.handleCriterionFunc(hasMarkersCriterionHandler(sceneFilter.HasMarkers))
	query.handleCriterionFunc(sceneIsMissingCriterionHandler(qb, sceneFilter.IsMissing))

	query.handleCriterionFunc(sceneTagsCriterionHandler(qb, sceneFilter.Tags))
	query.handleCriterionFunc(scenePerformersCriterionHandler(qb, sceneFilter.Performers))
	query.handleCriterionFunc(sceneStudioCriterionHandler(sceneFilter.Studios))
	query.handleCriterionFunc(sceneMoviesCriterionHandler(qb, sceneFilter.Movies))
	query.handleCriterionFunc(sceneStashIDsHandler(qb, sceneFilter.StashID))

	return query
}

func (qb *sceneQueryBuilder) Query(sceneFilter *models.SceneFilterType, findFilter *models.FindFilterType) ([]*models.Scene, int, error) {
	if sceneFilter == nil {
		sceneFilter = &models.SceneFilterType{}
	}
	if findFilter == nil {
		findFilter = &models.FindFilterType{}
	}

	query := qb.newQuery()

	query.body = selectDistinctIDs(sceneTable)

	if q := findFilter.Q; q != nil && *q != "" {
		query.join("scene_markers", "", "scene_markers.scene_id = scenes.id")
		searchColumns := []string{"scenes.title", "scenes.details", "scenes.path", "scenes.oshash", "scenes.checksum", "scene_markers.title"}
		clause, thisArgs := getSearchBinding(searchColumns, *q, false)
		query.addWhere(clause)
		query.addArg(thisArgs...)
	}

	filter := qb.makeFilter(sceneFilter)

	filter.addToQueryBuilder(&query)

	query.sortAndPagination = qb.getSceneSort(findFilter) + getPagination(findFilter)

	idsResult, countResult, err := query.executeFind()
	if err != nil {
		return nil, 0, err
	}

	var scenes []*models.Scene
	for _, id := range idsResult {
		scene, err := qb.Find(id)
		if err != nil {
			return nil, 0, err
		}
		scenes = append(scenes, scene)
	}

	return scenes, countResult, nil
}

func appendClause(clauses []string, clause string) []string {
	if clause != "" {
		return append(clauses, clause)
	}

	return clauses
}

func durationCriterionHandler(durationFilter *models.IntCriterionInput, column string) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if durationFilter != nil {
			clause, thisArgs := getDurationWhereClause(*durationFilter, column)
			f.addWhere(clause, thisArgs...)
		}
	}
}

func getDurationWhereClause(durationFilter models.IntCriterionInput, column string) (string, []interface{}) {
	// special case for duration. We accept duration as seconds as int but the
	// field is floating point. Change the equals filter to return a range
	// between x and x + 1
	// likewise, not equals needs to be duration < x OR duration >= x
	var clause string
	args := []interface{}{}

	value := durationFilter.Value
	if durationFilter.Modifier == models.CriterionModifierEquals {
		clause = fmt.Sprintf("%[1]s >= ? AND %[1]s < ?", column)
		args = append(args, value)
		args = append(args, value+1)
	} else if durationFilter.Modifier == models.CriterionModifierNotEquals {
		clause = fmt.Sprintf("(%[1]s < ? OR %[1]s >= ?)", column)
		args = append(args, value)
		args = append(args, value+1)
	} else {
		var count int
		clause, count = getIntCriterionWhereClause(column, durationFilter)
		if count == 1 {
			args = append(args, value)
		}
	}

	return clause, args
}

func resolutionCriterionHandler(resolution *models.ResolutionEnum, heightColumn string, widthColumn string) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if resolution != nil && resolution.IsValid() {
			min := resolution.GetMinResolution()
			max := resolution.GetMaxResolution()

			widthHeight := fmt.Sprintf("MIN(%s, %s)", widthColumn, heightColumn)

			if min > 0 {
				f.addWhere(widthHeight + " >= " + strconv.Itoa(min))
			}

			if max > 0 {
				f.addWhere(widthHeight + " < " + strconv.Itoa(max))
			}
		}
	}
}

func hasMarkersCriterionHandler(hasMarkers *string) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if hasMarkers != nil {
			f.addJoin("scene_markers", "", "scene_markers.scene_id = scenes.id")
			if *hasMarkers == "true" {
				f.addHaving("count(scene_markers.scene_id) > 0")
			} else {
				f.addWhere("scene_markers.id IS NULL")
			}
		}
	}
}

func sceneIsMissingCriterionHandler(qb *sceneQueryBuilder, isMissing *string) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if isMissing != nil && *isMissing != "" {
			// TODO - add joins
			switch *isMissing {
			case "galleries":
				qb.galleriesRepository().join(f, "galleries_join", "scenes.id")
				f.addWhere("galleries_join.scene_id IS NULL")
			case "studio":
				f.addWhere("scenes.studio_id IS NULL")
			case "movie":
				qb.moviesRepository().join(f, "movies_join", "scenes.id")
				f.addWhere("movies_join.scene_id IS NULL")
			case "performers":
				qb.performersRepository().join(f, "performers_join", "scenes.id")
				f.addWhere("performers_join.scene_id IS NULL")
			case "date":
				f.addWhere("scenes.date IS \"\" OR scenes.date IS \"0001-01-01\"")
			case "tags":
				qb.tagsRepository().join(f, "tags_join", "scenes.id")
				f.addWhere("tags_join.scene_id IS NULL")
			case "stash_id":
				qb.stashIDRepository().join(f, "scene_stash_ids", "scenes.id")
				f.addWhere("scene_stash_ids.scene_id IS NULL")
			default:
				f.addWhere("(scenes." + *isMissing + " IS NULL OR TRIM(scenes." + *isMissing + ") = '')")
			}
		}
	}
}

func sceneTagsCriterionHandler(qb *sceneQueryBuilder, tags *models.MultiCriterionInput) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if tags != nil && len(tags.Value) > 0 {
			var args []interface{}
			for _, tagID := range tags.Value {
				args = append(args, tagID)
			}

			qb.tagsRepository().join(f, "tags_join", "scenes.id")
			f.addJoin("tags", "", "tags_join.tag_id = tags.id")
			whereClause, havingClause := getMultiCriterionClause("scenes", "tags", "scenes_tags", "scene_id", "tag_id", tags)
			f.addWhere(whereClause, args...)
			f.addHaving(havingClause)
		}
	}
}

func scenePerformersCriterionHandler(qb *sceneQueryBuilder, performers *models.MultiCriterionInput) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if performers != nil && len(performers.Value) > 0 {
			var args []interface{}
			for _, performerID := range performers.Value {
				args = append(args, performerID)
			}

			qb.performersRepository().join(f, "performers_join", "scenes.id")
			f.addJoin("performers", "", "performers_join.performer_id = performers.id")
			whereClause, havingClause := getMultiCriterionClause("scenes", "performers", "performers_scenes", "scene_id", "performer_id", performers)
			f.addWhere(whereClause, args...)
			f.addHaving(havingClause)
		}
	}
}

func sceneStudioCriterionHandler(studios *models.MultiCriterionInput) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if studios != nil && len(studios.Value) > 0 {
			var args []interface{}
			for _, studioID := range studios.Value {
				args = append(args, studioID)
			}

			f.addJoin("studios", "studio", "studio.id = scenes.studio_id")
			whereClause, havingClause := getMultiCriterionClause("scenes", "studio", "", "", "studio_id", studios)
			f.addWhere(whereClause, args...)
			f.addHaving(havingClause)
		}
	}
}

func sceneMoviesCriterionHandler(qb *sceneQueryBuilder, movies *models.MultiCriterionInput) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if movies != nil && len(movies.Value) > 0 {
			var args []interface{}
			for _, movieID := range movies.Value {
				args = append(args, movieID)
			}

			qb.moviesRepository().join(f, "movies_join", "scenes.id")
			f.addJoin("movies", "", "movies_join.movie_id = movies.id")
			whereClause, havingClause := getMultiCriterionClause("scenes", "movies", "movies_scenes", "scene_id", "movie_id", movies)
			f.addWhere(whereClause, args...)
			f.addHaving(havingClause)
		}
	}
}

func sceneStashIDsHandler(qb *sceneQueryBuilder, stashID *string) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if stashID != nil && *stashID != "" {
			qb.stashIDRepository().join(f, "scene_stash_ids", "scenes.id")
			stringLiteralCriterionHandler(stashID, "scene_stash_ids.stash_id")(f)
		}
	}
}

func (qb *sceneQueryBuilder) getSceneSort(findFilter *models.FindFilterType) string {
	if findFilter == nil {
		return " ORDER BY scenes.path, scenes.date ASC "
	}
	sort := findFilter.GetSort("title")
	direction := findFilter.GetDirection()
	return getSort(sort, direction, "scenes")
}

func (qb *sceneQueryBuilder) queryScene(query string, args []interface{}) (*models.Scene, error) {
	results, err := qb.queryScenes(query, args)
	if err != nil || len(results) < 1 {
		return nil, err
	}
	return results[0], nil
}

func (qb *sceneQueryBuilder) queryScenes(query string, args []interface{}) ([]*models.Scene, error) {
	var ret models.Scenes
	if err := qb.query(query, args, &ret); err != nil {
		return nil, err
	}

	return []*models.Scene(ret), nil
}

func (qb *sceneQueryBuilder) imageRepository() *imageRepository {
	return &imageRepository{
		repository: repository{
			tx:        qb.tx,
			tableName: "scenes_cover",
			idColumn:  sceneIDColumn,
		},
		imageColumn: "cover",
	}
}

func (qb *sceneQueryBuilder) GetCover(sceneID int) ([]byte, error) {
	return qb.imageRepository().get(sceneID)
}

func (qb *sceneQueryBuilder) UpdateCover(sceneID int, image []byte) error {
	return qb.imageRepository().replace(sceneID, image)
}

func (qb *sceneQueryBuilder) DestroyCover(sceneID int) error {
	return qb.imageRepository().destroy([]int{sceneID})
}

func (qb *sceneQueryBuilder) moviesRepository() *repository {
	return &repository{
		tx:        qb.tx,
		tableName: moviesScenesTable,
		idColumn:  sceneIDColumn,
	}
}

func (qb *sceneQueryBuilder) GetMovies(id int) (ret []models.MoviesScenes, err error) {
	if err := qb.moviesRepository().getAll(id, func(rows *sqlx.Rows) error {
		var ms models.MoviesScenes
		if err := rows.StructScan(&ms); err != nil {
			return err
		}

		ret = append(ret, ms)
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (qb *sceneQueryBuilder) UpdateMovies(sceneID int, movies []models.MoviesScenes) error {
	// destroy existing joins
	r := qb.moviesRepository()
	if err := r.destroy([]int{sceneID}); err != nil {
		return err
	}

	for _, m := range movies {
		m.SceneID = sceneID
		if _, err := r.insert(m); err != nil {
			return err
		}
	}

	return nil
}

func (qb *sceneQueryBuilder) performersRepository() *joinRepository {
	return &joinRepository{
		repository: repository{
			tx:        qb.tx,
			tableName: performersScenesTable,
			idColumn:  sceneIDColumn,
		},
		fkColumn: performerIDColumn,
	}
}

func (qb *sceneQueryBuilder) GetPerformerIDs(id int) ([]int, error) {
	return qb.performersRepository().getIDs(id)
}

func (qb *sceneQueryBuilder) UpdatePerformers(id int, performerIDs []int) error {
	// Delete the existing joins and then create new ones
	return qb.performersRepository().replace(id, performerIDs)
}

func (qb *sceneQueryBuilder) tagsRepository() *joinRepository {
	return &joinRepository{
		repository: repository{
			tx:        qb.tx,
			tableName: scenesTagsTable,
			idColumn:  sceneIDColumn,
		},
		fkColumn: tagIDColumn,
	}
}

func (qb *sceneQueryBuilder) GetTagIDs(id int) ([]int, error) {
	return qb.tagsRepository().getIDs(id)
}

func (qb *sceneQueryBuilder) UpdateTags(id int, tagIDs []int) error {
	// Delete the existing joins and then create new ones
	return qb.tagsRepository().replace(id, tagIDs)
}

func (qb *sceneQueryBuilder) galleriesRepository() *joinRepository {
	return &joinRepository{
		repository: repository{
			tx:        qb.tx,
			tableName: scenesGalleriesTable,
			idColumn:  sceneIDColumn,
		},
		fkColumn: galleryIDColumn,
	}
}

func (qb *sceneQueryBuilder) GetGalleryIDs(id int) ([]int, error) {
	return qb.galleriesRepository().getIDs(id)
}

func (qb *sceneQueryBuilder) UpdateGalleries(id int, galleryIDs []int) error {
	// Delete the existing joins and then create new ones
	return qb.galleriesRepository().replace(id, galleryIDs)
}

func (qb *sceneQueryBuilder) stashIDRepository() *stashIDRepository {
	return &stashIDRepository{
		repository{
			tx:        qb.tx,
			tableName: "scene_stash_ids",
			idColumn:  sceneIDColumn,
		},
	}
}

func (qb *sceneQueryBuilder) GetStashIDs(sceneID int) ([]*models.StashID, error) {
	return qb.stashIDRepository().get(sceneID)
}

func (qb *sceneQueryBuilder) UpdateStashIDs(sceneID int, stashIDs []models.StashID) error {
	return qb.stashIDRepository().replace(sceneID, stashIDs)
}

package sqlite

import (
	"database/sql"
	"fmt"
	"github.com/stashapp/stash/pkg/database"
	"github.com/stashapp/stash/pkg/models"
)

const sceneMarkerTable = "scene_markers"

const countSceneMarkersForTagQuery = `
SELECT scene_markers.id FROM scene_markers
LEFT JOIN scene_markers_tags as tags_join on tags_join.scene_marker_id = scene_markers.id
WHERE tags_join.tag_id = ? OR scene_markers.primary_tag_id = ?
GROUP BY scene_markers.id
`

type sceneMarkerQueryBuilder struct {
	repository
}

func NewSceneMarkerReaderWriter(tx dbi) *sceneMarkerQueryBuilder {
	return &sceneMarkerQueryBuilder{
		repository{
			tx:        tx,
			tableName: sceneMarkerTable,
			idColumn:  idColumn,
		},
	}
}

func (qb *sceneMarkerQueryBuilder) Create(newObject models.SceneMarker) (*models.SceneMarker, error) {
	var ret models.SceneMarker
	if err := qb.insertObject(newObject, &ret); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (qb *sceneMarkerQueryBuilder) Update(updatedObject models.SceneMarker) (*models.SceneMarker, error) {
	const partial = false
	if err := qb.update(updatedObject.ID, updatedObject, partial); err != nil {
		return nil, err
	}

	var ret models.SceneMarker
	if err := qb.get(updatedObject.ID, &ret); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (qb *sceneMarkerQueryBuilder) Destroy(id int) error {
	return qb.destroyExisting([]int{id})
}

func (qb *sceneMarkerQueryBuilder) Find(id int) (*models.SceneMarker, error) {
	query := "SELECT * FROM scene_markers WHERE id = ? LIMIT 1"
	args := []interface{}{id}
	results, err := qb.querySceneMarkers(query, args)
	if err != nil || len(results) < 1 {
		return nil, err
	}
	return results[0], nil
}

func (qb *sceneMarkerQueryBuilder) FindMany(ids []int) ([]*models.SceneMarker, error) {
	var markers []*models.SceneMarker
	for _, id := range ids {
		marker, err := qb.Find(id)
		if err != nil {
			return nil, err
		}

		if marker == nil {
			return nil, fmt.Errorf("scene marker with id %d not found", id)
		}

		markers = append(markers, marker)
	}

	return markers, nil
}

func (qb *sceneMarkerQueryBuilder) FindBySceneID(sceneID int) ([]*models.SceneMarker, error) {
	query := `
		SELECT scene_markers.* FROM scene_markers
		WHERE scene_markers.scene_id = ?
		GROUP BY scene_markers.id
		ORDER BY scene_markers.seconds ASC
	`
	args := []interface{}{sceneID}
	return qb.querySceneMarkers(query, args)
}

func (qb *sceneMarkerQueryBuilder) CountByTagID(tagID int) (int, error) {
	args := []interface{}{tagID, tagID}
	return qb.runCountQuery(qb.buildCountQuery(countSceneMarkersForTagQuery), args)
}

func (qb *sceneMarkerQueryBuilder) GetMarkerStrings(q *string, sort *string) ([]*models.MarkerStringsResultType, error) {
	query := "SELECT count(*) as `count`, scene_markers.id as id, scene_markers.title as title FROM scene_markers"
	if q != nil {
		query = query + " WHERE title LIKE '%" + *q + "%'"
	}
	query = query + " GROUP BY title"
	if sort != nil && *sort == "count" {
		query = query + " ORDER BY `count` DESC"
	} else {
		query = query + " ORDER BY title ASC"
	}
	var args []interface{}
	return qb.queryMarkerStringsResultType(query, args)
}

func (qb *sceneMarkerQueryBuilder) Wall(q *string) ([]*models.SceneMarker, error) {
	s := ""
	if q != nil {
		s = *q
	}
	query := "SELECT scene_markers.* FROM scene_markers WHERE scene_markers.title LIKE '%" + s + "%' ORDER BY RANDOM() LIMIT 80"
	return qb.querySceneMarkers(query, nil)
}

func (qb *sceneMarkerQueryBuilder) makeFilter(sceneMarkerFilter *models.SceneMarkerFilterType) *filterBuilder {
	query := &filterBuilder{}

	query.handleCriterion(sceneMarkerTagsCriterionHandler(qb, sceneMarkerFilter.Tags))
	query.handleCriterion(sceneMarkerSceneTagsCriterionHandler(qb, sceneMarkerFilter.SceneTags))
	query.handleCriterion(sceneMarkerPerformersCriterionHandler(qb, sceneMarkerFilter.Performers))

	return query
}

func (qb *sceneMarkerQueryBuilder) Query(sceneMarkerFilter *models.SceneMarkerFilterType, findFilter *models.FindFilterType) ([]*models.SceneMarker, int, error) {
	if sceneMarkerFilter == nil {
		sceneMarkerFilter = &models.SceneMarkerFilterType{}
	}
	if findFilter == nil {
		findFilter = &models.FindFilterType{}
	}

	query := qb.newQuery()

	query.body = selectDistinctIDs("scene_markers")

	if q := findFilter.Q; q != nil && *q != "" {
		searchColumns := []string{"scene_markers.title", "scene.title"}
		clause, thisArgs := getSearchBinding(searchColumns, *q, false)
		query.addWhere(clause)
		query.addArg(thisArgs...)
	}

	filter := qb.makeFilter(sceneMarkerFilter)

	query.addFilter(filter)

	query.sortAndPagination = qb.getSceneMarkerSort(findFilter) + getPagination(findFilter)
	idsResult, countResult, err := query.executeFind()
	if err != nil {
		return nil, 0, err
	}

	var sceneMarkers []*models.SceneMarker
	for _, id := range idsResult {
		sceneMarker, err := qb.Find(id)
		if err != nil {
			return nil, 0, err
		}

		sceneMarkers = append(sceneMarkers, sceneMarker)
	}

	return sceneMarkers, countResult, nil
}

func sceneMarkerTagsCriterionHandler(qb *sceneMarkerQueryBuilder, tags *models.HierarchicalMultiCriterionInput) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if tags != nil && len(tags.Value) > 0 {
			addHierarchicalWithClause(f, tags.Value, "tags_for_marker", tagTable, "tags_relations", "", tags.Depth)

			f.addWith(`marker_tags AS (
SELECT mt.scene_marker_id, t.id AS root_tag_id, t.child_id FROM scene_markers_tags mt
INNER JOIN tags_for_marker t ON t.child_id = mt.tag_id
UNION
SELECT m.id, t.id, t.child_id FROM scene_markers m
INNER JOIN tags_for_marker t	 ON t.child_id = m.primary_tag_id
)`)

			f.addJoin("marker_tags", "", "marker_tags.scene_marker_id = scene_markers.id")

			addHierarchicalConditionClauses(f, tags, "marker_tags", "root_tag_id")
		}
	}
}

func sceneMarkerSceneTagsCriterionHandler(qb *sceneMarkerQueryBuilder, tags *models.HierarchicalMultiCriterionInput) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if tags != nil && len(tags.Value) > 0 {
			addHierarchicalWithClause(f, tags.Value, "tags_for_scene", tagTable, "tags_relations", "", tags.Depth)

			f.addWith(`scene_tags AS (
SELECT st.scene_id, t.id AS root_tag_id FROM scenes_tags st
INNER JOIN tags_for_scene t ON t.child_id = st.tag_id
)`)

			f.addJoin("scene_tags", "", "scene_tags.scene_id = scene_markers.scene_id")

			addHierarchicalConditionClauses(f, tags, "scene_tags", "root_tag_id")
		}
	}
}

func sceneMarkerPerformersCriterionHandler(qb *sceneMarkerQueryBuilder, performers *models.MultiCriterionInput) criterionHandlerFunc {
	h := joinedMultiCriterionHandlerBuilder{
		primaryTable: sceneTable,
		joinTable:    performersScenesTable,
		joinAs:       "performers_join",
		primaryFK:    sceneIDColumn,
		foreignFK:    performerIDColumn,

		addJoinTable: func(f *filterBuilder) {
			f.addJoin(performersScenesTable, "performers_join", "performers_join.scene_id = scene_markers.scene_id")
		},
	}

	handler := h.handler(performers)
	return func(f *filterBuilder) {
		// Make sure scenes is included, otherwise excludes filter fails
		f.addJoin(sceneTable, "", "scenes.id = scene_markers.scene_id")
		handler(f)
	}
}

func (qb *sceneMarkerQueryBuilder) getSceneMarkerSort(findFilter *models.FindFilterType) string {
	sort := findFilter.GetSort("title")
	direction := findFilter.GetDirection()
	tableName := "scene_markers"
	if sort == "scenes_updated_at" {
		sort = "updated_at"
		tableName = "scene"
	}
	return getSort(sort, direction, tableName)
}

func (qb *sceneMarkerQueryBuilder) querySceneMarkers(query string, args []interface{}) ([]*models.SceneMarker, error) {
	var ret models.SceneMarkers
	if err := qb.query(query, args, &ret); err != nil {
		return nil, err
	}

	return []*models.SceneMarker(ret), nil
}

func (qb *sceneMarkerQueryBuilder) queryMarkerStringsResultType(query string, args []interface{}) ([]*models.MarkerStringsResultType, error) {
	rows, err := database.DB.Queryx(query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	defer rows.Close()

	markerStrings := make([]*models.MarkerStringsResultType, 0)
	for rows.Next() {
		markerString := models.MarkerStringsResultType{}
		if err := rows.StructScan(&markerString); err != nil {
			return nil, err
		}
		markerStrings = append(markerStrings, &markerString)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return markerStrings, nil
}

func (qb *sceneMarkerQueryBuilder) tagsRepository() *joinRepository {
	return &joinRepository{
		repository: repository{
			tx:        qb.tx,
			tableName: "scene_markers_tags",
			idColumn:  "scene_marker_id",
		},
		fkColumn: tagIDColumn,
	}
}

func (qb *sceneMarkerQueryBuilder) GetTagIDs(id int) ([]int, error) {
	return qb.tagsRepository().getIDs(id)
}

func (qb *sceneMarkerQueryBuilder) UpdateTags(id int, tagIDs []int) error {
	// Delete the existing joins and then create new ones
	return qb.tagsRepository().replace(id, tagIDs)
}

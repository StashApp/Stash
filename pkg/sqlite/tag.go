package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/stashapp/stash/pkg/models"
)

const tagTable = "tags"
const tagIDColumn = "tag_id"
const tagAliasesTable = "tag_aliases"
const tagAliasColumn = "alias"

type tagQueryBuilder struct {
	repository
}

func NewTagReaderWriter(tx dbi) *tagQueryBuilder {
	return &tagQueryBuilder{
		repository{
			tx:        tx,
			tableName: tagTable,
			idColumn:  idColumn,
		},
	}
}

func (qb *tagQueryBuilder) Create(newObject models.Tag) (*models.Tag, error) {
	var ret models.Tag
	if err := qb.insertObject(newObject, &ret); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (qb *tagQueryBuilder) Update(updatedObject models.TagPartial) (*models.Tag, error) {
	const partial = true
	if err := qb.update(updatedObject.ID, updatedObject, partial); err != nil {
		return nil, err
	}

	return qb.Find(updatedObject.ID)
}

func (qb *tagQueryBuilder) UpdateFull(updatedObject models.Tag) (*models.Tag, error) {
	const partial = false
	if err := qb.update(updatedObject.ID, updatedObject, partial); err != nil {
		return nil, err
	}

	return qb.Find(updatedObject.ID)
}

func (qb *tagQueryBuilder) Destroy(id int) error {
	// TODO - add delete cascade to foreign key
	// delete tag from scenes and markers first
	_, err := qb.tx.Exec("DELETE FROM scenes_tags WHERE tag_id = ?", id)
	if err != nil {
		return err
	}

	// TODO - add delete cascade to foreign key
	_, err = qb.tx.Exec("DELETE FROM scene_markers_tags WHERE tag_id = ?", id)
	if err != nil {
		return err
	}

	// cannot unset primary_tag_id in scene_markers because it is not nullable
	countQuery := "SELECT COUNT(*) as count FROM scene_markers where primary_tag_id = ?"
	args := []interface{}{id}
	primaryMarkers, err := qb.runCountQuery(countQuery, args)
	if err != nil {
		return err
	}

	if primaryMarkers > 0 {
		return errors.New("Cannot delete tag used as a primary tag in scene markers")
	}

	return qb.destroyExisting([]int{id})
}

func (qb *tagQueryBuilder) Find(id int) (*models.Tag, error) {
	var ret models.Tag
	if err := qb.get(id, &ret); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ret, nil
}

func (qb *tagQueryBuilder) FindMany(ids []int) ([]*models.Tag, error) {
	var tags []*models.Tag
	for _, id := range ids {
		tag, err := qb.Find(id)
		if err != nil {
			return nil, err
		}

		if tag == nil {
			return nil, fmt.Errorf("tag with id %d not found", id)
		}

		tags = append(tags, tag)
	}

	return tags, nil
}

func (qb *tagQueryBuilder) FindBySceneID(sceneID int) ([]*models.Tag, error) {
	query := `
		SELECT tags.* FROM tags
		LEFT JOIN scenes_tags as scenes_join on scenes_join.tag_id = tags.id
		WHERE scenes_join.scene_id = ?
		GROUP BY tags.id
	`
	query += qb.getDefaultTagSort()
	args := []interface{}{sceneID}
	return qb.queryTags(query, args)
}

func (qb *tagQueryBuilder) FindByPerformerID(performerID int) ([]*models.Tag, error) {
	query := `
		SELECT tags.* FROM tags
		LEFT JOIN performers_tags as performers_join on performers_join.tag_id = tags.id
		WHERE performers_join.performer_id = ?
		GROUP BY tags.id
	`
	query += qb.getDefaultTagSort()
	args := []interface{}{performerID}
	return qb.queryTags(query, args)
}

func (qb *tagQueryBuilder) FindByImageID(imageID int) ([]*models.Tag, error) {
	query := `
		SELECT tags.* FROM tags
		LEFT JOIN images_tags as images_join on images_join.tag_id = tags.id
		WHERE images_join.image_id = ?
		GROUP BY tags.id
	`
	query += qb.getDefaultTagSort()
	args := []interface{}{imageID}
	return qb.queryTags(query, args)
}

func (qb *tagQueryBuilder) FindByGalleryID(galleryID int) ([]*models.Tag, error) {
	query := `
		SELECT tags.* FROM tags
		LEFT JOIN galleries_tags as galleries_join on galleries_join.tag_id = tags.id
		WHERE galleries_join.gallery_id = ?
		GROUP BY tags.id
	`
	query += qb.getDefaultTagSort()
	args := []interface{}{galleryID}
	return qb.queryTags(query, args)
}

func (qb *tagQueryBuilder) FindBySceneMarkerID(sceneMarkerID int) ([]*models.Tag, error) {
	query := `
		SELECT tags.* FROM tags
		LEFT JOIN scene_markers_tags as scene_markers_join on scene_markers_join.tag_id = tags.id
		WHERE scene_markers_join.scene_marker_id = ?
		GROUP BY tags.id
	`
	query += qb.getDefaultTagSort()
	args := []interface{}{sceneMarkerID}
	return qb.queryTags(query, args)
}

func (qb *tagQueryBuilder) FindByName(name string, nocase bool) (*models.Tag, error) {
	query := "SELECT * FROM tags WHERE name = ?"
	if nocase {
		query += " COLLATE NOCASE"
	}
	query += " LIMIT 1"
	args := []interface{}{name}
	return qb.queryTag(query, args)
}

func (qb *tagQueryBuilder) FindByNames(names []string, nocase bool) ([]*models.Tag, error) {
	query := "SELECT * FROM tags WHERE name"
	if nocase {
		query += " COLLATE NOCASE"
	}
	query += " IN " + getInBinding(len(names))
	var args []interface{}
	for _, name := range names {
		args = append(args, name)
	}
	return qb.queryTags(query, args)
}

func (qb *tagQueryBuilder) Count() (int, error) {
	return qb.runCountQuery(qb.buildCountQuery("SELECT tags.id FROM tags"), nil)
}

func (qb *tagQueryBuilder) All() ([]*models.Tag, error) {
	return qb.queryTags(selectAll("tags")+qb.getDefaultTagSort(), nil)
}

func (qb *tagQueryBuilder) QueryForAutoTag(words []string) ([]*models.Tag, error) {
	// TODO - Query needs to be changed to support queries of this type, and
	// this method should be removed
	query := selectAll(tagTable)
	query += " LEFT JOIN tag_aliases ON tag_aliases.tag_id = tags.id"

	var whereClauses []string
	var args []interface{}

	for _, w := range words {
		ww := w + "%"
		whereClauses = append(whereClauses, "tags.name like ?")
		args = append(args, ww)

		// include aliases
		whereClauses = append(whereClauses, "tag_aliases.alias like ?")
		args = append(args, ww)
	}

	where := strings.Join(whereClauses, " OR ")
	return qb.queryTags(query+" WHERE "+where, args)
}

func (qb *tagQueryBuilder) validateFilter(tagFilter *models.TagFilterType) error {
	const and = "AND"
	const or = "OR"
	const not = "NOT"

	if tagFilter.And != nil {
		if tagFilter.Or != nil {
			return illegalFilterCombination(and, or)
		}
		if tagFilter.Not != nil {
			return illegalFilterCombination(and, not)
		}

		return qb.validateFilter(tagFilter.And)
	}

	if tagFilter.Or != nil {
		if tagFilter.Not != nil {
			return illegalFilterCombination(or, not)
		}

		return qb.validateFilter(tagFilter.Or)
	}

	if tagFilter.Not != nil {
		return qb.validateFilter(tagFilter.Not)
	}

	return nil
}

func (qb *tagQueryBuilder) makeFilter(tagFilter *models.TagFilterType) *filterBuilder {
	query := &filterBuilder{}

	if tagFilter.And != nil {
		query.and(qb.makeFilter(tagFilter.And))
	}
	if tagFilter.Or != nil {
		query.or(qb.makeFilter(tagFilter.Or))
	}
	if tagFilter.Not != nil {
		query.not(qb.makeFilter(tagFilter.Not))
	}

	// if markerCount := tagFilter.MarkerCount; markerCount != nil {
	// 	clause, count := getIntCriterionWhereClause("count(distinct scene_markers.id)", *markerCount)
	// 	query.addHaving(clause)
	// 	if count == 1 {
	// 		query.addArg(markerCount.Value)
	// 	}
	// }

	query.handleCriterion(stringCriterionHandler(tagFilter.Name, tagTable+".name"))
	query.handleCriterion(tagAliasCriterionHandler(qb, tagFilter.Aliases))

	query.handleCriterion(tagIsMissingCriterionHandler(qb, tagFilter.IsMissing))
	query.handleCriterion(tagSceneCountCriterionHandler(qb, tagFilter.SceneCount))
	query.handleCriterion(tagImageCountCriterionHandler(qb, tagFilter.ImageCount))
	query.handleCriterion(tagGalleryCountCriterionHandler(qb, tagFilter.GalleryCount))
	query.handleCriterion(tagPerformerCountCriterionHandler(qb, tagFilter.PerformerCount))

	return query
}

func (qb *tagQueryBuilder) Query(tagFilter *models.TagFilterType, findFilter *models.FindFilterType) ([]*models.Tag, int, error) {
	if tagFilter == nil {
		tagFilter = &models.TagFilterType{}
	}
	if findFilter == nil {
		findFilter = &models.FindFilterType{}
	}

	query := qb.newQuery()

	query.body = selectDistinctIDs(tagTable)

	/*
		query.body += `
		left join tags_image on tags_image.tag_id = tags.id
		left join scenes_tags on scenes_tags.tag_id = tags.id
		left join scene_markers_tags on scene_markers_tags.tag_id = tags.id
		left join scene_markers on scene_markers.primary_tag_id = tags.id OR scene_markers.id = scene_markers_tags.scene_marker_id
		left join scenes on scenes_tags.scene_id = scenes.id`
	*/

	// the presence of joining on scene_markers.primary_tag_id and scene_markers_tags.tag_id
	// appears to confuse sqlite and causes serious performance issues.
	// Disabling querying/sorting on marker count for now.

	if q := findFilter.Q; q != nil && *q != "" {
		query.join(tagAliasesTable, "", "tag_aliases.tag_id = tags.id")
		searchColumns := []string{"tags.name", "tag_aliases.alias"}
		clause, thisArgs := getSearchBinding(searchColumns, *q, false)
		query.addWhere(clause)
		query.addArg(thisArgs...)
	}

	if err := qb.validateFilter(tagFilter); err != nil {
		return nil, 0, err
	}
	filter := qb.makeFilter(tagFilter)

	query.addFilter(filter)

	query.sortAndPagination = qb.getTagSort(&query, findFilter) + getPagination(findFilter)
	idsResult, countResult, err := query.executeFind()
	if err != nil {
		return nil, 0, err
	}

	var tags []*models.Tag
	for _, id := range idsResult {
		tag, err := qb.Find(id)
		if err != nil {
			return nil, 0, err
		}
		tags = append(tags, tag)
	}

	return tags, countResult, nil
}

func tagAliasCriterionHandler(qb *tagQueryBuilder, alias *models.StringCriterionInput) criterionHandlerFunc {
	h := stringListCriterionHandlerBuilder{
		joinTable:    tagAliasesTable,
		stringColumn: tagAliasColumn,
		addJoinTable: func(f *filterBuilder) {
			qb.aliasRepository().join(f, "", "tags.id")
		},
	}

	return h.handler(alias)
}

func tagIsMissingCriterionHandler(qb *tagQueryBuilder, isMissing *string) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if isMissing != nil && *isMissing != "" {
			switch *isMissing {
			case "image":
				qb.imageRepository().join(f, "", "tags.id")
				f.addWhere("tags_image.tag_id IS NULL")
			default:
				f.addWhere("(tags." + *isMissing + " IS NULL OR TRIM(tags." + *isMissing + ") = '')")
			}
		}
	}
}

func tagSceneCountCriterionHandler(qb *tagQueryBuilder, sceneCount *models.IntCriterionInput) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if sceneCount != nil {
			f.addJoin("scenes_tags", "", "scenes_tags.tag_id = tags.id")
			clause, args := getIntCriterionWhereClause("count(distinct scenes_tags.scene_id)", *sceneCount)

			f.addHaving(clause, args...)
		}
	}
}

func tagImageCountCriterionHandler(qb *tagQueryBuilder, imageCount *models.IntCriterionInput) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if imageCount != nil {
			f.addJoin("images_tags", "", "images_tags.tag_id = tags.id")
			clause, args := getIntCriterionWhereClause("count(distinct images_tags.image_id)", *imageCount)

			f.addHaving(clause, args...)
		}
	}
}

func tagGalleryCountCriterionHandler(qb *tagQueryBuilder, galleryCount *models.IntCriterionInput) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if galleryCount != nil {
			f.addJoin("galleries_tags", "", "galleries_tags.tag_id = tags.id")
			clause, args := getIntCriterionWhereClause("count(distinct galleries_tags.gallery_id)", *galleryCount)

			f.addHaving(clause, args...)
		}
	}
}

func tagPerformerCountCriterionHandler(qb *tagQueryBuilder, performerCount *models.IntCriterionInput) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if performerCount != nil {
			f.addJoin("performers_tags", "", "performers_tags.tag_id = tags.id")
			clause, args := getIntCriterionWhereClause("count(distinct performers_tags.performer_id)", *performerCount)

			f.addHaving(clause, args...)
		}
	}
}

func (qb *tagQueryBuilder) getDefaultTagSort() string {
	return getSort("name", "ASC", "tags")
}

func (qb *tagQueryBuilder) getTagSort(query *queryBuilder, findFilter *models.FindFilterType) string {
	var sort string
	var direction string
	if findFilter == nil {
		sort = "name"
		direction = "ASC"
	} else {
		sort = findFilter.GetSort("name")
		direction = findFilter.GetDirection()
	}

	if findFilter.Sort != nil {
		switch *findFilter.Sort {
		case "scenes_count":
			query.join("scenes_tags", "", "scenes_tags.tag_id = tags.id")
			return " ORDER BY COUNT(distinct scenes_tags.scene_id) " + direction
		case "images_count":
			query.join("images_tags", "", "images_tags.tag_id = tags.id")
			return " ORDER BY COUNT(distinct images_tags.image_id) " + direction
		case "galleries_count":
			query.join("galleries_tags", "", "galleries_tags.tag_id = tags.id")
			return " ORDER BY COUNT(distinct galleries_tags.gallery_id) " + direction
		case "performers_count":
			query.join("performers_tags", "", "performers_tags.tag_id = tags.id")
			return " ORDER BY COUNT(distinct performers_tags.performer_id) " + direction
		}
	}

	return getSort(sort, direction, "tags")
}

func (qb *tagQueryBuilder) queryTag(query string, args []interface{}) (*models.Tag, error) {
	results, err := qb.queryTags(query, args)
	if err != nil || len(results) < 1 {
		return nil, err
	}
	return results[0], nil
}

func (qb *tagQueryBuilder) queryTags(query string, args []interface{}) ([]*models.Tag, error) {
	var ret models.Tags
	if err := qb.query(query, args, &ret); err != nil {
		return nil, err
	}

	return []*models.Tag(ret), nil
}

func (qb *tagQueryBuilder) imageRepository() *imageRepository {
	return &imageRepository{
		repository: repository{
			tx:        qb.tx,
			tableName: "tags_image",
			idColumn:  tagIDColumn,
		},
		imageColumn: "image",
	}
}

func (qb *tagQueryBuilder) GetImage(tagID int) ([]byte, error) {
	return qb.imageRepository().get(tagID)
}

func (qb *tagQueryBuilder) HasImage(tagID int) (bool, error) {
	return qb.imageRepository().exists(tagID)
}

func (qb *tagQueryBuilder) UpdateImage(tagID int, image []byte) error {
	return qb.imageRepository().replace(tagID, image)
}

func (qb *tagQueryBuilder) DestroyImage(tagID int) error {
	return qb.imageRepository().destroy([]int{tagID})
}

func (qb *tagQueryBuilder) aliasRepository() *stringRepository {
	return &stringRepository{
		repository: repository{
			tx:        qb.tx,
			tableName: tagAliasesTable,
			idColumn:  tagIDColumn,
		},
		stringColumn: tagAliasColumn,
	}
}

func (qb *tagQueryBuilder) GetAliases(tagID int) ([]string, error) {
	return qb.aliasRepository().get(tagID)
}

func (qb *tagQueryBuilder) UpdateAliases(tagID int, aliases []string) error {
	return qb.aliasRepository().replace(tagID, aliases)
}

func (qb *tagQueryBuilder) Merge(source []int, destination int) error {
	if len(source) == 0 {
		return nil
	}

	inBinding := getInBinding(len(source))

	args := []interface{}{destination}
	for _, id := range source {
		if id == destination {
			return errors.New("cannot merge where source == destination")
		}
		args = append(args, id)
	}

	tagTables := map[string]string{
		scenesTagsTable:      sceneIDColumn,
		"scene_markers_tags": "scene_marker_id",
		galleriesTagsTable:   galleryIDColumn,
		imagesTagsTable:      imageIDColumn,
		"performers_tags":    "performer_id",
	}

	tagArgs := append(args, destination)
	for table, idColumn := range tagTables {
		_, err := qb.tx.Exec(`UPDATE `+table+`
SET tag_id = ?
WHERE tag_id IN `+inBinding+`
AND NOT EXISTS(SELECT 1 FROM `+table+` o WHERE o.`+idColumn+` = `+table+`.`+idColumn+` AND o.tag_id = ?)`,
			tagArgs...,
		)
		if err != nil {
			return err
		}
	}

	_, err := qb.tx.Exec("UPDATE "+sceneMarkerTable+" SET primary_tag_id = ? WHERE primary_tag_id IN "+inBinding, args...)
	if err != nil {
		return err
	}

	_, err = qb.tx.Exec("INSERT INTO "+tagAliasesTable+" (tag_id, alias) SELECT ?, name FROM "+tagTable+" WHERE id IN "+inBinding, args...)
	if err != nil {
		return err
	}

	_, err = qb.tx.Exec("UPDATE "+tagAliasesTable+" SET tag_id = ? WHERE tag_id IN "+inBinding, args...)
	if err != nil {
		return err
	}

	for _, id := range source {
		err = qb.Destroy(id)
		if err != nil {
			return err
		}
	}

	return nil
}

package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/stashapp/stash/pkg/models"
)

const galleryTable = "galleries"

const performersGalleriesTable = "performers_galleries"
const galleriesTagsTable = "galleries_tags"
const galleriesImagesTable = "galleries_images"
const galleriesScenesTable = "scenes_galleries"
const galleryIDColumn = "gallery_id"

type galleryQueryBuilder struct {
	repository
}

func NewGalleryReaderWriter(tx dbi) *galleryQueryBuilder {
	return &galleryQueryBuilder{
		repository{
			tx:        tx,
			tableName: galleryTable,
			idColumn:  idColumn,
		},
	}
}

func (qb *galleryQueryBuilder) Create(newObject models.Gallery) (*models.Gallery, error) {
	var ret models.Gallery
	if err := qb.insertObject(newObject, &ret); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (qb *galleryQueryBuilder) Update(updatedObject models.Gallery) (*models.Gallery, error) {
	const partial = false
	if err := qb.update(updatedObject.ID, updatedObject, partial); err != nil {
		return nil, err
	}

	return qb.Find(updatedObject.ID)
}

func (qb *galleryQueryBuilder) UpdatePartial(updatedObject models.GalleryPartial) (*models.Gallery, error) {
	const partial = true
	if err := qb.update(updatedObject.ID, updatedObject, partial); err != nil {
		return nil, err
	}

	return qb.Find(updatedObject.ID)
}

func (qb *galleryQueryBuilder) UpdateChecksum(id int, checksum string) error {
	return qb.updateMap(id, map[string]interface{}{
		"checksum": checksum,
	})
}

func (qb *galleryQueryBuilder) UpdateFileModTime(id int, modTime models.NullSQLiteTimestamp) error {
	return qb.updateMap(id, map[string]interface{}{
		"file_mod_time": modTime,
	})
}

func (qb *galleryQueryBuilder) Destroy(id int) error {
	return qb.destroyExisting([]int{id})
}

func (qb *galleryQueryBuilder) Find(id int) (*models.Gallery, error) {
	var ret models.Gallery
	if err := qb.get(id, &ret); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ret, nil
}

func (qb *galleryQueryBuilder) FindMany(ids []int) ([]*models.Gallery, error) {
	var galleries []*models.Gallery
	for _, id := range ids {
		gallery, err := qb.Find(id)
		if err != nil {
			return nil, err
		}

		if gallery == nil {
			return nil, fmt.Errorf("gallery with id %d not found", id)
		}

		galleries = append(galleries, gallery)
	}

	return galleries, nil
}

func (qb *galleryQueryBuilder) FindByChecksum(checksum string) (*models.Gallery, error) {
	query := "SELECT * FROM galleries WHERE checksum = ? LIMIT 1"
	args := []interface{}{checksum}
	return qb.queryGallery(query, args)
}

func (qb *galleryQueryBuilder) FindByChecksums(checksums []string) ([]*models.Gallery, error) {
	query := "SELECT * FROM galleries WHERE checksum IN " + getInBinding(len(checksums))
	var args []interface{}
	for _, checksum := range checksums {
		args = append(args, checksum)
	}
	return qb.queryGalleries(query, args)
}

func (qb *galleryQueryBuilder) FindByPath(path string) (*models.Gallery, error) {
	query := "SELECT * FROM galleries WHERE path = ? LIMIT 1"
	args := []interface{}{path}
	return qb.queryGallery(query, args)
}

func (qb *galleryQueryBuilder) FindBySceneID(sceneID int) ([]*models.Gallery, error) {
	query := selectAll(galleryTable) + `
		LEFT JOIN scenes_galleries as scenes_join on scenes_join.gallery_id = galleries.id
		WHERE scenes_join.scene_id = ?
		GROUP BY galleries.id
	`
	args := []interface{}{sceneID}
	return qb.queryGalleries(query, args)
}

func (qb *galleryQueryBuilder) FindByImageID(imageID int) ([]*models.Gallery, error) {
	query := selectAll(galleryTable) + `
	LEFT JOIN galleries_images as images_join on images_join.gallery_id = galleries.id
	WHERE images_join.image_id = ?
	GROUP BY galleries.id
	`
	args := []interface{}{imageID}
	return qb.queryGalleries(query, args)
}

func (qb *galleryQueryBuilder) CountByImageID(imageID int) (int, error) {
	query := `SELECT image_id FROM galleries_images
	WHERE image_id = ?
	GROUP BY gallery_id`
	args := []interface{}{imageID}
	return qb.runCountQuery(qb.buildCountQuery(query), args)
}

func (qb *galleryQueryBuilder) Count() (int, error) {
	return qb.runCountQuery(qb.buildCountQuery("SELECT galleries.id FROM galleries"), nil)
}

func (qb *galleryQueryBuilder) All() ([]*models.Gallery, error) {
	return qb.queryGalleries(selectAll("galleries")+qb.getGallerySort(nil), nil)
}

func (qb *galleryQueryBuilder) validateFilter(galleryFilter *models.GalleryFilterType) error {
	const and = "AND"
	const or = "OR"
	const not = "NOT"

	if galleryFilter.And != nil {
		if galleryFilter.Or != nil {
			return illegalFilterCombination(and, or)
		}
		if galleryFilter.Not != nil {
			return illegalFilterCombination(and, not)
		}

		return qb.validateFilter(galleryFilter.And)
	}

	if galleryFilter.Or != nil {
		if galleryFilter.Not != nil {
			return illegalFilterCombination(or, not)
		}

		return qb.validateFilter(galleryFilter.Or)
	}

	if galleryFilter.Not != nil {
		return qb.validateFilter(galleryFilter.Not)
	}

	return nil
}

func (qb *galleryQueryBuilder) makeFilter(galleryFilter *models.GalleryFilterType) *filterBuilder {
	query := &filterBuilder{}

	if galleryFilter.And != nil {
		query.and(qb.makeFilter(galleryFilter.And))
	}
	if galleryFilter.Or != nil {
		query.or(qb.makeFilter(galleryFilter.Or))
	}
	if galleryFilter.Not != nil {
		query.not(qb.makeFilter(galleryFilter.Not))
	}

	query.handleCriterion(stringCriterionHandler(galleryFilter.Title, "galleries.title"))
	query.handleCriterion(stringCriterionHandler(galleryFilter.Details, "galleries.details"))
	query.handleCriterion(stringCriterionHandler(galleryFilter.Checksum, "galleries.checksum"))
	query.handleCriterion(boolCriterionHandler(galleryFilter.IsZip, "galleries.zip"))
	query.handleCriterion(stringCriterionHandler(galleryFilter.Path, "galleries.path"))
	query.handleCriterion(intCriterionHandler(galleryFilter.Rating, "galleries.rating"))
	query.handleCriterion(stringCriterionHandler(galleryFilter.URL, "galleries.url"))
	query.handleCriterion(boolCriterionHandler(galleryFilter.Organized, "galleries.organized"))
	query.handleCriterion(galleryIsMissingCriterionHandler(qb, galleryFilter.IsMissing))
	query.handleCriterion(galleryTagsCriterionHandler(qb, galleryFilter.Tags))
	query.handleCriterion(galleryTagCountCriterionHandler(qb, galleryFilter.TagCount))
	query.handleCriterion(galleryPerformersCriterionHandler(qb, galleryFilter.Performers))
	query.handleCriterion(galleryPerformerCountCriterionHandler(qb, galleryFilter.PerformerCount))
	query.handleCriterion(galleryStudioCriterionHandler(qb, galleryFilter.Studios))
	query.handleCriterion(galleryPerformerTagsCriterionHandler(qb, galleryFilter.PerformerTags))
	query.handleCriterion(galleryAverageResolutionCriterionHandler(qb, galleryFilter.AverageResolution))
	query.handleCriterion(galleryImageCountCriterionHandler(qb, galleryFilter.ImageCount))

	return query
}

func (qb *galleryQueryBuilder) makeQuery(galleryFilter *models.GalleryFilterType, findFilter *models.FindFilterType) (*queryBuilder, error) {
	if galleryFilter == nil {
		galleryFilter = &models.GalleryFilterType{}
	}
	if findFilter == nil {
		findFilter = &models.FindFilterType{}
	}

	query := qb.newQuery()

	query.body = selectDistinctIDs(galleryTable)

	if q := findFilter.Q; q != nil && *q != "" {
		searchColumns := []string{"galleries.title", "galleries.path", "galleries.checksum"}
		clause, thisArgs := getSearchBinding(searchColumns, *q, false)
		query.addWhere(clause)
		query.addArg(thisArgs...)
	}

	if err := qb.validateFilter(galleryFilter); err != nil {
		return nil, err
	}
	filter := qb.makeFilter(galleryFilter)

	query.addFilter(filter)

	query.sortAndPagination = qb.getGallerySort(findFilter) + getPagination(findFilter)

	return &query, nil
}

func (qb *galleryQueryBuilder) Query(galleryFilter *models.GalleryFilterType, findFilter *models.FindFilterType) ([]*models.Gallery, int, error) {
	query, err := qb.makeQuery(galleryFilter, findFilter)
	if err != nil {
		return nil, 0, err
	}

	idsResult, countResult, err := query.executeFind()
	if err != nil {
		return nil, 0, err
	}

	var galleries []*models.Gallery
	for _, id := range idsResult {
		gallery, err := qb.Find(id)
		if err != nil {
			return nil, 0, err
		}

		galleries = append(galleries, gallery)
	}

	return galleries, countResult, nil
}

func (qb *galleryQueryBuilder) QueryCount(galleryFilter *models.GalleryFilterType, findFilter *models.FindFilterType) (int, error) {
	query, err := qb.makeQuery(galleryFilter, findFilter)
	if err != nil {
		return 0, err
	}

	return query.executeCount()
}

func galleryIsMissingCriterionHandler(qb *galleryQueryBuilder, isMissing *string) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if isMissing != nil && *isMissing != "" {
			switch *isMissing {
			case "scenes":
				f.addJoin("scenes_galleries", "scenes_join", "scenes_join.gallery_id = galleries.id")
				f.addWhere("scenes_join.gallery_id IS NULL")
			case "studio":
				f.addWhere("galleries.studio_id IS NULL")
			case "performers":
				qb.performersRepository().join(f, "performers_join", "galleries.id")
				f.addWhere("performers_join.gallery_id IS NULL")
			case "date":
				f.addWhere("galleries.date IS \"\" OR galleries.date IS \"0001-01-01\"")
			case "tags":
				qb.tagsRepository().join(f, "tags_join", "galleries.id")
				f.addWhere("tags_join.gallery_id IS NULL")
			default:
				f.addWhere("(galleries." + *isMissing + " IS NULL OR TRIM(galleries." + *isMissing + ") = '')")
			}
		}
	}
}

func (qb *galleryQueryBuilder) getMultiCriterionHandlerBuilder(foreignTable, joinTable, foreignFK string, addJoinsFunc func(f *filterBuilder)) multiCriterionHandlerBuilder {
	return multiCriterionHandlerBuilder{
		primaryTable: galleryTable,
		foreignTable: foreignTable,
		joinTable:    joinTable,
		primaryFK:    galleryIDColumn,
		foreignFK:    foreignFK,
		addJoinsFunc: addJoinsFunc,
	}
}

func galleryTagsCriterionHandler(qb *galleryQueryBuilder, tags *models.MultiCriterionInput) criterionHandlerFunc {
	h := joinedMultiCriterionHandlerBuilder{
		primaryTable: galleryTable,
		joinTable:    galleriesTagsTable,
		joinAs:       "tags_join",
		primaryFK:    galleryIDColumn,
		foreignFK:    tagIDColumn,

		addJoinTable: func(f *filterBuilder) {
			qb.tagsRepository().join(f, "tags_join", "galleries.id")
		},
	}

	return h.handler(tags)
}

func galleryTagCountCriterionHandler(qb *galleryQueryBuilder, tagCount *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: galleryTable,
		joinTable:    galleriesTagsTable,
		primaryFK:    galleryIDColumn,
	}

	return h.handler(tagCount)
}

func galleryPerformersCriterionHandler(qb *galleryQueryBuilder, performers *models.MultiCriterionInput) criterionHandlerFunc {
	h := joinedMultiCriterionHandlerBuilder{
		primaryTable: galleryTable,
		joinTable:    performersGalleriesTable,
		joinAs:       "performers_join",
		primaryFK:    galleryIDColumn,
		foreignFK:    performerIDColumn,

		addJoinTable: func(f *filterBuilder) {
			qb.performersRepository().join(f, "performers_join", "galleries.id")
		},
	}

	return h.handler(performers)
}

func galleryPerformerCountCriterionHandler(qb *galleryQueryBuilder, performerCount *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: galleryTable,
		joinTable:    performersGalleriesTable,
		primaryFK:    galleryIDColumn,
	}

	return h.handler(performerCount)
}

func galleryImageCountCriterionHandler(qb *galleryQueryBuilder, imageCount *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: galleryTable,
		joinTable:    galleriesImagesTable,
		primaryFK:    galleryIDColumn,
	}

	return h.handler(imageCount)
}

func galleryStudioCriterionHandler(qb *galleryQueryBuilder, studios *models.HierarchicalMultiCriterionInput) criterionHandlerFunc {
	h := hierarchicalMultiCriterionHandlerBuilder{
		primaryTable: galleryTable,
		foreignTable: studioTable,
		foreignFK:    studioIDColumn,
		derivedTable: "studio",
		parentFK:     "parent_id",
	}

	return h.handler(studios)
}

func galleryPerformerTagsCriterionHandler(qb *galleryQueryBuilder, performerTagsFilter *models.MultiCriterionInput) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if performerTagsFilter != nil && len(performerTagsFilter.Value) > 0 {
			qb.performersRepository().join(f, "performers_join", "galleries.id")
			f.addJoin("performers_tags", "performer_tags_join", "performers_join.performer_id = performer_tags_join.performer_id")

			var args []interface{}
			for _, tagID := range performerTagsFilter.Value {
				args = append(args, tagID)
			}

			if performerTagsFilter.Modifier == models.CriterionModifierIncludes {
				// includes any of the provided ids
				f.addWhere("performer_tags_join.tag_id IN "+getInBinding(len(performerTagsFilter.Value)), args...)
			} else if performerTagsFilter.Modifier == models.CriterionModifierIncludesAll {
				// includes all of the provided ids
				f.addWhere("performer_tags_join.tag_id IN "+getInBinding(len(performerTagsFilter.Value)), args...)
				f.addHaving(fmt.Sprintf("count(distinct performer_tags_join.tag_id) IS %d", len(performerTagsFilter.Value)))
			} else if performerTagsFilter.Modifier == models.CriterionModifierExcludes {
				f.addWhere(fmt.Sprintf(`not exists
					(select performers_galleries.performer_id from performers_galleries
						left join performers_tags on performers_tags.performer_id = performers_galleries.performer_id where
						performers_galleries.gallery_id = galleries.id AND
						performers_tags.tag_id in %s)`, getInBinding(len(performerTagsFilter.Value))), args...)
			}
		}
	}
}

func galleryAverageResolutionCriterionHandler(qb *galleryQueryBuilder, resolution *models.ResolutionCriterionInput) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if resolution != nil && resolution.Value.IsValid() {
			qb.imagesRepository().join(f, "images_join", "galleries.id")
			f.addJoin("images", "", "images_join.image_id = images.id")

			min := resolution.Value.GetMinResolution()
			max := resolution.Value.GetMaxResolution()

			const widthHeight = "avg(MIN(images.width, images.height))"

			if resolution.Modifier == models.CriterionModifierEquals {
				f.addHaving(fmt.Sprintf("%s BETWEEN %d AND %d", widthHeight, min, max))
			} else if resolution.Modifier == models.CriterionModifierNotEquals {
				f.addHaving(fmt.Sprintf("%s NOT BETWEEN %d AND %d", widthHeight, min, max))
			} else if resolution.Modifier == models.CriterionModifierLessThan {
				f.addHaving(fmt.Sprintf("%s < %d", widthHeight, min))
			} else if resolution.Modifier == models.CriterionModifierGreaterThan {
				f.addHaving(fmt.Sprintf("%s > %d", widthHeight, max))
			}
		}
	}
}

func (qb *galleryQueryBuilder) getGallerySort(findFilter *models.FindFilterType) string {
	var sort string
	var direction string
	if findFilter == nil {
		sort = "path"
		direction = "ASC"
	} else {
		sort = findFilter.GetSort("path")
		direction = findFilter.GetDirection()
	}

	switch sort {
	case "images_count":
		return getCountSort(galleryTable, galleriesImagesTable, galleryIDColumn, direction)
	case "tag_count":
		return getCountSort(galleryTable, galleriesTagsTable, galleryIDColumn, direction)
	case "performer_count":
		return getCountSort(galleryTable, performersGalleriesTable, galleryIDColumn, direction)
	default:
		return getSort(sort, direction, "galleries")
	}
}

func (qb *galleryQueryBuilder) queryGallery(query string, args []interface{}) (*models.Gallery, error) {
	results, err := qb.queryGalleries(query, args)
	if err != nil || len(results) < 1 {
		return nil, err
	}
	return results[0], nil
}

func (qb *galleryQueryBuilder) queryGalleries(query string, args []interface{}) ([]*models.Gallery, error) {
	var ret models.Galleries
	if err := qb.query(query, args, &ret); err != nil {
		return nil, err
	}

	return []*models.Gallery(ret), nil
}

func (qb *galleryQueryBuilder) performersRepository() *joinRepository {
	return &joinRepository{
		repository: repository{
			tx:        qb.tx,
			tableName: performersGalleriesTable,
			idColumn:  galleryIDColumn,
		},
		fkColumn: "performer_id",
	}
}

func (qb *galleryQueryBuilder) GetPerformerIDs(galleryID int) ([]int, error) {
	return qb.performersRepository().getIDs(galleryID)
}

func (qb *galleryQueryBuilder) UpdatePerformers(galleryID int, performerIDs []int) error {
	// Delete the existing joins and then create new ones
	return qb.performersRepository().replace(galleryID, performerIDs)
}

func (qb *galleryQueryBuilder) tagsRepository() *joinRepository {
	return &joinRepository{
		repository: repository{
			tx:        qb.tx,
			tableName: galleriesTagsTable,
			idColumn:  galleryIDColumn,
		},
		fkColumn: "tag_id",
	}
}

func (qb *galleryQueryBuilder) GetTagIDs(galleryID int) ([]int, error) {
	return qb.tagsRepository().getIDs(galleryID)
}

func (qb *galleryQueryBuilder) UpdateTags(galleryID int, tagIDs []int) error {
	// Delete the existing joins and then create new ones
	return qb.tagsRepository().replace(galleryID, tagIDs)
}

func (qb *galleryQueryBuilder) imagesRepository() *joinRepository {
	return &joinRepository{
		repository: repository{
			tx:        qb.tx,
			tableName: galleriesImagesTable,
			idColumn:  galleryIDColumn,
		},
		fkColumn: "image_id",
	}
}

func (qb *galleryQueryBuilder) GetImageIDs(galleryID int) ([]int, error) {
	return qb.imagesRepository().getIDs(galleryID)
}

func (qb *galleryQueryBuilder) UpdateImages(galleryID int, imageIDs []int) error {
	// Delete the existing joins and then create new ones
	return qb.imagesRepository().replace(galleryID, imageIDs)
}

func (qb *galleryQueryBuilder) scenesRepository() *joinRepository {
	return &joinRepository{
		repository: repository{
			tx:        qb.tx,
			tableName: galleriesScenesTable,
			idColumn:  galleryIDColumn,
		},
		fkColumn: sceneIDColumn,
	}
}

func (qb *galleryQueryBuilder) GetSceneIDs(galleryID int) ([]int, error) {
	return qb.scenesRepository().getIDs(galleryID)
}

func (qb *galleryQueryBuilder) UpdateScenes(galleryID int, sceneIDs []int) error {
	// Delete the existing joins and then create new ones
	return qb.scenesRepository().replace(galleryID, sceneIDs)
}

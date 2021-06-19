package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/stashapp/stash/pkg/models"
)

const imageTable = "images"
const imageIDColumn = "image_id"
const performersImagesTable = "performers_images"
const imagesTagsTable = "images_tags"

var imagesForGalleryQuery = selectAll(imageTable) + `
LEFT JOIN galleries_images as galleries_join on galleries_join.image_id = images.id
WHERE galleries_join.gallery_id = ?
GROUP BY images.id
`

var countImagesForGalleryQuery = `
SELECT gallery_id FROM galleries_images
WHERE gallery_id = ?
GROUP BY image_id
`

type imageQueryBuilder struct {
	repository
}

func NewImageReaderWriter(tx dbi) *imageQueryBuilder {
	return &imageQueryBuilder{
		repository{
			tx:        tx,
			tableName: imageTable,
			idColumn:  idColumn,
		},
	}
}

func (qb *imageQueryBuilder) Create(newObject models.Image) (*models.Image, error) {
	var ret models.Image
	if err := qb.insertObject(newObject, &ret); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (qb *imageQueryBuilder) Update(updatedObject models.ImagePartial) (*models.Image, error) {
	const partial = true
	if err := qb.update(updatedObject.ID, updatedObject, partial); err != nil {
		return nil, err
	}

	return qb.find(updatedObject.ID)
}

func (qb *imageQueryBuilder) UpdateFull(updatedObject models.Image) (*models.Image, error) {
	const partial = false
	if err := qb.update(updatedObject.ID, updatedObject, partial); err != nil {
		return nil, err
	}

	return qb.find(updatedObject.ID)
}

func (qb *imageQueryBuilder) IncrementOCounter(id int) (int, error) {
	_, err := qb.tx.Exec(
		`UPDATE `+imageTable+` SET o_counter = o_counter + 1 WHERE `+imageTable+`.id = ?`,
		id,
	)
	if err != nil {
		return 0, err
	}

	image, err := qb.find(id)
	if err != nil {
		return 0, err
	}

	return image.OCounter, nil
}

func (qb *imageQueryBuilder) DecrementOCounter(id int) (int, error) {
	_, err := qb.tx.Exec(
		`UPDATE `+imageTable+` SET o_counter = o_counter - 1 WHERE `+imageTable+`.id = ? and `+imageTable+`.o_counter > 0`,
		id,
	)
	if err != nil {
		return 0, err
	}

	image, err := qb.find(id)
	if err != nil {
		return 0, err
	}

	return image.OCounter, nil
}

func (qb *imageQueryBuilder) ResetOCounter(id int) (int, error) {
	_, err := qb.tx.Exec(
		`UPDATE `+imageTable+` SET o_counter = 0 WHERE `+imageTable+`.id = ?`,
		id,
	)
	if err != nil {
		return 0, err
	}

	image, err := qb.find(id)
	if err != nil {
		return 0, err
	}

	return image.OCounter, nil
}

func (qb *imageQueryBuilder) Destroy(id int) error {
	return qb.destroyExisting([]int{id})
}

func (qb *imageQueryBuilder) Find(id int) (*models.Image, error) {
	return qb.find(id)
}

func (qb *imageQueryBuilder) FindMany(ids []int) ([]*models.Image, error) {
	var images []*models.Image
	for _, id := range ids {
		image, err := qb.Find(id)
		if err != nil {
			return nil, err
		}

		if image == nil {
			return nil, fmt.Errorf("image with id %d not found", id)
		}

		images = append(images, image)
	}

	return images, nil
}

func (qb *imageQueryBuilder) find(id int) (*models.Image, error) {
	var ret models.Image
	if err := qb.get(id, &ret); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ret, nil
}

func (qb *imageQueryBuilder) FindByChecksum(checksum string) (*models.Image, error) {
	query := "SELECT * FROM images WHERE checksum = ? LIMIT 1"
	args := []interface{}{checksum}
	return qb.queryImage(query, args)
}

func (qb *imageQueryBuilder) FindByPath(path string) (*models.Image, error) {
	query := selectAll(imageTable) + "WHERE path = ? LIMIT 1"
	args := []interface{}{path}
	return qb.queryImage(query, args)
}

func (qb *imageQueryBuilder) FindByGalleryID(galleryID int) ([]*models.Image, error) {
	args := []interface{}{galleryID}
	return qb.queryImages(imagesForGalleryQuery+qb.getImageSort(nil), args)
}

func (qb *imageQueryBuilder) CountByGalleryID(galleryID int) (int, error) {
	args := []interface{}{galleryID}
	return qb.runCountQuery(qb.buildCountQuery(countImagesForGalleryQuery), args)
}

func (qb *imageQueryBuilder) Count() (int, error) {
	return qb.runCountQuery(qb.buildCountQuery("SELECT images.id FROM images"), nil)
}

func (qb *imageQueryBuilder) Size() (float64, error) {
	return qb.runSumQuery("SELECT SUM(cast(size as double)) as sum FROM images", nil)
}

func (qb *imageQueryBuilder) All() ([]*models.Image, error) {
	return qb.queryImages(selectAll(imageTable)+qb.getImageSort(nil), nil)
}

func (qb *imageQueryBuilder) validateFilter(imageFilter *models.ImageFilterType) error {
	const and = "AND"
	const or = "OR"
	const not = "NOT"

	if imageFilter.And != nil {
		if imageFilter.Or != nil {
			return illegalFilterCombination(and, or)
		}
		if imageFilter.Not != nil {
			return illegalFilterCombination(and, not)
		}

		return qb.validateFilter(imageFilter.And)
	}

	if imageFilter.Or != nil {
		if imageFilter.Not != nil {
			return illegalFilterCombination(or, not)
		}

		return qb.validateFilter(imageFilter.Or)
	}

	if imageFilter.Not != nil {
		return qb.validateFilter(imageFilter.Not)
	}

	return nil
}

func (qb *imageQueryBuilder) makeFilter(imageFilter *models.ImageFilterType) *filterBuilder {
	query := &filterBuilder{}

	if imageFilter.And != nil {
		query.and(qb.makeFilter(imageFilter.And))
	}
	if imageFilter.Or != nil {
		query.or(qb.makeFilter(imageFilter.Or))
	}
	if imageFilter.Not != nil {
		query.not(qb.makeFilter(imageFilter.Not))
	}

	query.handleCriterion(stringCriterionHandler(imageFilter.Checksum, "images.checksum"))
	query.handleCriterion(stringCriterionHandler(imageFilter.Title, "images.title"))
	query.handleCriterion(stringCriterionHandler(imageFilter.Path, "images.path"))
	query.handleCriterion(intCriterionHandler(imageFilter.Rating, "images.rating"))
	query.handleCriterion(intCriterionHandler(imageFilter.OCounter, "images.o_counter"))
	query.handleCriterion(boolCriterionHandler(imageFilter.Organized, "images.organized"))
	query.handleCriterion(resolutionCriterionHandler(imageFilter.Resolution, "images.height", "images.width"))
	query.handleCriterion(imageIsMissingCriterionHandler(qb, imageFilter.IsMissing))

	query.handleCriterion(imageTagsCriterionHandler(qb, imageFilter.Tags))
	query.handleCriterion(imageTagCountCriterionHandler(qb, imageFilter.TagCount))
	query.handleCriterion(imageGalleriesCriterionHandler(qb, imageFilter.Galleries))
	query.handleCriterion(imagePerformersCriterionHandler(qb, imageFilter.Performers))
	query.handleCriterion(imagePerformerCountCriterionHandler(qb, imageFilter.PerformerCount))
	query.handleCriterion(imageStudioCriterionHandler(qb, imageFilter.Studios))
	query.handleCriterion(imagePerformerTagsCriterionHandler(qb, imageFilter.PerformerTags))

	return query
}

func (qb *imageQueryBuilder) makeQuery(imageFilter *models.ImageFilterType, findFilter *models.FindFilterType) (*queryBuilder, error) {
	if imageFilter == nil {
		imageFilter = &models.ImageFilterType{}
	}
	if findFilter == nil {
		findFilter = &models.FindFilterType{}
	}

	query := qb.newQuery()

	query.body = selectDistinctIDs(imageTable)

	if q := findFilter.Q; q != nil && *q != "" {
		searchColumns := []string{"images.title", "images.path", "images.checksum"}
		clause, thisArgs := getSearchBinding(searchColumns, *q, false)
		query.addWhere(clause)
		query.addArg(thisArgs...)
	}

	if err := qb.validateFilter(imageFilter); err != nil {
		return nil, err
	}
	filter := qb.makeFilter(imageFilter)

	query.addFilter(filter)

	query.sortAndPagination = qb.getImageSort(findFilter) + getPagination(findFilter)

	return &query, nil
}

func (qb *imageQueryBuilder) Query(imageFilter *models.ImageFilterType, findFilter *models.FindFilterType) ([]*models.Image, int, error) {
	query, err := qb.makeQuery(imageFilter, findFilter)
	if err != nil {
		return nil, 0, err
	}

	idsResult, countResult, err := query.executeFind()
	if err != nil {
		return nil, 0, err
	}

	var images []*models.Image
	for _, id := range idsResult {
		image, err := qb.Find(id)
		if err != nil {
			return nil, 0, err
		}

		images = append(images, image)
	}

	return images, countResult, nil
}

func (qb *imageQueryBuilder) QueryCount(imageFilter *models.ImageFilterType, findFilter *models.FindFilterType) (int, error) {
	query, err := qb.makeQuery(imageFilter, findFilter)
	if err != nil {
		return 0, err
	}

	return query.executeCount()
}

func imageIsMissingCriterionHandler(qb *imageQueryBuilder, isMissing *string) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if isMissing != nil && *isMissing != "" {
			switch *isMissing {
			case "studio":
				f.addWhere("images.studio_id IS NULL")
			case "performers":
				qb.performersRepository().join(f, "performers_join", "images.id")
				f.addWhere("performers_join.image_id IS NULL")
			case "galleries":
				qb.galleriesRepository().join(f, "galleries_join", "images.id")
				f.addWhere("galleries_join.image_id IS NULL")
			case "tags":
				qb.tagsRepository().join(f, "tags_join", "images.id")
				f.addWhere("tags_join.image_id IS NULL")
			default:
				f.addWhere("(images." + *isMissing + " IS NULL OR TRIM(images." + *isMissing + ") = '')")
			}
		}
	}
}

func (qb *imageQueryBuilder) getMultiCriterionHandlerBuilder(foreignTable, joinTable, foreignFK string, addJoinsFunc func(f *filterBuilder)) multiCriterionHandlerBuilder {
	return multiCriterionHandlerBuilder{
		primaryTable: imageTable,
		foreignTable: foreignTable,
		joinTable:    joinTable,
		primaryFK:    imageIDColumn,
		foreignFK:    foreignFK,
		addJoinsFunc: addJoinsFunc,
	}
}

func imageTagsCriterionHandler(qb *imageQueryBuilder, tags *models.HierarchicalMultiCriterionInput) criterionHandlerFunc {
	h := joinedHierarchicalMultiCriterionHandlerBuilder{
		primaryTable: imageTable,
		foreignTable: tagTable,
		foreignFK:    "tag_id",
		derivedTable: "tag",

		relationsTable: "tags_relations",
		joinAs:         "image_tag",
		joinTable:      imagesTagsTable,
		primaryFK:      imageIDColumn,
	}

	return h.handler(tags)
}

func imageTagCountCriterionHandler(qb *imageQueryBuilder, tagCount *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: imageTable,
		joinTable:    imagesTagsTable,
		primaryFK:    imageIDColumn,
	}

	return h.handler(tagCount)
}

func imageGalleriesCriterionHandler(qb *imageQueryBuilder, galleries *models.MultiCriterionInput) criterionHandlerFunc {
	addJoinsFunc := func(f *filterBuilder) {
		qb.galleriesRepository().join(f, "galleries_join", "images.id")
		f.addJoin(galleryTable, "", "galleries_join.gallery_id = galleries.id")
	}
	h := qb.getMultiCriterionHandlerBuilder(galleryTable, galleriesImagesTable, galleryIDColumn, addJoinsFunc)

	return h.handler(galleries)
}

func imagePerformersCriterionHandler(qb *imageQueryBuilder, performers *models.MultiCriterionInput) criterionHandlerFunc {
	h := joinedMultiCriterionHandlerBuilder{
		primaryTable: imageTable,
		joinTable:    performersImagesTable,
		joinAs:       "performers_join",
		primaryFK:    imageIDColumn,
		foreignFK:    performerIDColumn,

		addJoinTable: func(f *filterBuilder) {
			qb.performersRepository().join(f, "performers_join", "images.id")
		},
	}

	return h.handler(performers)
}

func imagePerformerCountCriterionHandler(qb *imageQueryBuilder, performerCount *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: imageTable,
		joinTable:    performersImagesTable,
		primaryFK:    imageIDColumn,
	}

	return h.handler(performerCount)
}

func imageStudioCriterionHandler(qb *imageQueryBuilder, studios *models.HierarchicalMultiCriterionInput) criterionHandlerFunc {
	h := hierarchicalMultiCriterionHandlerBuilder{
		primaryTable: imageTable,
		foreignTable: studioTable,
		foreignFK:    studioIDColumn,
		derivedTable: "studio",
		parentFK:     "parent_id",
	}

	return h.handler(studios)
}

func imagePerformerTagsCriterionHandler(qb *imageQueryBuilder, tags *models.HierarchicalMultiCriterionInput) criterionHandlerFunc {
	return func(f *filterBuilder) {
		if tags != nil && len(tags.Value) > 0 {
			addHierarchicalWithClause(f, tags.Value, "tags_for_performer", tagTable, "tags_relations", "", tags.Depth)

			f.addWith(`performer_tags AS (
SELECT pi.image_id, t.id AS root_tag_id FROM performers_images pi
INNER JOIN performers_tags pt ON pt.performer_id = pi.performer_id
INNER JOIN tags_for_performer t ON t.child_id = pt.tag_id
)`)

			f.addJoin("performer_tags", "", "performer_tags.image_id = images.id")

			addHierarchicalConditionClauses(f, tags, "performer_tags", "root_tag_id")
		}
	}
}

func (qb *imageQueryBuilder) getImageSort(findFilter *models.FindFilterType) string {
	if findFilter == nil {
		return " ORDER BY images.path ASC "
	}
	sort := findFilter.GetSort("title")
	direction := findFilter.GetDirection()

	switch sort {
	case "tag_count":
		return getCountSort(imageTable, imagesTagsTable, imageIDColumn, direction)
	case "performer_count":
		return getCountSort(imageTable, performersImagesTable, imageIDColumn, direction)
	default:
		return getSort(sort, direction, "images")
	}
}

func (qb *imageQueryBuilder) queryImage(query string, args []interface{}) (*models.Image, error) {
	results, err := qb.queryImages(query, args)
	if err != nil || len(results) < 1 {
		return nil, err
	}
	return results[0], nil
}

func (qb *imageQueryBuilder) queryImages(query string, args []interface{}) ([]*models.Image, error) {
	var ret models.Images
	if err := qb.query(query, args, &ret); err != nil {
		return nil, err
	}

	return []*models.Image(ret), nil
}

func (qb *imageQueryBuilder) galleriesRepository() *joinRepository {
	return &joinRepository{
		repository: repository{
			tx:        qb.tx,
			tableName: galleriesImagesTable,
			idColumn:  imageIDColumn,
		},
		fkColumn: galleryIDColumn,
	}
}

func (qb *imageQueryBuilder) GetGalleryIDs(imageID int) ([]int, error) {
	return qb.galleriesRepository().getIDs(imageID)
}

func (qb *imageQueryBuilder) UpdateGalleries(imageID int, galleryIDs []int) error {
	// Delete the existing joins and then create new ones
	return qb.galleriesRepository().replace(imageID, galleryIDs)
}

func (qb *imageQueryBuilder) performersRepository() *joinRepository {
	return &joinRepository{
		repository: repository{
			tx:        qb.tx,
			tableName: performersImagesTable,
			idColumn:  imageIDColumn,
		},
		fkColumn: performerIDColumn,
	}
}

func (qb *imageQueryBuilder) GetPerformerIDs(imageID int) ([]int, error) {
	return qb.performersRepository().getIDs(imageID)
}

func (qb *imageQueryBuilder) UpdatePerformers(imageID int, performerIDs []int) error {
	// Delete the existing joins and then create new ones
	return qb.performersRepository().replace(imageID, performerIDs)
}

func (qb *imageQueryBuilder) tagsRepository() *joinRepository {
	return &joinRepository{
		repository: repository{
			tx:        qb.tx,
			tableName: imagesTagsTable,
			idColumn:  imageIDColumn,
		},
		fkColumn: tagIDColumn,
	}
}

func (qb *imageQueryBuilder) GetTagIDs(imageID int) ([]int, error) {
	return qb.tagsRepository().getIDs(imageID)
}

func (qb *imageQueryBuilder) UpdateTags(imageID int, tagIDs []int) error {
	// Delete the existing joins and then create new ones
	return qb.tagsRepository().replace(imageID, tagIDs)
}

package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/stashapp/stash/pkg/models"
)

const studioTable = "studios"
const studioIDColumn = "studio_id"

type studioQueryBuilder struct {
	repository
}

func NewStudioReaderWriter(tx dbi) *studioQueryBuilder {
	return &studioQueryBuilder{
		repository{
			tx:        tx,
			tableName: studioTable,
			idColumn:  idColumn,
		},
	}
}

func (qb *studioQueryBuilder) Create(newObject models.Studio) (*models.Studio, error) {
	var ret models.Studio
	if err := qb.insertObject(newObject, &ret); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (qb *studioQueryBuilder) Update(updatedObject models.StudioPartial) (*models.Studio, error) {
	const partial = true
	if err := qb.update(updatedObject.ID, updatedObject, partial); err != nil {
		return nil, err
	}

	return qb.Find(updatedObject.ID)
}

func (qb *studioQueryBuilder) UpdateFull(updatedObject models.Studio) (*models.Studio, error) {
	const partial = false
	if err := qb.update(updatedObject.ID, updatedObject, partial); err != nil {
		return nil, err
	}

	return qb.Find(updatedObject.ID)
}

func (qb *studioQueryBuilder) Destroy(id int) error {
	// TODO - set null on foreign key in scraped items
	// remove studio from scraped items
	_, err := qb.tx.Exec("UPDATE scraped_items SET studio_id = null WHERE studio_id = ?", id)
	if err != nil {
		return err
	}

	return qb.destroyExisting([]int{id})
}

func (qb *studioQueryBuilder) Find(id int) (*models.Studio, error) {
	var ret models.Studio
	if err := qb.get(id, &ret); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ret, nil
}

func (qb *studioQueryBuilder) FindMany(ids []int) ([]*models.Studio, error) {
	var studios []*models.Studio
	for _, id := range ids {
		studio, err := qb.Find(id)
		if err != nil {
			return nil, err
		}

		if studio == nil {
			return nil, fmt.Errorf("studio with id %d not found", id)
		}

		studios = append(studios, studio)
	}

	return studios, nil
}

func (qb *studioQueryBuilder) FindChildren(id int) ([]*models.Studio, error) {
	query := "SELECT studios.* FROM studios WHERE studios.parent_id = ?"
	args := []interface{}{id}
	return qb.queryStudios(query, args)
}

func (qb *studioQueryBuilder) FindBySceneID(sceneID int) (*models.Studio, error) {
	query := "SELECT studios.* FROM studios JOIN scenes ON studios.id = scenes.studio_id WHERE scenes.id = ? LIMIT 1"
	args := []interface{}{sceneID}
	return qb.queryStudio(query, args)
}

func (qb *studioQueryBuilder) FindByName(name string, nocase bool) (*models.Studio, error) {
	query := "SELECT * FROM studios WHERE name = ?"
	if nocase {
		query += " COLLATE NOCASE"
	}
	query += " LIMIT 1"
	args := []interface{}{name}
	return qb.queryStudio(query, args)
}

func (qb *studioQueryBuilder) Count() (int, error) {
	return qb.runCountQuery(qb.buildCountQuery("SELECT studios.id FROM studios"), nil)
}

func (qb *studioQueryBuilder) All() ([]*models.Studio, error) {
	return qb.queryStudios(selectAll("studios")+qb.getStudioSort(nil), nil)
}

func (qb *studioQueryBuilder) QueryForAutoTag(words []string) ([]*models.Studio, error) {
	// TODO - Query needs to be changed to support queries of this type, and
	// this method should be removed
	query := selectAll(studioTable)

	var whereClauses []string
	var args []interface{}

	for _, w := range words {
		whereClauses = append(whereClauses, "name like ?")
		args = append(args, w+"%")
	}

	where := strings.Join(whereClauses, " OR ")
	return qb.queryStudios(query+" WHERE "+where, args)
}

func (qb *studioQueryBuilder) Query(studioFilter *models.StudioFilterType, findFilter *models.FindFilterType) ([]*models.Studio, int, error) {
	if studioFilter == nil {
		studioFilter = &models.StudioFilterType{}
	}
	if findFilter == nil {
		findFilter = &models.FindFilterType{}
	}

	query := qb.newQuery()

	query.body = selectDistinctIDs("studios")
	query.body += `
		left join scenes on studios.id = scenes.studio_id
		left join studio_stash_ids on studio_stash_ids.studio_id = studios.id
	`

	if q := findFilter.Q; q != nil && *q != "" {
		searchColumns := []string{"studios.name"}

		clause, thisArgs := getSearchBinding(searchColumns, *q, false)
		query.addWhere(clause)
		query.addArg(thisArgs...)
	}

	if parentsFilter := studioFilter.Parents; parentsFilter != nil && len(parentsFilter.Value) > 0 {
		query.body += `
			left join studios as parent_studio on parent_studio.id = studios.parent_id
		`

		for _, studioID := range parentsFilter.Value {
			query.addArg(studioID)
		}

		whereClause, havingClause := getMultiCriterionClause("studios", "parent_studio", "", "", "parent_id", parentsFilter)

		query.addWhere(whereClause)
		query.addHaving(havingClause)
	}

	if rating := studioFilter.Rating; rating != nil {
		query.handleIntCriterionInput(studioFilter.Rating, "studios.rating")
	}
	query.handleCountCriterion(studioFilter.SceneCount, studioTable, sceneTable, studioIDColumn)
	query.handleCountCriterion(studioFilter.ImageCount, studioTable, imageTable, studioIDColumn)
	query.handleCountCriterion(studioFilter.GalleryCount, studioTable, galleryTable, studioIDColumn)
	query.handleStringCriterionInput(studioFilter.Name, "studios.name")
	query.handleStringCriterionInput(studioFilter.Details, "studios.details")
	query.handleStringCriterionInput(studioFilter.URL, "studios.url")
	query.handleStringCriterionInput(studioFilter.StashID, "studio_stash_ids.stash_id")

	if isMissingFilter := studioFilter.IsMissing; isMissingFilter != nil && *isMissingFilter != "" {
		switch *isMissingFilter {
		case "image":
			query.body += `left join studios_image on studios_image.studio_id = studios.id
			`
			query.addWhere("studios_image.studio_id IS NULL")
		case "stash_id":
			query.addWhere("studio_stash_ids.studio_id IS NULL")
		default:
			query.addWhere("studios." + *isMissingFilter + " IS NULL")
		}
	}

	query.sortAndPagination = qb.getStudioSort(findFilter) + getPagination(findFilter)
	idsResult, countResult, err := query.executeFind()
	if err != nil {
		return nil, 0, err
	}

	var studios []*models.Studio
	for _, id := range idsResult {
		studio, err := qb.Find(id)
		if err != nil {
			return nil, 0, err
		}

		studios = append(studios, studio)
	}

	return studios, countResult, nil
}

func (qb *studioQueryBuilder) getStudioSort(findFilter *models.FindFilterType) string {
	var sort string
	var direction string
	if findFilter == nil {
		sort = "name"
		direction = "ASC"
	} else {
		sort = findFilter.GetSort("name")
		direction = findFilter.GetDirection()
	}

	switch sort {
	case "images_count":
		return getCountSort(studioTable, imageTable, studioIDColumn, direction)
	case "galleries_count":
		return getCountSort(studioTable, galleryTable, studioIDColumn, direction)
	default:
		return getSort(sort, direction, "studios")
	}
}

func (qb *studioQueryBuilder) queryStudio(query string, args []interface{}) (*models.Studio, error) {
	results, err := qb.queryStudios(query, args)
	if err != nil || len(results) < 1 {
		return nil, err
	}
	return results[0], nil
}

func (qb *studioQueryBuilder) queryStudios(query string, args []interface{}) ([]*models.Studio, error) {
	var ret models.Studios
	if err := qb.query(query, args, &ret); err != nil {
		return nil, err
	}

	return []*models.Studio(ret), nil
}

func (qb *studioQueryBuilder) imageRepository() *imageRepository {
	return &imageRepository{
		repository: repository{
			tx:        qb.tx,
			tableName: "studios_image",
			idColumn:  studioIDColumn,
		},
		imageColumn: "image",
	}
}

func (qb *studioQueryBuilder) GetImage(studioID int) ([]byte, error) {
	return qb.imageRepository().get(studioID)
}

func (qb *studioQueryBuilder) HasImage(studioID int) (bool, error) {
	return qb.imageRepository().exists(studioID)
}

func (qb *studioQueryBuilder) UpdateImage(studioID int, image []byte) error {
	return qb.imageRepository().replace(studioID, image)
}

func (qb *studioQueryBuilder) DestroyImage(studioID int) error {
	return qb.imageRepository().destroy([]int{studioID})
}

func (qb *studioQueryBuilder) stashIDRepository() *stashIDRepository {
	return &stashIDRepository{
		repository{
			tx:        qb.tx,
			tableName: "studio_stash_ids",
			idColumn:  studioIDColumn,
		},
	}
}

func (qb *studioQueryBuilder) GetStashIDs(studioID int) ([]*models.StashID, error) {
	return qb.stashIDRepository().get(studioID)
}

func (qb *studioQueryBuilder) UpdateStashIDs(studioID int, stashIDs []models.StashID) error {
	return qb.stashIDRepository().replace(studioID, stashIDs)
}

func (qb *studioQueryBuilder) FindByStashID(stashID string, stashboxEndpoint string) ([]*models.Studio, error) {
	query := selectAll("studios") + `
		JOIN studio_stash_ids as stashid_join on stashid_join.studio_id = studios.id
		WHERE stashid_join.stash_id = ?
		AND stashid_join.endpoint = ?
	`
	args := []interface{}{stashID, stashboxEndpoint}
	return qb.queryStudios(query, args)
}

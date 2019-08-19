package models

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/stashapp/stash/pkg/database"
)

type PerformerQueryBuilder struct{}

func NewPerformerQueryBuilder() PerformerQueryBuilder {
	return PerformerQueryBuilder{}
}

func (qb *PerformerQueryBuilder) Create(newPerformer Performer, tx *sqlx.Tx) (*Performer, error) {
	ensureTx(tx)
	result, err := tx.NamedExec(
		`INSERT INTO performers (image, checksum, name, url, twitter, instagram, birthdate, ethnicity, country,
                        				eye_color, height, measurements, fake_tits, career_length, tattoos, piercings,
                        				aliases, favorite, created_at, updated_at)
				VALUES (:image, :checksum, :name, :url, :twitter, :instagram, :birthdate, :ethnicity, :country,
                        :eye_color, :height, :measurements, :fake_tits, :career_length, :tattoos, :piercings,
                        :aliases, :favorite, :created_at, :updated_at)
		`,
		newPerformer,
	)
	if err != nil {
		return nil, err
	}
	performerID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	if err := tx.Get(&newPerformer, `SELECT * FROM performers WHERE id = ? LIMIT 1`, performerID); err != nil {
		return nil, err
	}
	return &newPerformer, nil
}

func (qb *PerformerQueryBuilder) Update(updatedPerformer Performer, tx *sqlx.Tx) (*Performer, error) {
	ensureTx(tx)
	_, err := tx.NamedExec(
		`UPDATE performers SET `+SQLGenKeys(updatedPerformer)+` WHERE performers.id = :id`,
		updatedPerformer,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Get(&updatedPerformer, `SELECT * FROM performers WHERE id = ? LIMIT 1`, updatedPerformer.ID); err != nil {
		return nil, err
	}
	return &updatedPerformer, nil
}

func (qb *PerformerQueryBuilder) Destroy(id string, tx *sqlx.Tx) error {
	_, err := tx.Exec("DELETE FROM performers_scenes WHERE performer_id = ?", id)
	if err != nil {
		return err
	}

	return executeDeleteQuery("performers", id, tx)
}

func (qb *PerformerQueryBuilder) Find(id int) (*Performer, error) {
	query := "SELECT * FROM performers WHERE id = ? LIMIT 1"
	args := []interface{}{id}
	results, err := qb.queryPerformers(query, args, nil)
	if err != nil || len(results) < 1 {
		return nil, err
	}
	return results[0], nil
}

func (qb *PerformerQueryBuilder) FindBySceneID(sceneID int, tx *sqlx.Tx) ([]*Performer, error) {
	query := `
		SELECT performers.* FROM performers
		LEFT JOIN performers_scenes as scenes_join on scenes_join.performer_id = performers.id
		LEFT JOIN scenes on scenes_join.scene_id = scenes.id
		WHERE scenes.id = ?
		GROUP BY performers.id
	`
	args := []interface{}{sceneID}
	return qb.queryPerformers(query, args, tx)
}

func (qb *PerformerQueryBuilder) FindByNames(names []string, tx *sqlx.Tx) ([]*Performer, error) {
	query := "SELECT * FROM performers WHERE name IN " + getInBinding(len(names))
	var args []interface{}
	for _, name := range names {
		args = append(args, name)
	}
	return qb.queryPerformers(query, args, tx)
}

func (qb *PerformerQueryBuilder) Count() (int, error) {
	return runCountQuery(buildCountQuery("SELECT performers.id FROM performers"), nil)
}

func (qb *PerformerQueryBuilder) All() ([]*Performer, error) {
	return qb.queryPerformers(selectAll("performers")+qb.getPerformerSort(nil), nil, nil)
}

func (qb *PerformerQueryBuilder) Query(performerFilter *PerformerFilterType, findFilter *FindFilterType) ([]*Performer, int) {
	if performerFilter == nil {
		performerFilter = &PerformerFilterType{}
	}
	if findFilter == nil {
		findFilter = &FindFilterType{}
	}

	var whereClauses []string
	var havingClauses []string
	var args []interface{}
	body := selectDistinctIDs("performers")
	body += `
		left join performers_scenes as scenes_join on scenes_join.performer_id = performers.id
		left join scenes on scenes_join.scene_id = scenes.id
	`

	if q := findFilter.Q; q != nil && *q != "" {
		searchColumns := []string{"performers.name", "performers.checksum", "performers.birthdate", "performers.ethnicity"}
		whereClauses = append(whereClauses, getSearch(searchColumns, *q))
	}

	if favoritesFilter := performerFilter.FilterFavorites; favoritesFilter != nil {
		if *favoritesFilter == true {
			whereClauses = append(whereClauses, "performers.favorite = 1")
		} else {
			whereClauses = append(whereClauses, "performers.favorite = 0")
		}
	}

	sortAndPagination := qb.getPerformerSort(findFilter) + getPagination(findFilter)
	idsResult, countResult := executeFindQuery("performers", body, args, sortAndPagination, whereClauses, havingClauses)

	var performers []*Performer
	for _, id := range idsResult {
		performer, _ := qb.Find(id)
		performers = append(performers, performer)
	}

	return performers, countResult
}

func (qb *PerformerQueryBuilder) getPerformerSort(findFilter *FindFilterType) string {
	var sort string
	var direction string
	if findFilter == nil {
		sort = "name"
		direction = "ASC"
	} else {
		sort = findFilter.GetSort("name")
		direction = findFilter.GetDirection()
	}
	return getSort(sort, direction, "performers")
}

func (qb *PerformerQueryBuilder) queryPerformers(query string, args []interface{}, tx *sqlx.Tx) ([]*Performer, error) {
	var rows *sqlx.Rows
	var err error
	if tx != nil {
		rows, err = tx.Queryx(query, args...)
	} else {
		rows, err = database.DB.Queryx(query, args...)
	}

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	defer rows.Close()

	performers := make([]*Performer, 0)
	for rows.Next() {
		performer := Performer{}
		if err := rows.StructScan(&performer); err != nil {
			return nil, err
		}
		performers = append(performers, &performer)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return performers, nil
}

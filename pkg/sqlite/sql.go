package sqlite

import (
	"database/sql"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
)

var randomSortFloat = rand.Float64()

func selectAll(tableName string) string {
	idColumn := getColumn(tableName, "*")
	return "SELECT " + idColumn + " FROM " + tableName + " "
}

func selectDistinctIDs(tableName string) string {
	idColumn := getColumn(tableName, "id")
	return "SELECT DISTINCT " + idColumn + " FROM " + tableName + " "
}

func getColumn(tableName string, columnName string) string {
	return tableName + "." + columnName
}

func getPagination(findFilter *models.FindFilterType) string {
	if findFilter == nil {
		panic("nil find filter for pagination")
	}

	if findFilter.IsGetAll() {
		return " "
	}

	return getPaginationSQL(findFilter.GetPage(), findFilter.GetPageSize())
}

func getPaginationSQL(page int, perPage int) string {
	page = (page - 1) * perPage
	return " LIMIT " + strconv.Itoa(perPage) + " OFFSET " + strconv.Itoa(page) + " "
}

func getSortDirection(direction string) string {
	if direction != "ASC" && direction != "DESC" {
		return "ASC"
	} else {
		return direction
	}
}
func getSort(sort string, direction string, tableName string) string {
	direction = getSortDirection(direction)

	const randomSeedPrefix = "random_"

	if strings.HasSuffix(sort, "_count") {
		var relationTableName = strings.TrimSuffix(sort, "_count") // TODO: pluralize?
		colName := getColumn(relationTableName, "id")
		return " ORDER BY COUNT(distinct " + colName + ") " + direction
	} else if strings.Compare(sort, "filesize") == 0 {
		colName := getColumn(tableName, "size")
		return " ORDER BY cast(" + colName + " as integer) " + direction
	} else if strings.HasPrefix(sort, randomSeedPrefix) {
		// seed as a parameter from the UI
		// turn the provided seed into a float
		seedStr := "0." + sort[len(randomSeedPrefix):]
		seed, err := strconv.ParseFloat(seedStr, 32)
		if err != nil {
			// fallback to default seed
			seed = randomSortFloat
		}
		return getRandomSort(tableName, direction, seed)
	} else if strings.Compare(sort, "random") == 0 {
		return getRandomSort(tableName, direction, randomSortFloat)
	} else {
		colName := getColumn(tableName, sort)
		var additional string
		if tableName == "scenes" {
			additional = ", bitrate DESC, framerate DESC, scenes.rating DESC, scenes.duration DESC"
		} else if tableName == "scene_markers" {
			additional = ", scene_markers.scene_id ASC, scene_markers.seconds ASC"
		}
		if strings.Compare(sort, "name") == 0 {
			return " ORDER BY " + colName + " COLLATE NOCASE " + direction + additional
		}
		if strings.Compare(sort, "title") == 0 {
			return " ORDER BY " + colName + " COLLATE NATURAL_CS " + direction + additional
		}

		return " ORDER BY " + colName + " " + direction + additional
	}
}

func getRandomSort(tableName string, direction string, seed float64) string {
	// https://stackoverflow.com/a/24511461
	colName := getColumn(tableName, "id")
	randomSortString := strconv.FormatFloat(seed, 'f', 16, 32)
	return " ORDER BY " + "(substr(" + colName + " * " + randomSortString + ", length(" + colName + ") + 2))" + " " + direction
}

func getCountSort(primaryTable, joinTable, primaryFK, direction string) string {
	return fmt.Sprintf(" ORDER BY (SELECT COUNT(*) FROM %s WHERE %s = %s.id) %s", joinTable, primaryFK, primaryTable, getSortDirection(direction))
}

func getSearchBinding(columns []string, q string, not bool) (string, []interface{}) {
	var likeClauses []string
	var args []interface{}

	notStr := ""
	binaryType := " OR "
	if not {
		notStr = " NOT"
		binaryType = " AND "
	}
	q = strings.TrimSpace(q)
	trimmedQuery := strings.Trim(q, "\"")

	if trimmedQuery == q {
		q = regexp.MustCompile(`\s+`).ReplaceAllString(q, " ")
		queryWords := strings.Split(q, " ")
		// Search for any word
		for _, word := range queryWords {
			for _, column := range columns {
				likeClauses = append(likeClauses, column+notStr+" LIKE ?")
				args = append(args, "%"+word+"%")
			}
		}
	} else {
		// Search the exact query
		for _, column := range columns {
			likeClauses = append(likeClauses, column+notStr+" LIKE ?")
			args = append(args, "%"+trimmedQuery+"%")
		}
	}
	likes := strings.Join(likeClauses, binaryType)

	return "(" + likes + ")", args
}

func getInBinding(length int) string {
	bindings := strings.Repeat("?, ", length)
	bindings = strings.TrimRight(bindings, ", ")
	return "(" + bindings + ")"
}

func getCriterionModifierBinding(criterionModifier models.CriterionModifier, value interface{}) (string, int) {
	var length int
	switch x := value.(type) {
	case []string:
		length = len(x)
	case []int:
		length = len(x)
	default:
		length = 1
	}
	if modifier := criterionModifier.String(); criterionModifier.IsValid() {
		switch modifier {
		case "EQUALS", "NOT_EQUALS", "GREATER_THAN", "LESS_THAN", "IS_NULL", "NOT_NULL", "BETWEEN", "NOT_BETWEEN":
			return getSimpleCriterionClause(criterionModifier, "?")
		case "INCLUDES":
			return "IN " + getInBinding(length), length // TODO?
		case "EXCLUDES":
			return "NOT IN " + getInBinding(length), length // TODO?
		default:
			logger.Errorf("todo")
			return "= ?", 1 // TODO
		}
	}
	return "= ?", 1 // TODO
}

func getSimpleCriterionClause(criterionModifier models.CriterionModifier, rhs string) (string, int) {
	if modifier := criterionModifier.String(); criterionModifier.IsValid() {
		switch modifier {
		case "EQUALS":
			return "= " + rhs, 1
		case "NOT_EQUALS":
			return "!= " + rhs, 1
		case "GREATER_THAN":
			return "> " + rhs, 1
		case "LESS_THAN":
			return "< " + rhs, 1
		case "IS_NULL":
			return "IS NULL", 0
		case "NOT_NULL":
			return "IS NOT NULL", 0
		case "BETWEEN":
			// Make BETWEEN behave exclusive, same as there are also exclusive greater / less than modifiers
			return "BETWEEN (" + rhs + " + 1) AND (" + rhs + " - 1)", 2
		case "NOT_BETWEEN":
			return "NOT BETWEEN (" + rhs + " + 1) AND (" + rhs + " - 1)", 2
		default:
			logger.Errorf("todo")
			return "= ?", 1 // TODO
		}
	}

	return "= ?", 1 // TODO
}

func getIntCriterionWhereClause(column string, input models.IntCriterionInput) (string, []interface{}) {
	binding, _ := getSimpleCriterionClause(input.Modifier, "?")
	var args []interface{}

	switch input.Modifier {
	case "EQUALS", "NOT_EQUALS":
		args = []interface{}{input.Exact}
		break
	case "LESS_THAN":
		args = []interface{}{input.Upper}
		break
	case "GREATER_THAN":
		args = []interface{}{input.Lower}
		break
	case "BETWEEN", "NOT_BETWEEN":
		args = []interface{}{input.Lower, input.Upper}
		break
	}

	return column + " " + binding, args
}

// returns where clause and having clause
func getMultiCriterionClause(primaryTable, foreignTable, joinTable, primaryFK, foreignFK string, criterion *models.MultiCriterionInput) (string, string) {
	whereClause := ""
	havingClause := ""
	if criterion.Modifier == models.CriterionModifierIncludes {
		// includes any of the provided ids
		whereClause = foreignTable + ".id IN " + getInBinding(len(criterion.Value))
	} else if criterion.Modifier == models.CriterionModifierIncludesAll {
		// includes all of the provided ids
		whereClause = foreignTable + ".id IN " + getInBinding(len(criterion.Value))
		havingClause = "count(distinct " + foreignTable + ".id) IS " + strconv.Itoa(len(criterion.Value))
	} else if criterion.Modifier == models.CriterionModifierExcludes {
		// excludes all of the provided ids
		if joinTable != "" {
			whereClause = "not exists (select " + joinTable + "." + primaryFK + " from " + joinTable + " where " + joinTable + "." + primaryFK + " = " + primaryTable + ".id and " + joinTable + "." + foreignFK + " in " + getInBinding(len(criterion.Value)) + ")"
		} else {
			whereClause = "not exists (select s.id from " + primaryTable + " as s where s.id = " + primaryTable + ".id and s." + foreignFK + " in " + getInBinding(len(criterion.Value)) + ")"
		}
	}

	return whereClause, havingClause
}

func getCountCriterionClause(primaryTable, joinTable, primaryFK string, criterion models.IntCriterionInput) (string, []interface{}) {
	lhs := fmt.Sprintf("(SELECT COUNT(*) FROM %s s WHERE s.%s = %s.id)", joinTable, primaryFK, primaryTable)
	return getIntCriterionWhereClause(lhs, criterion)
}

func ensureTx(tx *sqlx.Tx) {
	if tx == nil {
		panic("must use a transaction")
	}
}

func getImage(tx dbi, query string, args ...interface{}) ([]byte, error) {
	rows, err := tx.Queryx(query, args...)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	defer rows.Close()

	var ret []byte
	if rows.Next() {
		if err := rows.Scan(&ret); err != nil {
			return nil, err
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ret, nil
}

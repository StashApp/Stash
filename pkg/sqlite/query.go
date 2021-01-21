package sqlite

import "github.com/stashapp/stash/pkg/models"

type queryBuilder struct {
	repository *repository

	body string

	whereClauses  []string
	havingClauses []string
	args          []interface{}

	sortAndPagination string
}

func (qb queryBuilder) executeFind() ([]int, int, error) {
	return qb.repository.executeFindQuery(qb.body, qb.args, qb.sortAndPagination, qb.whereClauses, qb.havingClauses)
}

func (qb *queryBuilder) addWhere(clauses ...string) {
	for _, clause := range clauses {
		if len(clause) > 0 {
			qb.whereClauses = append(qb.whereClauses, clause)
		}
	}
}

func (qb *queryBuilder) addHaving(clauses ...string) {
	for _, clause := range clauses {
		if len(clause) > 0 {
			qb.havingClauses = append(qb.havingClauses, clause)
		}
	}
}

func (qb *queryBuilder) addArg(args ...interface{}) {
	qb.args = append(qb.args, args...)
}

func (qb *queryBuilder) handleIntCriterionInput(c *models.IntCriterionInput, column string) {
	if c != nil {
		clause, count := getIntCriterionWhereClause(column, *c)
		qb.addWhere(clause)
		if count == 1 {
			qb.addArg(c.Value)
		}
	}
}

func (qb *queryBuilder) handleStringCriterionInput(c *models.StringCriterionInput, column string) {
	if c != nil {
		if modifier := c.Modifier; c.Modifier.IsValid() {
			switch modifier {
			case models.CriterionModifierIncludes:
				clause, thisArgs := getSearchBinding([]string{column}, c.Value, false)
				qb.addWhere(clause)
				qb.addArg(thisArgs...)
			case models.CriterionModifierExcludes:
				clause, thisArgs := getSearchBinding([]string{column}, c.Value, true)
				qb.addWhere(clause)
				qb.addArg(thisArgs...)
			case models.CriterionModifierEquals:
				qb.addWhere(column + " LIKE ?")
				qb.addArg(c.Value)
			case models.CriterionModifierNotEquals:
				qb.addWhere(column + " NOT LIKE ?")
				qb.addArg(c.Value)
			case models.CriterionModifierMatchesRegex:
				qb.addWhere(column + " regexp ?")
				qb.addArg(c.Value)
			case models.CriterionModifierNotMatchesRegex:
				qb.addWhere(column + " NOT regexp ?")
				qb.addArg(c.Value)
			default:
				clause, count := getSimpleCriterionClause(modifier, "?")
				qb.addWhere(column + " " + clause)
				if count == 1 {
					qb.addArg(c.Value)
				}
			}
		}
	}
}

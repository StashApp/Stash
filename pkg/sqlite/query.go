package sqlite

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/stashapp/stash/pkg/models"
)

type queryBuilder struct {
	repository *repository

	body string

	joins         joins
	whereClauses  []string
	havingClauses []string
	args          []interface{}
	withClauses   []string

	sortAndPagination string

	err error
}

func (qb queryBuilder) executeFind() ([]int, int, error) {
	if qb.err != nil {
		return nil, 0, qb.err
	}

	body := qb.body
	body += qb.joins.toSQL()

	return qb.repository.executeFindQuery(body, qb.args, qb.sortAndPagination, qb.whereClauses, qb.havingClauses, qb.withClauses)
}

func (qb queryBuilder) executeCount() (int, error) {
	if qb.err != nil {
		return 0, qb.err
	}

	body := qb.body
	body += qb.joins.toSQL()

	withClause := ""
	if len(qb.withClauses) > 0 {
		withClause = "WITH " + strings.Join(qb.withClauses, ", ") + " "
	}

	body = qb.repository.buildQueryBody(body, qb.whereClauses, qb.havingClauses)
	countQuery := withClause + qb.repository.buildCountQuery(body)
	return qb.repository.runCountQuery(countQuery, qb.args)
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

func (qb *queryBuilder) addWith(clauses ...string) {
	for _, clause := range clauses {
		if len(clause) > 0 {
			qb.withClauses = append(qb.withClauses, clause)
		}
	}
}

func (qb *queryBuilder) addArg(args ...interface{}) {
	qb.args = append(qb.args, args...)
}

func (qb *queryBuilder) join(table, as, onClause string) {
	newJoin := join{
		table:    table,
		as:       as,
		onClause: onClause,
	}

	qb.joins.add(newJoin)
}

func (qb *queryBuilder) addJoins(joins ...join) {
	qb.joins.add(joins...)
}

func (qb *queryBuilder) addFilter(f *filterBuilder) {
	err := f.getError()
	if err != nil {
		qb.err = err
		return
	}

	clause, args := f.generateWithClauses()
	if len(clause) > 0 {
		qb.addWith(clause)
	}

	if len(args) > 0 {
		// WITH clause always comes first and thus precedes alk args
		qb.args = append(args, qb.args...)
	}

	clause, args = f.generateWhereClauses()
	if len(clause) > 0 {
		qb.addWhere(clause)
	}

	if len(args) > 0 {
		qb.addArg(args...)
	}

	clause, args = f.generateHavingClauses()
	if len(clause) > 0 {
		qb.addHaving(clause)
	}

	if len(args) > 0 {
		qb.addArg(args...)
	}

	qb.addJoins(f.getAllJoins()...)
}

func (qb *queryBuilder) handleIntCriterionInput(c *models.IntCriterionInput, column string) {
	if c != nil {
		clause, args := getIntCriterionWhereClause(column, *c)
		qb.addWhere(clause)
		qb.addArg(args...)
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
				if _, err := regexp.Compile(c.Value); err != nil {
					qb.err = err
					return
				}
				qb.addWhere(fmt.Sprintf("(%s IS NOT NULL AND %[1]s regexp ?)", column))
				qb.addArg(c.Value)
			case models.CriterionModifierNotMatchesRegex:
				if _, err := regexp.Compile(c.Value); err != nil {
					qb.err = err
					return
				}
				qb.addWhere(fmt.Sprintf("(%s IS NULL OR %[1]s NOT regexp ?)", column))
				qb.addArg(c.Value)
			case models.CriterionModifierIsNull:
				qb.addWhere("(" + column + " IS NULL OR TRIM(" + column + ") = '')")
			case models.CriterionModifierNotNull:
				qb.addWhere("(" + column + " IS NOT NULL AND TRIM(" + column + ") != '')")
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

func (qb *queryBuilder) handleCountCriterion(countFilter *models.IntCriterionInput, primaryTable, joinTable, primaryFK string) {
	if countFilter != nil {
		clause, args := getCountCriterionClause(primaryTable, joinTable, primaryFK, *countFilter)

		qb.addWhere(clause)
		qb.addArg(args...)
	}
}

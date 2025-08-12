package postgres

import (
	"context"
	"database/sql"
)

// selectQuery is a query that selects data from the database.
type selectQuery struct {
	postgres      *postgres
	query         string
	keyValuePairs []any
	destination   any
	arguments     map[string]any
	debug         bool
}

// Select is an interface for selecting data from the database.
type Select interface {
	Debug() Select
	One(ctx context.Context) (found bool, err error)
	Many(ctx context.Context) (found bool, err error)
}

// Select is a query that selects data from the database.
func (query *selectQuery) Debug() Select {
	query.debug = true
	return query
}

// One selects a single row from the database.
func (query *selectQuery) One(ctx context.Context) (found bool, err error) {
	// Convert keyValuePairs to arguments map
	if query.arguments == nil {
		query.arguments, err = Pairs(query.keyValuePairs)
		if err != nil {
			return false, err
		}
	}

	// Debug query if either global debug or instance debug is enabled
	if query.debug {
		debugQuery(query.query, query.arguments)
	}

	preparedStatement, err := query.postgres.database.PrepareNamedContext(ctx, query.query)
	if err != nil {
		return false, err
	}
	defer preparedStatement.Close()

	err = preparedStatement.GetContext(ctx, query.destination, query.arguments)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Many selects multiple rows from the database.
func (query *selectQuery) Many(ctx context.Context) (found bool, err error) {
	// Convert keyValuePairs to arguments map
	if query.arguments == nil {
		query.arguments, err = Pairs(query.keyValuePairs)
		if err != nil {
			return false, err
		}
	}

	// Debug query if either global debug or instance debug is enabled
	if query.debug {
		debugQuery(query.query, query.arguments)
	}

	preparedStatement, err := query.postgres.database.PrepareNamedContext(ctx, query.query)
	if err != nil {
		return false, err
	}
	defer preparedStatement.Close()

	err = preparedStatement.SelectContext(ctx, query.destination, query.arguments) // Use SelectContext for slice results
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

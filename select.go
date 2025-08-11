package postgres

import (
	"context"
	"database/sql"
)

// selectQuery is a query that selects data from the database.
type selectQuery struct {
	p     *postgres
	query string
	kv    []any
	dest  any
	arg   map[string]any
	debug bool
}

// Select is an interface for selecting data from the database.
type Select interface {
	Debug() Select
	One(ctx context.Context) (found bool, err error)
	Many(ctx context.Context) (found bool, err error)
}

// Select is a query that selects data from the database.
func (q *selectQuery) Debug() Select {
	q.debug = true
	return q
}

// One selects a single row from the database.
func (q *selectQuery) One(ctx context.Context) (found bool, err error) {
	// Convert kv to arg map
	if q.arg == nil {
		q.arg, err = Pairs(q.kv)
		if err != nil {
			return false, err
		}
	}

	prep, err := q.p.database.PrepareNamedContext(ctx, q.query)
	if err != nil {
		return false, err
	}
	defer prep.Close()

	err = prep.GetContext(ctx, q.dest, q.arg)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Many selects multiple rows from the database.
func (q *selectQuery) Many(ctx context.Context) (found bool, err error) {
	// Convert kv to arg map
	if q.arg == nil {
		q.arg, err = Pairs(q.kv)
		if err != nil {
			return false, err
		}
	}

	prep, err := q.p.database.PrepareNamedContext(ctx, q.query)
	if err != nil {
		return false, err
	}
	defer prep.Close()

	err = prep.SelectContext(ctx, q.dest, q.arg) // Use SelectContext for slice results
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

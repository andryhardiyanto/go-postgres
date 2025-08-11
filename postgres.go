package postgres

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// postgres is the postgres database client.
type postgres struct {
	database *sqlx.DB
	debug    bool
}

// Postgres is the interface for the postgres database client.
type Postgres interface {
	Select(query string, dest any, kv ...any) Select
	Insert(query string, kv ...any) Exec
	Update(query string, kv ...any) Exec
	Delete(query string, kv ...any) Exec
	FromResult(from string) string
}

// Select is a query that selects data from the database.
func (p *postgres) Select(query string, dest any, kv ...any) Select {
	return &selectQuery{
		p:     p,
		query: query,
		kv:    kv,
		dest:  dest,
	}
}

// Insert is a query that inserts data into the database.
func (p *postgres) Insert(query string, kv ...any) Exec {
	return newExecQuery(p, query, kv)
}

// Update is a query that updates data in the database.
func (p *postgres) Update(query string, kv ...any) Exec {
	return newExecQuery(p, query, kv)
}

// Delete is a query that deletes data from the database.
func (p *postgres) Delete(query string, kv ...any) Exec {
	return newExecQuery(p, query, kv)
}

// FromResult is a query that returns the result of a query.
func (p *postgres) FromResult(from string) string {
	return fmt.Sprintf("%s%s", qResult, from)
}

package postgres

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// postgres is the postgres database client.
type postgres struct {
	database *sqlx.DB
}

// Postgres is the interface for the postgres database client.
type Postgres interface {
	Select(query string, destination any, keyValuePairs ...any) Select
	Insert(query string, keyValuePairs ...any) Exec
	Update(query string, keyValuePairs ...any) Exec
	Delete(query string, keyValuePairs ...any) Exec
	FromResult(from string) string
}

// Select is a query that selects data from the database.
func (postgresInstance *postgres) Select(query string, destination any, keyValuePairs ...any) Select {
	return &selectQuery{
		postgres:      postgresInstance,
		query:         query,
		keyValuePairs: keyValuePairs,
		destination:   destination,
	}
}

// Insert is a query that inserts data into the database.
func (postgresInstance *postgres) Insert(query string, keyValuePairs ...any) Exec {
	return newExecQuery(postgresInstance, query, keyValuePairs)
}

// Update is a query that updates data in the database.
func (postgresInstance *postgres) Update(query string, keyValuePairs ...any) Exec {
	return newExecQuery(postgresInstance, query, keyValuePairs)
}

// Delete is a query that deletes data from the database.
func (postgresInstance *postgres) Delete(query string, keyValuePairs ...any) Exec {
	return newExecQuery(postgresInstance, query, keyValuePairs)
}

// FromResult is a query that returns the result of a query.
func (postgresInstance *postgres) FromResult(from string) string {
	return fmt.Sprintf("%s%s", qResult, from)
}

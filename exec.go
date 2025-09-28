package postgres

import (
	"context"

	"github.com/pkg/errors"
)

// execQuery is a query that executes a statement against the database.
type execQuery struct {
	postgres      *postgres
	query         string
	keyValuePairs []any
	pipeline      *pipeline
	debug         bool
}

// ExecResult is the result of an exec query.
type ExecResult struct {
	ids map[string]any
}

type Exec interface {
	Debug() Exec
	Exec(ctx context.Context) (any, error)
	ExecInTx(ctx context.Context) (result *ExecResult, err error)
	Insert(query string, keyValuePairs ...any) Exec
	Update(query string, keyValuePairs ...any) Exec
	Delete(query string, keyValuePairs ...any) Exec
	Select(query string, destination any, keyValuePairs ...any) Exec
	Wrap(exec Exec) Exec
	FromResult(from string) string
}

func newExecQuery(postgresInstance *postgres, query string, keyValuePairs []any) Exec {
	return &execQuery{
		postgres:      postgresInstance,
		query:         query,
		keyValuePairs: keyValuePairs,
		pipeline:      NewPipeline(),
	}
}

func (e *execQuery) Debug() Exec {
	e.debug = true
	return e
}

func (e *execQuery) FromResult(from string) string {
	return e.postgres.FromResult(e.pipeline.uniqueQuery(from))
}
func (e *execQuery) Exec(ctx context.Context) (any, error) {
	if e.pipeline.isTrans() {
		return 0, errors.New("invalid operation: this query is part of a transaction pipeline. Please use ExecInTx() method instead of Exec() to execute transaction-based queries")
	}
	arguments, err := Pairs(e.keyValuePairs)
	if err != nil {
		return 0, err
	}

	// Debug query if either global debug or instance debug is enabled
	if e.debug {
		debugQuery(e.query, arguments)
	}

	if queryType(e.query) == qInsert {
		return insert(ctx, e.postgres.database, e.query, arguments)
	} else if queryType(e.query) == qDelete {
		return delete(ctx, e.postgres.database, e.query, arguments)
	}
	return update(ctx, e.postgres.database, e.query, arguments)
}

func (e *execQuery) ExecInTx(ctx context.Context) (result *ExecResult, err error) {
	if !e.pipeline.isTrans() {
		return nil, errors.New("invalid operation: no transaction pipeline found. Please use Insert(), Update(), or Delete() methods to build a transaction pipeline before calling ExecInTx()")
	}
	e.pipeline.addFirstPipeline(e.query, e.keyValuePairs)

	transaction, err := e.postgres.database.Beginx()
	if err != nil {
		return nil, err
	}
	defer func() {
		if panicValue := recover(); panicValue != nil {
			_ = transaction.Rollback()
			panic(panicValue)
		} else if err != nil {
			_ = transaction.Rollback()
		} else {
			err = transaction.Commit()
		}
	}()

	result, err = e.pipeline.runPipeline(ctx, transaction, e.debug)

	return
}

func (e *execQuery) Wrap(exec Exec) Exec {
	execQuery, ok := exec.(*execQuery)
	if !ok || execQuery == nil {
		return e
	}
	execQuery.query = e.pipeline.uniqueQuery(execQuery.query)
	execQuery.pipeline.addFirstPipeline(execQuery.query, execQuery.keyValuePairs)
	e.pipeline.appendPipeline(execQuery.pipeline)
	return e
}

func (e *execQuery) Insert(query string, keyValuePairs ...any) Exec {
	e.pipeline.addPipeline(query, keyValuePairs)
	return e
}

func (e *execQuery) Update(query string, keyValuePairs ...any) Exec {
	e.pipeline.addPipeline(query, keyValuePairs)
	return e
}

func (e *execQuery) Delete(query string, keyValuePairs ...any) Exec {
	e.pipeline.addPipeline(query, keyValuePairs)
	return e
}
func (e *execQuery) Select(query string, destination any, keyValuePairs ...any) Exec {
	e.pipeline.addPipeline(query, keyValuePairs)
	return e
}
func (e *ExecResult) TxResult(query string) any {
	return e.ids[query]
}

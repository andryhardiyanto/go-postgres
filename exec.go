package postgres

import (
	"context"

	"github.com/pkg/errors"
)

// execQuery is a query that executes a statement against the database.
type execQuery struct {
	p        *postgres
	query    string
	kv       []any
	pipeline *pipeline
	debug    bool
}

// ExecResult is the result of an exec query.
type ExecResult struct {
	ids map[string]any
}

type Exec interface {
	Debug() Exec
	Exec(ctx context.Context) (any, error)
	ExecInTx(ctx context.Context) (res *ExecResult, err error)
	Insert(query string, kv ...any) Exec
	Update(query string, kv ...any) Exec
	Delete(query string, kv ...any) Exec
	Wrap(exec Exec) Exec
	FromResult(from string) string
}

func newExecQuery(p *postgres, query string, kv []any) Exec {
	return &execQuery{
		p:        p,
		query:    query,
		kv:       kv,
		pipeline: NewPipeline(),
	}
}

func (e *execQuery) Debug() Exec {
	e.debug = true
	return e
}

func (e *execQuery) FromResult(from string) string {
	return e.p.FromResult(e.pipeline.uniqueQuery(from))
}
func (e *execQuery) Exec(ctx context.Context) (any, error) {
	if e.pipeline.isTrans() {
		return 0, errors.New("transaction please use ExecInTx()")
	}
	arg, err := Pairs(e.kv)
	if err != nil {
		return 0, err
	}

	if e.p.debug {
		debugQuery(e.query, arg, ctx)
	}

	if queryType(e.query) == qInsert {
		return insert(ctx, e.p.database, e.query, arg)
	} else if queryType(e.query) == qDelete {
		return nil, delete(ctx, e.p.database, e.query, arg)
	}
	return nil, update(ctx, e.p.database, e.query, arg)
}

func (e *execQuery) ExecInTx(ctx context.Context) (res *ExecResult, err error) {
	if !e.pipeline.isTrans() {
		return nil, errors.New("not transaction please use Exec()")
	}
	e.pipeline.addFirstPipeline(e.query, e.kv)

	tx, err := e.p.database.Beginx()
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	res, err = e.pipeline.runPipeline(ctx, tx)

	return
}

func (e *execQuery) Wrap(exec Exec) Exec {
	ex, ok := exec.(*execQuery)
	if !ok || ex == nil {
		return e
	}
	ex.query = e.pipeline.uniqueQuery(ex.query)
	ex.pipeline.addFirstPipeline(ex.query, ex.kv)
	e.pipeline.appendPipeline(ex.pipeline)
	return e
}

func (e *execQuery) Insert(query string, kv ...any) Exec {
	e.pipeline.addPipeline(query, kv)
	return e
}

func (e *execQuery) Update(query string, kv ...any) Exec {
	e.pipeline.addPipeline(query, kv)
	return e
}

func (e *execQuery) Delete(query string, kv ...any) Exec {
	e.pipeline.addPipeline(query, kv)
	return e
}

func (e *ExecResult) TxResult(query string) any {
	return e.ids[query]
}

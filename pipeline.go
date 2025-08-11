package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// pipeline is a pipeline of queries.
type pipeline struct {
	m map[string][]any
	k []string
}

// NewPipeline creates a new pipeline
func NewPipeline() *pipeline {
	return &pipeline{
		m: make(map[string][]any),
		k: make([]string, 0),
	}
}

// runPipeline runs a pipeline of queries in a transaction
// and returns the ExecResult
func (p *pipeline) runPipeline(ctx context.Context, tx *sqlx.Tx) (*ExecResult, error) {
	res := &ExecResult{
		ids: make(map[string]any),
	}
	for _, q := range p.k {
		var id any
		v := p.m[q]

		arg, err := PairsHook(v, res.ids, qResult)
		if err != nil {
			return nil, err
		}

		qType := queryType(q)
		switch {
		case strings.EqualFold(qType, qInsert):
			id, err = insertTx(ctx, tx, q, arg)
		case strings.EqualFold(qType, qDelete):
			err = deleteTx(ctx, tx, q, arg)
		default:
			err = updateTx(ctx, tx, q, arg)
		}
		if err != nil {
			return nil, err
		}
		res.ids[q] = id
	}
	return res, nil
}

// addPipeline adds a query to the pipeline
func (p *pipeline) addPipeline(query string, kv []any) {
	query = p.uniqueQuery(query)
	p.m[query] = kv
	p.k = append(p.k, query)
}

// addFirstPipeline adds a query to the beginning of the pipeline
func (p *pipeline) addFirstPipeline(query string, kv []any) {
	if query == "" {
		return
	}
	query = p.uniqueQuery(query)
	p.m[query] = kv
	p.k = append([]string{query}, p.k...)
}

// appendPipeline appends a pipeline to the current pipeline
func (p *pipeline) appendPipeline(pip *pipeline) {
	for _, q := range pip.k {
		q = p.uniqueQuery(q)
		p.m[q] = pip.m[q]
		p.k = append(p.k, q)
	}
}

// isTrans returns true if the pipeline has at least one query
func (p *pipeline) isTrans() bool {
	return len(p.k) > 0
}

// add uniqueness to duplicate query
func (p *pipeline) uniqueQuery(query string) string {
	_, ok := p.m[query]
	if ok {
		query = fmt.Sprintf("%s/*%d*/", query, len(p.k))
	}

	return query
}

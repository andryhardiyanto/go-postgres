package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// pipeline represents a sequence of database queries that will be executed in a transaction.
// It maintains the order of queries and handles parameter resolution between queries.
type pipeline struct {
	queryParameters map[string][]any // Query to parameters mapping
	queryKeys       []string         // Ordered list of queries
}

// NewPipeline creates a new empty pipeline instance.
// The pipeline can be used to chain multiple database operations
// that should be executed atomically in a transaction.
func NewPipeline() *pipeline {
	return &pipeline{
		queryParameters: make(map[string][]any),
		queryKeys:       make([]string, 0),
	}
}

// runPipeline executes all queries in the pipeline within the provided transaction.
// It processes queries in the order they were added and resolves parameter dependencies
// between queries using the qResult hook mechanism.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - tx: Database transaction
//   - debug: Enable debug logging for queries
//
// Returns:
//   - *ExecResult: Contains the results and IDs from executed queries
//   - error: Any error that occurred during execution
func (p *pipeline) runPipeline(ctx context.Context, tx *sqlx.Tx, debug bool) (*ExecResult, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}

	if len(p.queryKeys) == 0 {
		return &ExecResult{ids: make(map[string]any)}, nil
	}

	result := &ExecResult{
		ids: make(map[string]any, len(p.queryKeys)),
	}

	for index, query := range p.queryKeys {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		parameters, exists := p.queryParameters[query]
		if !exists {
			return nil, fmt.Errorf("query parameters not found for query at index %d", index)
		}

		arguments, err := PairsHook(parameters, result.ids, qResult)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve parameters for query at index %d: %w", index, err)
		}

		// Debug transaction query if enabled
		if debug {
			debugQuery(query, arguments)
		}

		var queryID any
		queryType := queryType(query)

		switch {
		case strings.EqualFold(queryType, qInsert):
			queryID, err = insertTx(ctx, tx, query, arguments)
		case strings.EqualFold(queryType, qDelete):
			queryID, err = deleteTx(ctx, tx, query, arguments)
		default:
			queryID, err = updateTx(ctx, tx, query, arguments)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to execute query at index %d: %w", index, err)
		}

		result.ids[query] = queryID
	}

	return result, nil
}

// addPipeline adds a query with its parameters to the end of the pipeline.
// If the same query already exists, it will be made unique by appending a comment.
//
// Parameters:
//   - query: The SQL query to add
//   - keyValuePairs: Key-value pairs for query parameters
func (p *pipeline) addPipeline(query string, keyValuePairs []any) {
	if query == "" {
		return
	}

	uniqueQuery := p.uniqueQuery(query)
	p.queryParameters[uniqueQuery] = keyValuePairs
	p.queryKeys = append(p.queryKeys, uniqueQuery)
}

// addFirstPipeline adds a query to the beginning of the pipeline.
// This is useful when you need to ensure a query executes before all others.
//
// Parameters:
//   - query: The SQL query to add at the beginning
//   - keyValuePairs: Key-value pairs for query parameters
func (p *pipeline) addFirstPipeline(query string, keyValuePairs []any) {
	if query == "" {
		return
	}

	uniqueQuery := p.uniqueQuery(query)
	p.queryParameters[uniqueQuery] = keyValuePairs
	p.queryKeys = append([]string{uniqueQuery}, p.queryKeys...)
}

// appendPipeline merges another pipeline into the current one.
// All queries from the source pipeline are added to the end of the current pipeline.
// Query uniqueness is maintained during the merge process.
//
// Parameters:
//   - sourcePipeline: The pipeline to append to the current one
func (p *pipeline) appendPipeline(sourcePipeline *pipeline) {
	if sourcePipeline == nil || len(sourcePipeline.queryKeys) == 0 {
		return
	}

	// Pre-allocate slice capacity for better performance
	if cap(p.queryKeys) < len(p.queryKeys)+len(sourcePipeline.queryKeys) {
		newQueryKeys := make([]string, len(p.queryKeys), len(p.queryKeys)+len(sourcePipeline.queryKeys))
		copy(newQueryKeys, p.queryKeys)
		p.queryKeys = newQueryKeys
	}

	for _, query := range sourcePipeline.queryKeys {
		originalQuery := query
		uniqueQuery := p.uniqueQuery(query)

		// Copy parameters from source pipeline
		if parameters, exists := sourcePipeline.queryParameters[originalQuery]; exists {
			p.queryParameters[uniqueQuery] = parameters
			p.queryKeys = append(p.queryKeys, uniqueQuery)
		}
	}
}

// isTrans returns true if the pipeline contains at least one query.
// This is used to determine if a transaction should be started.
func (p *pipeline) isTrans() bool {
	return len(p.queryKeys) > 0
}

// Len returns the number of queries in the pipeline.
func (p *pipeline) Len() int {
	return len(p.queryKeys)
}

// Clear removes all queries from the pipeline, resetting it to an empty state.
func (p *pipeline) Clear() {
	p.queryParameters = make(map[string][]any)
	p.queryKeys = p.queryKeys[:0] // Keep underlying array but reset length
}

// uniqueQuery ensures query uniqueness by appending a comment with an index
// if the same query already exists in the pipeline.
//
// This prevents map key collisions while maintaining query functionality.
func (p *pipeline) uniqueQuery(query string) string {
	if _, exists := p.queryParameters[query]; exists {
		query = fmt.Sprintf("%s/*%d*/", query, len(p.queryKeys))
	}
	return query
}

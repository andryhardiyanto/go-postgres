package postgres

import (
	"context"

	"github.com/andryhardiyanto/go-async"
	"github.com/pkg/errors"
)

type pagination[T any] struct {
	postgres Postgres
	debug    bool
}

func NewPagination[T any](repo Postgres) Pagination[T] {
	return &pagination[T]{
		postgres: repo,
	}
}

type Pagination[T any] interface {
	Debug() Pagination[T]
	Offset(ctx context.Context, req *RequestPaginationOffset) (*ResponsePaginationOffset[T], error)
	Cursor(ctx context.Context, req *RequestPaginationCursor) (*ResponsePaginationCursor[T], error)
}

func (p *pagination[T]) Debug() Pagination[T] {
	p.debug = true
	return p
}

func (p *pagination[T]) Offset(ctx context.Context, req *RequestPaginationOffset) (resp *ResponsePaginationOffset[T], err error) {
	asyncRunner := async.NewAsyncRunner()

	if req.Page < 1 {
		req.Page = 1
	}

	if req.Size < 1 {
		req.Size = 10
	}

	var total int64
	var items []T

	err = asyncRunner.
		RunInAsync().
		Task(async.Bind(&total, func(ctx context.Context) (int64, error) {
			var count int64
			if req.QueryCount != "" {
				_, err = p.postgres.Select(req.QueryCount, &count, req.Kv...).One(ctx)
				if err != nil {
					return 0, err
				}
			}
			return count, nil
		})).
		Task(async.Bind(&items, func(ctx context.Context) ([]T, error) {
			var data []T
			_, err = p.postgres.Select(req.Query, &data, req.Kv...).Many(ctx)
			if err != nil {
				return nil, err
			}
			return data, nil
		})).
		Go(ctx)
	if err != nil {
		return
	}

	page := req.Page

	if len(items) == 0 {
		items = make([]T, 0)
	}

	return &ResponsePaginationOffset[T]{
		Page:       page,
		Items:      items,
		TotalItems: total,
		TotalPages: req.GetTotalPages(total),
	}, nil
}

func (p *pagination[T]) Cursor(ctx context.Context, req *RequestPaginationCursor) (*ResponsePaginationCursor[T], error) {
	if req.Size < 1 {
		req.Size = 10
	}

	if len(req.Sorts) == 0 {
		return nil, errors.New("cursor sorts is required")
	}

	cursor, err := getCursorTokenFromEncoded(&RequestGetCursorTokenFromEncoded{
		Token:       req.Token,
		Sorts:       req.Sorts,
		Query:       req.Query,
		Kv:          req.Kv,
		IsCustomCTE: req.IsCustomCTE,
	})
	if err != nil {
		return nil, err
	}

	limitWithExtra := req.Size + 1
	cursor.Kv = append(cursor.Kv, "limit", limitWithExtra)
	cursor.Query += " LIMIT :limit"

	var items = make([]T, 0)
	var total int64

	asyncRunner := async.NewAsyncRunner()
	err = asyncRunner.
		RunInAsync().
		Task(async.Bind(&items, func(ctx context.Context) ([]T, error) {
			var tempItems = make([]T, 0)

			selectQuery := p.postgres.Select(cursor.Query, &tempItems, cursor.Kv...)
			if p.debug {
				selectQuery = selectQuery.Debug()
			}

			found, err := selectQuery.Many(ctx)
			if err != nil {
				return nil, err
			}

			if !found {
				return tempItems, nil
			}

			return tempItems, nil
		})).
		Task(async.Bind(&total, func(ctx context.Context) (int64, error) {
			return p.getTotalItems(ctx, req, cursor)
		})).
		Go(ctx)

	if err != nil {
		return nil, err
	}

	hasMore := len(items) > req.Size
	if hasMore {
		items = items[:req.Size]
	}

	if cursor.ReverseResult {
		reverseSlice(items)
	}

	var prevT, nextT *string
	if len(items) > 0 {
		firstItem := items[0]
		lastItem := items[len(items)-1]

		nextOp, prevOp := " > ", " < "
		if req.Sorts[0].IsDesc {
			nextOp, prevOp = " < ", " > "
		}

		if req.Token != "" {
			if !(cursor.IsPrev && !hasMore) {
				enc, _ := makeCursorFromItem(firstItem, cursor.Keys, true, prevOp, total)
				prevT = &enc
			}
		}

		if hasMore || cursor.IsPrev {
			enc, _ := makeCursorFromItem(lastItem, cursor.Keys, false, nextOp, total)
			nextT = &enc
		}
	}

	return &ResponsePaginationCursor[T]{
		PreviousToken: prevT,
		NextToken:     nextT,
		TotalItems:    total,
		Items:         items,
	}, nil
}

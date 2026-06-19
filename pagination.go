package postgres

import (
	"context"

	"github.com/andryhardiyanto/go-async"
	"github.com/pkg/errors"
)

type OptionPagination[T any] func(*pagination[T])

type pagination[T any] struct {
	postgres    Postgres
	debug       bool
	minPage     int
	maxPage     int
	minPageSize int
	maxPageSize int
}

func NewPagination[T any](repo Postgres, opts ...OptionPagination[T]) Pagination[T] {
	pagination := &pagination[T]{
		postgres:    repo,
		minPage:     1,
		maxPage:     100,
		minPageSize: 10,
		maxPageSize: 50,
	}

	for _, opt := range opts {
		opt(pagination)
	}

	return pagination
}

type Pagination[T any] interface {
	Debug() Pagination[T]
	Offset(ctx context.Context, req *RequestPaginationOffset) (*ResponsePaginationOffset[T], error)
	Cursor(ctx context.Context, req *RequestPaginationCursor) (*ResponsePaginationCursor[T], error)
	GetKvLimit(pageSize int) []any
	GetKvOffset(page, pageSize int) []any
}

func (p *pagination[T]) Debug() Pagination[T] {
	p.debug = true
	return p
}

func (p *pagination[T]) getPage(page int) int {
	if page < p.minPage {
		page = p.minPage
	}

	if page > p.maxPage {
		page = p.maxPage
	}

	return page
}

func (p *pagination[T]) GetKvLimit(pageSize int) []any {
	return []any{"limit", p.getPageSize(pageSize)}
}

func (p *pagination[T]) GetKvOffset(page, pageSize int) []any {
	return []any{"offset", (p.getPage(page) - 1) * p.getPageSize(pageSize)}
}

func (p *pagination[T]) getPageSize(pageSize int) int {
	if pageSize < p.minPageSize {
		pageSize = p.minPageSize
	}

	if pageSize > p.maxPageSize {
		pageSize = p.maxPageSize
	}

	return pageSize
}

func (p *pagination[T]) Offset(ctx context.Context, req *RequestPaginationOffset) (resp *ResponsePaginationOffset[T], err error) {
	asyncRunner := async.NewAsyncRunner()

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

	limitWithExtra := p.getPageSize(req.Size) + 1
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

package pagination

import (
	"context"

	"math"

	"github.com/andryhardiyanto/go-async"
)

func (r *RequestPaginationOffset) GetOffset() int64 {
	return (r.Page - 1) * r.Size
}

func (r *RequestPaginationOffset) GetLimit() int64 {
	return r.Size
}

func (r *RequestPaginationOffset) GetTotalPage(total int64) int64 {
	return int64(math.Ceil(float64(total) / float64(r.Size)))
}

func (r *RequestPaginationOffset) GetTotalItem(total int64) int64 {
	return total - (r.Page-1)*r.Size
}

func (r *RequestPaginationOffset) GetHasMore(total int64) bool {
	return r.Page < r.GetTotalPage(total)
}

func (r *RequestPaginationOffset) GetHasPrevious() bool {
	return r.Page > 1
}

func PaginationOffset[T any](ctx context.Context, req *RequestPaginationOffset) (resp *ResponsePaginationOffset[T], err error) {
	asyncRunner := async.NewAsyncRunner()

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 {
		req.Size = 10
	}

	var total int64
	var items []*T

	err = asyncRunner.
		RunInAsync().
		Task(&total, func() (any, error) {
			var count int64
			if req.QueryCount != "" {
				_, err = req.Repo.Select(req.QueryCount, &count, req.Kv...).One(ctx)
				if err != nil {
					return nil, err
				}
			}
			return count, nil
		}).
		Task(&items, func() (any, error) {
			var data []*T
			_, err = req.Repo.Select(req.Query, &data, req.Kv...).Many(ctx)
			if err != nil {
				return nil, err
			}

			return data, nil
		}).
		Go(ctx)
	if err != nil {
		return
	}

	page := req.Page

	if len(items) == 0 {
		items = make([]*T, 0)
	}

	return &ResponsePaginationOffset[T]{
		Page:       page,
		Items:      items,
		TotalItems: total,
		TotalPages: req.GetTotalPage(total),
	}, nil
}

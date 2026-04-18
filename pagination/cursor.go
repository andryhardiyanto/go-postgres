package pagination

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/andryhardiyanto/go-async"
)

func PaginationCursor[T any](ctx context.Context, req *RequestPaginationCursor) (*ResponsePaginationCursor[T], error) {
	cursor, err := GetCursorTokenFromEncoded(&RequestGetCursorTokenFromEncoded{
		Token:         req.Token,
		CursorColumns: req.CursorColumns,
		DefaultOrder:  req.DefaultOrder,
		Query:         req.Query,
		Kv:            req.Kv,
		IsCustomCTE:   req.IsCustomCTE,
		Keys:          req.Keys,
	})
	if err != nil {
		return nil, err
	}

	cursor.Query += " LIMIT :limit"

	var items = make([]T, 0)
	var total int64
	asyncRunner := async.NewAsyncRunner()

	err = asyncRunner.
		RunInAsync().
		Task(&items, func() (any, error) {
			var tempItems = make([]T, 0)

			found, err := req.Repo.Select(cursor.Query, &tempItems, cursor.Kv...).Many(ctx)
			if err != nil {
				return nil, err
			}
			if !found {
				return tempItems, nil
			}

			return tempItems, nil
		}).
		Task(&total, func() (any, error) {
			var tempTotal int64
			if req.Token != "" {
				v, ok := cursor.CursorToken["total_items"]
				if !ok || v == nil {
					return 0, fmt.Errorf("total_items missing in cursor token")
				}

				switch x := v.(type) {
				case int:
					return int64(x), nil
				case int64:
					return x, nil
				case float64:
					return int64(x), nil
				case string:
					n, err := strconv.ParseInt(x, 10, 64)
					if err != nil {
						return 0, err
					}
					return n, nil
				case json.Number:
					n, err := x.Int64()
					if err != nil {
						return 0, err
					}
					return n, nil
				default:
					return 0, nil
				}
			}

			found, err := req.Repo.Select(req.QueryCount, &tempTotal, cursor.Kv...).One(ctx)
			if err != nil {
				return nil, err
			}
			if !found {
				return tempTotal, nil
			}

			return tempTotal, nil
		}).
		Go(ctx)
	if err != nil {
		return nil, err
	}

	hasMore := false
	if len(items) > req.Size {
		hasMore = true
		items = items[0:req.Size]
	}

	if cursor.ReverseResult {
		ReverseSlice(items)
	}

	var previousCursor *string
	var nextCursor *string

	if len(items) > 0 {
		firstItem := items[0]
		lastItem := items[len(items)-1]

		nextOp, prevOp := cursorOperators(req.DefaultOrder)

		if req.Token != "" {
			if !(cursor.IsPrev && !hasMore) {
				encodedPrev, err := makeCursorFromItem(firstItem, req.Keys, true, prevOp, total)
				if err != nil {
					return nil, err
				}
				previousCursor = &encodedPrev
			}
		}

		if hasMore {
			encodedNext, err := makeCursorFromItem(lastItem, req.Keys, false, nextOp, total)
			if err != nil {
				return nil, err
			}
			nextCursor = &encodedNext
		}
	}

	return &ResponsePaginationCursor[T]{
		PreviousToken: previousCursor,
		NextToken:     nextCursor,
		Items:         items,
		TotalItems:    total,
	}, nil
}

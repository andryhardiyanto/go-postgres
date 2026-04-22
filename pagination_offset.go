package postgres

import (
	"math"
)

type RequestPaginationOffset struct {
	Page       int64
	Size       int64
	Query      string
	Kv         []any
	QueryCount string
}

type ResponsePaginationOffset[T any] struct {
	Page       int64 `json:"page"`
	Items      []T   `json:"items"`
	TotalItems int64 `json:"total_items"`
	TotalPages int64 `json:"total_pages"`
}

func (r *RequestPaginationOffset) GetTotalPages(total int64) int64 {
	return int64(math.Ceil(float64(total) / float64(r.Size)))
}

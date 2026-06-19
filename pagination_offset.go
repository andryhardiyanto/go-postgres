package postgres

import (
	"math"
)

type RequestPaginationOffset struct {
	Page       int
	Size       int
	Query      string
	Kv         []any
	QueryCount string
}

type ResponsePaginationOffset[T any] struct {
	Page       int   `json:"page"`
	Items      []T   `json:"items"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

func (r *RequestPaginationOffset) GetTotalPages(total int64) int {
	return int(math.Ceil(float64(total) / float64(r.Size)))
}

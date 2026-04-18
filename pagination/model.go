package pagination

import goPostgres "github.com/andryhardiyanto/go-postgres"

type CursorToken map[string]any

type RequestGetCursorTokenFromEncoded struct {
	Token         string   // Encoded cursor token string
	CursorColumns []string // SQL expressions used as cursor columns, e.g. []{"c.created_at", "c.id"}
	DefaultOrder  string   // ORDER BY clause without the "ORDER BY" keyword, e.g. "c.created_at DESC, c.id DESC"
	Query         string   // Base SQL query WITHOUT ORDER BY and WITHOUT cursor conditions
	Kv            []any    // Existing query parameters (named), in the form key, value, key, value ...
	IsCustomCTE   bool     // true if the Query uses fmt.Sprintf with placeholders (custom CTE)
	Keys          []string // Keys in the token and struct field names, e.g. []{"created_at", "id"}
}

type ResponseGetCursorTokenFromEncoded struct {
	IsPrev        bool
	Query         string
	ReverseResult bool
	CursorToken   map[string]any
	Kv            []any
}

type RequestPaginationCursor struct {
	Repo          goPostgres.Postgres
	Size          int
	Token         string
	CursorColumns []string
	DefaultOrder  string
	Query         string
	Kv            []any
	IsCustomCTE   bool
	Keys          []string
	QueryCount    string
}

type RequestPaginationOffset struct {
	Page       int64
	Size       int64
	Query      string
	Kv         []any
	QueryCount string
	Repo       goPostgres.Postgres
}

type ResponsePaginationOffset[T any] struct {
	Page       int64 `json:"page"`
	Items      []*T  `json:"items"`
	TotalItems int64 `json:"total_items"`
	TotalPages int64 `json:"total_pages"`
}

type ResponsePaginationCursor[T any] struct {
	PreviousToken *string `json:"previous_token"`
	NextToken     *string `json:"next_token"`
	TotalItems    int64   `json:"total_items"`
	Items         []T     `json:"items"`
}

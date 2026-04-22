package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type CursorToken map[string]any

type structMetadata struct {
	indexMap map[string]int
	typeName string
}

var structMetadataRegistry sync.Map

// SortField defines the mapping between a database column and a token key for ordering.
type SortField struct {
	// Column is the full SQL column name, including table aliases if necessary.
	// Example: "u.created_at" or "p.id".
	Column string

	// Key is the field name used as the key within the JSON map/token.
	// This should match the JSON tag or DB tag on the target struct.
	// Example: "created_at" or "id".
	Key string

	// IsDesc determines the sort direction. If true, it uses DESC; if false, it uses ASC.
	IsDesc bool
}

// RequestGetCursorTokenFromEncoded holds the parameters required to decode a cursor
// and rebuild the SQL query for the next or previous page.
type RequestGetCursorTokenFromEncoded struct {
	// Token is the base64 encoded string containing the cursor state from a previous request.
	Token string

	// Sorts is a list of sorting definitions that acts as the "Single Source of Truth."
	// This field is automatically used to build two components:
	// 1. The SQL ORDER BY clause.
	// 2. The Row Comparison filter condition (e.g., (col1, col2) > (:val1, :val2)).
	//
	// IMPORTANT: The order in this slice determines sort priority. The last element
	// MUST be a unique column (such as a Primary Key/ID) to ensure deterministic
	// pagination (preventing skipped or duplicate data across pages).
	Sorts []SortField

	// Query is the base SQL query excluding the ORDER BY clause and cursor conditions.
	Query string

	// Kv contains existing query parameters in a key-value pair format (named parameters).
	Kv []any

	// IsCustomCTE should be set to true if the Query uses fmt.Sprintf placeholders (%s).
	// Typically, the first placeholder is for the cursor condition and the second for ORDER BY.
	IsCustomCTE bool

	// Keys is a list of field keys to be extracted from the token.
	// Usually populated automatically based on the 'Key' fields defined in Sorts.
	Keys []string
}

type RequestPaginationCursor struct {
	Query       string
	QueryCount  string
	Token       string
	Size        int
	IsCustomCTE bool
	Kv          []any
	Sorts       []SortField
}

type ResponsePaginationCursor[T any] struct {
	PreviousToken *string `json:"previous_token"`
	NextToken     *string `json:"next_token"`
	TotalItems    int64   `json:"total_items"`
	Items         []T     `json:"items"`
}

type ResponseGetCursorTokenFromEncoded struct {
	IsPrev        bool
	Query         string
	ReverseResult bool
	CursorToken   map[string]any
	Kv            []any
	Keys          []string
}

func (p *pagination[T]) getTotalItems(ctx context.Context, req *RequestPaginationCursor, cursor *ResponseGetCursorTokenFromEncoded) (int64, error) {
	if req.Token != "" {
		if v, ok := cursor.CursorToken["total_items"]; ok && v != nil {
			return parseTotalItems(v)
		}
	}

	if req.QueryCount == "" {
		return 0, nil
	}

	var tempTotal int64
	selectQuery := p.postgres.Select(req.QueryCount, &tempTotal, cursor.Kv...)
	if p.debug {
		selectQuery = selectQuery.Debug()
	}

	found, err := selectQuery.One(ctx)
	if err != nil || !found {
		return 0, err
	}
	return tempTotal, nil
}

func makeCursorFromItem[T any](item T, keys []string, isPrev bool, operator string, totalItems int64) (string, error) {
	v := reflect.Indirect(reflect.ValueOf(item))
	typ := v.Type()

	meta := getMetadata(typ)

	m := make(map[string]any, len(keys)+3)

	for _, key := range keys {
		fieldIdx, ok := meta.indexMap[key]
		if !ok {
			fieldIdx, ok = meta.indexMap[strings.ToLower(key)]
		}

		if !ok {
			return "", fmt.Errorf("field '%s' not found in struct %s", key, meta.typeName)
		}

		m[key] = v.Field(fieldIdx).Interface()
	}

	m["operator"] = operator
	m["is_prev"] = isPrev
	m["total_items"] = totalItems

	return encodeCursorToken(m)
}

func getMetadata(typ reflect.Type) *structMetadata {
	if val, ok := structMetadataRegistry.Load(typ); ok {
		return val.(*structMetadata)
	}
	meta := &structMetadata{
		indexMap: make(map[string]int),
		typeName: typ.Name(),
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}

		name := strings.ToLower(field.Name)
		jsonTag := strings.Split(field.Tag.Get("json"), ",")[0]
		dbTag := strings.Split(field.Tag.Get("db"), ",")[0]

		meta.indexMap[name] = i
		if jsonTag != "" && jsonTag != "-" {
			meta.indexMap[jsonTag] = i
		}
		if dbTag != "" && dbTag != "-" {
			meta.indexMap[dbTag] = i
		}
	}

	structMetadataRegistry.Store(typ, meta)
	return meta
}

func parseTotalItems(v any) (int64, error) {
	switch x := v.(type) {
	case float64:
		return int64(x), nil
	case int64:
		return x, nil
	case string:
		return strconv.ParseInt(x, 10, 64)
	case json.Number:
		return x.Int64()
	default:
		return 0, nil
	}
}

func getCursorTokenFromEncoded(req *RequestGetCursorTokenFromEncoded) (*ResponseGetCursorTokenFromEncoded, error) {
	orderBy, cols, keys := buildFromSorts(req.Sorts, false)
	if req.Token == "" {
		finalQuery := req.Query
		if req.IsCustomCTE {
			finalQuery = fmt.Sprintf(req.Query, "", orderBy)
		} else {
			finalQuery += orderBy
		}

		return &ResponseGetCursorTokenFromEncoded{
			IsPrev:        false,
			Query:         finalQuery,
			ReverseResult: false,
			Kv:            req.Kv,
			Keys:          keys,
		}, nil
	}

	token, err := decodeCursorToken(req.Token)
	if err != nil {
		return nil, err
	}
	op, _ := token["operator"].(string)
	isPrev, _ := token["is_prev"].(bool)
	revOrderBy, _, _ := buildFromSorts(req.Sorts, isPrev)
	condition := buildRowComparison(cols, keys, op)

	for _, k := range keys {
		if val, ok := token[k]; ok {
			req.Kv = append(req.Kv, k, val)
		}
	}

	finalQuery := req.Query
	if req.IsCustomCTE {
		finalQuery = fmt.Sprintf(req.Query, " AND "+condition, revOrderBy)
	} else {
		finalQuery += " AND " + condition + revOrderBy
	}

	return &ResponseGetCursorTokenFromEncoded{
		IsPrev:        isPrev,
		Query:         finalQuery,
		ReverseResult: isPrev,
		CursorToken:   token,
		Kv:            req.Kv,
		Keys:          keys,
	}, nil
}

func encodeCursorToken(token any) (string, error) {
	bytes, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(bytes), nil
}

func decodeCursorToken(encoded string) (CursorToken, error) {
	bytes, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var token CursorToken
	err = json.Unmarshal(bytes, &token)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func buildRowComparison(cols []string, keys []string, operator string) string {
	var params []string
	for _, k := range keys {
		params = append(params, ":"+k)
	}
	return fmt.Sprintf("(%s) %s (%s)", strings.Join(cols, ", "), operator, strings.Join(params, ", "))
}

func reverseSlice[T any](items []T) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

func buildFromSorts(sorts []SortField, reverse bool) (orderBy string, cols []string, keys []string) {
	var parts []string

	for _, s := range sorts {
		dir := "ASC"
		actualDesc := s.IsDesc

		if reverse {
			actualDesc = !s.IsDesc
		}

		if actualDesc {
			dir = "DESC"
		}

		parts = append(parts, fmt.Sprintf("%s %s", s.Column, dir))
		cols = append(cols, s.Column)
		keys = append(keys, s.Key)
	}

	return " ORDER BY " + strings.Join(parts, ", "), cols, keys
}

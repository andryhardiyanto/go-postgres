package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

func GetCursorTokenFromEncoded(req *RequestGetCursorTokenFromEncoded) (*ResponseGetCursorTokenFromEncoded, error) {
	if req.Token == "" {
		orderBy := buildOrderBy(req.DefaultOrder, false)

		if req.IsCustomCTE {
			req.Query = fmt.Sprintf(req.Query, orderBy)
		} else {
			req.Query += orderBy
		}

		return &ResponseGetCursorTokenFromEncoded{
			IsPrev:        false,
			Query:         req.Query,
			ReverseResult: false,
			Kv:            req.Kv,
		}, nil
	}

	token, err := DecodeCursorToken(req.Token)
	if err != nil {
		return nil, err
	}

	op, _ := token["operator"].(string)

	isPrev := false
	if v, ok := token["is_prev"]; ok {
		if b, ok2 := v.(bool); ok2 {
			isPrev = b
		}
	} else {
		isPrev = op == "<"
	}

	condition := BuildCursorCondition(req.CursorColumns, req.Keys, op)

	for _, key := range req.Keys {
		if val, ok := token[key]; ok {
			req.Kv = append(req.Kv, key, val)
		}
	}

	reverse := isPrev
	orderBy := buildOrderBy(req.DefaultOrder, reverse)

	if req.IsCustomCTE {
		condition = " AND " + condition
		req.Query = fmt.Sprintf(req.Query, condition, orderBy)
	} else {
		req.Query += " AND " + condition
		req.Query += orderBy
	}

	return &ResponseGetCursorTokenFromEncoded{
		IsPrev:        isPrev,
		Query:         req.Query,
		ReverseResult: reverse,
		CursorToken:   token,
		Kv:            req.Kv,
	}, nil
}

func EncodeCursorToken(token any) (string, error) {
	bytes, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(bytes), nil
}

func DecodeCursorToken(encoded string) (CursorToken, error) {
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

func BuildCursorCondition(columns []string, keys []string, operator string) string {
	var colParams []string
	for range keys {
		colParams = append(colParams, ":"+keys[len(colParams)])
	}
	return "(" + strings.Join(columns, ", ") + ") " + operator + " (" + strings.Join(colParams, ", ") + ")"
}

func reverseOrder(order string) string {
	order = strings.TrimSpace(order)
	upper := strings.ToUpper(order)
	if strings.HasSuffix(upper, "ASC") {
		return strings.Replace(order, "ASC", "DESC", 1)
	}
	if strings.HasSuffix(upper, "DESC") {
		return strings.Replace(order, "DESC", "ASC", 1)
	}
	return order
}

func buildOrderBy(order string, reverse bool) string {
	if strings.TrimSpace(order) == "" {
		return ""
	}
	if reverse {
		return " ORDER BY " + reverseOrder(order)
	}
	return " ORDER BY " + order
}

func isDescOrder(order string) bool {
	if order == "" {
		return false
	}
	upper := strings.ToUpper(order)
	idxAsc := strings.Index(upper, "ASC")
	idxDesc := strings.Index(upper, "DESC")

	switch {
	case idxAsc == -1 && idxDesc == -1:
		return false
	case idxAsc == -1:
		return true
	case idxDesc == -1:
		return false
	default:
		return idxDesc < idxAsc
	}
}

// nextOp untuk "halaman setelah", prevOp untuk "halaman sebelum"
func cursorOperators(order string) (nextOp, prevOp string) {
	if isDescOrder(order) {
		return "<", ">"
	}
	return ">", "<"
}

func makeCursorFromItem(item interface{}, keys []string, isPrev bool, operator string, totalItems int64) (string, error) {
	m := make(map[string]interface{})
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for _, key := range keys {
		f, ok := findFieldByTagOrName(v, key)
		if !ok || !f.IsValid() {
			return "", fmt.Errorf("field %s not found", key)
		}
		m[key] = f.Interface()
	}
	m["operator"] = operator
	m["is_prev"] = isPrev
	m["total_items"] = totalItems
	return EncodeCursorToken(m)
}

func ReverseSlice[T any](items []T) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

func firstTagName(tagVal string) string {
	if tagVal == "" || tagVal == "-" {
		return ""
	}
	if i := strings.IndexByte(tagVal, ','); i >= 0 {
		return tagVal[:i]
	}
	return tagVal
}

func findFieldByTagOrName(v reflect.Value, key string) (reflect.Value, bool) {
	v = reflect.Indirect(v)
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		sf := t.Field(i)

		if sf.PkgPath != "" {
			continue
		}

		if jsonName := firstTagName(sf.Tag.Get("json")); jsonName != "" &&
			strings.EqualFold(jsonName, key) {
			return v.Field(i), true
		}

		if dbName := firstTagName(sf.Tag.Get("db")); dbName != "" &&
			strings.EqualFold(dbName, key) {
			return v.Field(i), true
		}

		if strings.EqualFold(sf.Name, key) {
			return v.Field(i), true
		}

		if sf.Anonymous {
			sub := v.Field(i)
			if sub.Kind() == reflect.Ptr && sub.IsNil() {
				continue
			}
			if f, ok := findFieldByTagOrName(sub, key); ok {
				return f, true
			}
		}
	}
	return reflect.Value{}, false
}

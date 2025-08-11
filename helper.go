package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/csv"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type (
	StringSlice []string
)

var (
	quoteEscapeRegex = regexp.MustCompile(`([^\\]([\\]{2})*)\\"`)
)

const (
	qInsert = "insert"
	qUpdate = "update"
	qDelete = "delete"
	qSelect = "select"

	qResult = "q-result---"
)

func debugQuery(query string, args map[string]any, ctx context.Context) {
	replacer := strings.NewReplacer("\n", " ", "\t", " ")

	// Replace parameters in query
	for k, v := range args {
		query = strings.ReplaceAll(query, ":"+k, fmt.Sprintf("'%v'", v))
	}

	// Apply replacer to clean up the query
	cleanQuery := replacer.Replace(query)
	log.Printf("SQL Query: %s", cleanQuery)
}

func queryType(query string) string {
	query = strings.TrimSpace(query)
	if len(query) < 6 {
		return qSelect // Default to select for short queries
	}

	queryPrefix := strings.ToLower(query[:6])

	switch {
	case strings.HasPrefix(queryPrefix, qInsert):
		return qInsert
	case strings.HasPrefix(queryPrefix, qUpdate):
		return qUpdate
	case strings.HasPrefix(queryPrefix, qDelete):
		return qDelete
	default:
		return qSelect
	}
}

// insertTx inserts data into the database using a transaction
// and returns the inserted ID
func insertTx(ctx context.Context, tx *sqlx.Tx, query string, arg map[string]any) (any, error) {
	var insertedID any
	prep, err := tx.PrepareNamedContext(ctx, query)
	if err != nil {
		return 0, err
	}
	defer prep.Close()

	err = prep.GetContext(ctx, &insertedID, arg)
	if err != nil {
		return "", err
	}
	if insertedID == nil {
		return "", errors.WithStack(errors.New("error data insertedId 0"))
	}
	return insertedID, nil
}

// updateTx updates data in the database using a transaction
func updateTx(ctx context.Context, tx *sqlx.Tx, query string, arg map[string]any) error {
	prep, err := tx.PrepareNamedContext(ctx, query)
	if err != nil {
		return errors.WithStack(err)
	}
	defer prep.Close()

	res, err := prep.ExecContext(ctx, arg)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = res.RowsAffected()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return errors.WithStack(err)
	}

	return nil
}

// deleteTx deletes data from the database using a transaction
func deleteTx(ctx context.Context, tx *sqlx.Tx, query string, arg map[string]any) error {
	prep, err := tx.PrepareNamedContext(ctx, query)
	if err != nil {
		return errors.WithStack(err)
	}
	defer prep.Close()

	res, err := prep.ExecContext(ctx, arg)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = res.RowsAffected()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return errors.WithStack(err)
	}

	return nil
}

// insert inserts data into the database
// and returns the inserted ID
func insert(ctx context.Context, db *sqlx.DB, query string, arg map[string]any) (any, error) {
	var insertedID any
	prep, err := db.PrepareNamedContext(ctx, query)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	defer prep.Close()

	err = prep.GetContext(ctx, &insertedID, arg)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	if insertedID == nil {
		return 0, errors.WithStack(errors.New("error data insertedId 0"))
	}

	return insertedID, err
}

// update updates data in the database
func update(ctx context.Context, db *sqlx.DB, query string, arg map[string]any) error {
	prep, err := db.PrepareNamedContext(ctx, query)
	if err != nil {
		return errors.WithStack(err)
	}
	defer prep.Close()

	res, err := prep.ExecContext(ctx, arg)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = res.RowsAffected()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return errors.WithStack(err)
	}

	return nil
}

// delete deletes data from the database
func delete(ctx context.Context, db *sqlx.DB, query string, arg map[string]any) error {
	prep, err := db.PrepareNamedContext(ctx, query)
	if err != nil {
		return errors.WithStack(err)
	}
	defer prep.Close()

	res, err := prep.ExecContext(ctx, arg)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = res.RowsAffected()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return errors.WithStack(err)
	}

	return nil
}

func (s *StringSlice) Scan(src any) error {
	var str string
	switch src := src.(type) {
	case []byte:
		str = string(src)
	case string:
		str = src
	case nil:
		*s = nil
		return nil
	}

	str = quoteEscapeRegex.ReplaceAllString(str, `$1""`)
	str = strings.ReplaceAll(str, `\\`, `\`)

	str = str[1 : len(str)-1]

	if len(str) == 0 {
		*s = []string{}
		return nil
	}

	csvReader := csv.NewReader(strings.NewReader(str))
	slice, err := csvReader.Read()
	if err != nil {
		return err
	}
	*s = slice

	return nil
}

func (s StringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}

	var buffer bytes.Buffer

	buffer.WriteString("{")
	last := len(s) - 1
	for i, val := range s {
		buffer.WriteString(strconv.Quote(val))
		if i != last {
			buffer.WriteString(",")
		}
	}
	buffer.WriteString("}")

	return buffer.String(), nil
}

// Pairs converts a slice of key-value pairs to a map.
func Pairs(kv []any) (map[string]any, error) {
	if len(kv)%2 == 1 {
		return nil, fmt.Errorf("kv got the odd number of input pairs %d", len(kv))
	}
	arg := map[string]any{}
	for i := 0; i < len(kv); i += 2 {
		key := fmt.Sprintf("%v", kv[i])
		arg[key] = kv[i+1]
	}
	return arg, nil
}

// PairsHook converts a slice of key-value pairs to a map.
// If the value is a string and starts with the hook, it will be replaced with the value from the ids map.
func PairsHook(kv []any, ids map[string]any, hook string) (map[string]any, error) {
	if len(kv)%2 == 1 {
		return nil, fmt.Errorf("kv got the odd number of input pairs %d", len(kv))
	}
	arg := map[string]any{}
	for i := 0; i < len(kv); i += 2 {
		key := fmt.Sprintf("%v", kv[i])
		val := kv[i+1]
		s, ok := val.(string)
		if ok && len(s) > 11 && s[:11] == hook {
			val = ids[s[11:]]
		}
		arg[key] = val
	}
	return arg, nil
}

// Filter filters the slice of strings based on the map.
func Filter(sl []string, m map[string]string) (result []string) {
	for _, v := range sl {
		_, ok := m[v]
		if !ok {
			result = append(result, v)
		}
	}
	return
}

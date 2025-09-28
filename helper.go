package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/csv"
	"fmt"
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

func debugQuery(query string, arguments map[string]any) {
	// Replace parameters in query
	finalQuery := query
	for key, value := range arguments {
		finalQuery = strings.ReplaceAll(finalQuery, ":"+key, fmt.Sprintf("'%v'", value))
	}

	fmt.Println("[DEBUG SQL]", finalQuery)
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
func insertTx(ctx context.Context, transaction *sqlx.Tx, query string, arguments map[string]any) (any, error) {
	var insertedID any
	preparedStatement, err := transaction.PrepareNamedContext(ctx, query)
	if err != nil {
		return 0, err
	}
	defer preparedStatement.Close()

	err = preparedStatement.GetContext(ctx, &insertedID, arguments)
	if err != nil {
		return "", err
	}
	if insertedID == nil {
		return "", errors.WithStack(errors.New("insert operation failed: no ID was returned from the database. This may indicate that the table does not have an auto-increment primary key or the insert did not complete successfully"))
	}
	return insertedID, nil
}

// updateTx updates data in the database using a transaction
func updateTx(ctx context.Context, transaction *sqlx.Tx, query string, arguments map[string]any) (int64, error) {
	preparedStatement, err := transaction.PrepareNamedContext(ctx, query)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	defer preparedStatement.Close()

	result, err := preparedStatement.ExecContext(ctx, arguments)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, errors.WithStack(err)
	}

	return rowsAffected, nil
}

// deleteTx deletes data from the database using a transaction
func deleteTx(ctx context.Context, transaction *sqlx.Tx, query string, arguments map[string]any) (int64, error) {
	preparedStatement, err := transaction.PrepareNamedContext(ctx, query)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	defer preparedStatement.Close()

	result, err := preparedStatement.ExecContext(ctx, arguments)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, errors.WithStack(err)
	}

	return rowsAffected, nil
}

// insert inserts data into the database
// and returns the inserted ID
func insert(ctx context.Context, database *sqlx.DB, query string, arguments map[string]any) (any, error) {
	var insertedID any
	preparedStatement, err := database.PrepareNamedContext(ctx, query)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	defer preparedStatement.Close()

	err = preparedStatement.GetContext(ctx, &insertedID, arguments)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	if insertedID == nil {
		return 0, errors.WithStack(errors.New("insert operation failed: no ID was returned from the database. This may indicate that the table does not have an auto-increment primary key or the insert did not complete successfully"))
	}

	return insertedID, err
}

// update updates data in the database
func update(ctx context.Context, database *sqlx.DB, query string, arguments map[string]any) (int64, error) {
	preparedStatement, err := database.PrepareNamedContext(ctx, query)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	defer preparedStatement.Close()

	result, err := preparedStatement.ExecContext(ctx, arguments)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, errors.WithStack(err)
	}

	return rowsAffected, nil
}

// delete deletes data from the database
func delete(ctx context.Context, database *sqlx.DB, query string, arguments map[string]any) (int64, error) {
	preparedStatement, err := database.PrepareNamedContext(ctx, query)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	defer preparedStatement.Close()

	result, err := preparedStatement.ExecContext(ctx, arguments)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, errors.WithStack(err)
	}

	return rowsAffected, nil
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
func Pairs(keyValuePairs []any) (map[string]any, error) {
	if len(keyValuePairs)%2 == 1 {
		return nil, fmt.Errorf("invalid key-value pairs: expected even number of arguments but got %d. Key-value pairs must be provided in pairs like: key1, value1, key2, value2", len(keyValuePairs))
	}
	arguments := map[string]any{}
	for i := 0; i < len(keyValuePairs); i += 2 {
		key := fmt.Sprintf("%v", keyValuePairs[i])
		arguments[key] = keyValuePairs[i+1]
	}
	return arguments, nil
}

// PairsHook converts a slice of key-value pairs to a map.
// If the value is a string and starts with the hook, it will be replaced with the value from the ids map.
func PairsHook(keyValuePairs []any, identifiers map[string]any, hook string) (map[string]any, error) {
	if len(keyValuePairs)%2 == 1 {
		return nil, fmt.Errorf("invalid key-value pairs: expected even number of arguments but got %d. Key-value pairs must be provided in pairs like: key1, value1, key2, value2", len(keyValuePairs))
	}
	arguments := map[string]any{}
	for i := 0; i < len(keyValuePairs); i += 2 {
		key := fmt.Sprintf("%v", keyValuePairs[i])
		value := keyValuePairs[i+1]
		stringValue, ok := value.(string)
		if ok && len(stringValue) > 11 && stringValue[:11] == hook {
			value = identifiers[stringValue[11:]]
		}
		arguments[key] = value
	}
	return arguments, nil
}

// Filter filters the slice of strings based on the map.
func Filter(slice []string, filterMap map[string]string) (result []string) {
	for _, value := range slice {
		_, exists := filterMap[value]
		if !exists {
			result = append(result, value)
		}
	}
	return
}

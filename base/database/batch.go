package database

// This file was adapted from https://github.com/bombsimon/gorm-bulk
import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// We no longer need BulkInsertChunk as GORMv2 supports batch insert by default
// However we still need to use BulkInsert when using UPSERT

// bulkNow holds a global now state and will be used for each records as
// CreatedAt and UpdatedAt value if they're empty. This value will be set to the
// value from gorm.NowFunc() in scopeFromObjects to ensure all objects get the
// same value.
// nolint: gochecknoglobals
var bulkNow time.Time

func UnnestInsert(db *gorm.DB, query string, objects interface{}) error {
	// transpose values from
	// rows := [{1 2} {3 4} {5 6}]
	// to columns := [[1 3 5] [2 4 6]]
	var column []interface{}
	objSlice := reflect.ValueOf(objects)
	if objSlice.Len() == 0 {
		return nil
	}
	for j := 0; j < objSlice.Len(); j++ {
		inSlice := objSlice.Index(j)
		for i := 0; i < inSlice.NumField(); i++ {
			if len(column) <= i {
				column = append(column, []interface{}{})
			}
			column[i] = append(column[i].([]interface{}), inSlice.Field(i).Interface())
		}
	}

	return db.Exec(query, column...).Error
}

func BulkInsert(db *gorm.DB, objects interface{}) error {
	if reflect.TypeOf(objects).Kind() != reflect.Slice {
		return errors.New("This method only works on slices")
	}
	// Reflect the objects array into generic value
	objectsVal := reflect.ValueOf(objects)

	return bulkExec(db, objectsVal)
}

// bulkExec will convert a slice of interfaces to bulk SQL statement.
func bulkExec(db *gorm.DB, objects reflect.Value) error {
	statement, err := statementFromObjects(db, objects)
	if err != nil {
		return err
	}

	// No scope and no error means nothing to do
	if statement == nil {
		return nil
	}

	rows, err := db.Raw(statement.SQL.String(), statement.Vars...).Rows()

	if err != nil {
		return err
	}
	defer rows.Close()

	for i := 0; i <= objects.Len() && rows.Next(); i++ {
		// Perform scan into the address of an element, intepreted as an interface
		err = db.ScanRows(rows, objects.Index(i).Addr().Interface())
		db.RowsAffected++
		if err != nil {
			return err
		}
	}
	return nil
}

func statementFromObjects(db *gorm.DB, objects reflect.Value) (*gorm.Statement, error) {
	if objects.Len() < 1 {
		return nil, nil
	}
	// Retrieve 0th element as an interface
	firstElem := objects.Index(0).Interface()
	schema, err := schema.Parse(firstElem, &sync.Map{}, db.NamingStrategy)
	if err != nil {
		return nil, err
	}

	statement := gorm.Statement{DB: db, Schema: schema, Clauses: db.Statement.Clauses}

	// Ensure we set the correct time and reset it after we're done.
	bulkNow = db.NowFunc()

	defer func() {
		bulkNow = time.Time{}
	}()
	// Get a map of the first element to calculate field names and number of
	// placeholders.
	firstObjectFields, err := objectToMap(db, firstElem)
	if err != nil {
		return nil, err
	}
	columnNames, placeholders := buildColumnsAndPlaceholders(firstObjectFields)

	// We must setup quotedColumnNames after sorting columnNames since sorting
	// of quoted fields might differ from sorting without. This way we know that
	// columnNames is the master of the order and will be used both when setting
	// field and values order.
	quotedColumnNames := make([]string, len(columnNames))
	tableName := statement.Schema.Table
	for i := range columnNames {
		quotedColumnNames[i] = statement.Quote(db.NamingStrategy.ColumnName(tableName, columnNames[i]))
	}
	groups := make([]string, objects.Len())
	for i := 0; i < objects.Len(); i++ {
		// Retrieve ith element as an interface
		r := objects.Index(i).Interface()
		row, err := objectToMap(db, r)
		if err != nil {
			return nil, err
		}
		for _, key := range columnNames {
			statement.Vars = append(statement.Vars, row[key])
		}

		groups[i] = fmt.Sprintf("(%s)", strings.Join(placeholders, ", "))
	}

	query := insertFunc(db, statement.Quote(tableName), quotedColumnNames, groups)
	statement.SQL.WriteString(query)
	return &statement, nil
}

func buildColumnsAndPlaceholders(firstObjectFields map[string]interface{}) ([]string, []string) {
	columnNames := make([]string, 0, len(firstObjectFields))
	placeholders := make([]string, 0, len(firstObjectFields))
	for k := range firstObjectFields {
		if k == "" {
			continue
		}
		// Add raw column names to use for iteration over each row later to get
		// the correct order of columns.
		columnNames = append(columnNames, k)

		// Add as many placeholders (question marks) as there are columns.
		placeholders = append(placeholders, "?")

		// Sort the column names to ensure the right order.
		sort.Strings(columnNames)
	}
	return columnNames, placeholders
}

func insertFunc(db *gorm.DB, quotedTableName string, columnNames, groups []string) string {
	var extraOptions string
	if clauseOnConflict, ok := db.Statement.Clauses["ON CONFLICT"]; ok {
		// Add the extra insert option
		extraOptions = parseClause(clauseOnConflict.Expression)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s %s RETURNING *",
		quotedTableName,
		strings.Join(columnNames, ", "),
		strings.Join(groups, ", "),
		extraOptions,
	)

	return query
}

func parseClause(clauseOnConflict clause.Expression) string {
	rvClause := reflect.ValueOf(clauseOnConflict)
	clauseValue := rvClause.Interface().(clause.OnConflict)
	if clauseValue.DoNothing {
		return "ON CONFLICT DO NOTHING"
	}
	keys := []string{}
	for _, column := range clauseValue.Columns {
		keys = append(keys, column.Name)
	}
	keyStrs := strings.Join(keys, ", ")

	updateStrs := []string{}
	for _, update := range clauseValue.DoUpdates {
		var val string
		if rvColumn := reflect.ValueOf(update.Value); rvColumn.Kind() == reflect.TypeOf(clause.Column{}).Kind() {
			column := rvColumn.Interface().(clause.Column)
			val = fmt.Sprintf("%s = %s.%s", update.Column.Name, column.Table, column.Name)
		} else {
			val = fmt.Sprintf("%s = %s", update.Column.Name, update.Value)
		}
		updateStrs = append(updateStrs, val)
	}
	valStr := strings.Join(updateStrs, ", ")

	SQLstring := fmt.Sprintf("ON CONFLICT (%v) DO UPDATE SET %v", keyStrs, valStr) // nolint:gosimple
	return SQLstring
}

// objectToMap takes any object of type <T> and returns a map with the gorm
// field DB name as key and the value as value
func objectToMap(db *gorm.DB, object interface{}) (map[string]interface{}, error) {
	attributes := make(map[string]interface{})
	now := bulkNow

	// De-reference pointers (and it's values)
	rv := reflect.ValueOf(object)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
		object = rv.Interface()
	}
	if rv.Kind() != reflect.Struct {
		return nil, errors.New("value must be kind of Struct")
	}

	if now.IsZero() {
		now = Db.NowFunc()
	}

	parsedSchema, _ := schema.Parse(object, &sync.Map{}, db.NamingStrategy)
	for _, field := range parsedSchema.Fields {
		// Exclude relational record because it's not directly contained in database columns
		fieldVaule := rv.FieldByName(field.Name).Interface()
		_, hasForeignKey := field.TagSettings["FOREIGNKEY"]
		if hasForeignKey {
			continue
		}

		if ok := field.TagSettings["-"]; ok != "" {
			continue
		}

		// Let the DBM set the default values since these might be meta values such as 'CURRENT_TIMESTAMP'. Has default
		// will be set to true also for 'AUTO_INCREMENT' fields which is not primary keys so we must check that we've
		// ACTUALLY configured a default value and uses the tag before we skip it.
		if field.HasDefaultValue && field != nil {
			if _, ok := field.TagSettings["DEFAULT"]; ok {
				continue
			}
		}

		// Skip blank primary key fields named ID. They're probably coming from
		// `gorm.Model` which doesn't have the AUTO_INCREMENT tag.
		_, isAutoInc := field.TagSettings["AUTOINCREMENT"]
		if field.DBName == "id" && field.PrimaryKey && (field != nil || isAutoInc) {
			continue
		}

		if field.StructField.Name == "CreatedAt" || field.StructField.Name == "UpdatedAt" {
			if field != nil {
				attributes[field.DBName] = now
				continue
			}
		}
		attributes[field.DBName] = fieldVaule
	}
	return attributes, nil
}

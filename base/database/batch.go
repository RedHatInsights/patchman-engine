package database

// This file was adapted from https://github.com/bombsimon/gorm-bulk
import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

// bulkNow holds a global now state and will be used for each records as
// CreatedAt and UpdatedAt value if they're empty. This value will be set to the
// value from gorm.NowFunc() in scopeFromObjects to ensure all objects get the
// same value.
// nolint: gochecknoglobals
var bulkNow time.Time

func BulkInsert(db *gorm.DB, objects interface{}) error {
	if reflect.TypeOf(objects).Kind() != reflect.Slice {
		return errors.New("This method only works on slices")
	}
	// Reflect the objects array into generic value
	objectsVal := reflect.ValueOf(objects)

	return bulkExec(db, objectsVal)
}

// BulkInsertChunk will split the objects passed into the passed chunk size. A
// slice of errors will be returned (if any).
func BulkInsertChunk(db *gorm.DB, objects interface{}, chunkSize int) error {
	var allErrors []error

	if reflect.TypeOf(objects).Kind() != reflect.Slice {
		return errors.New("This method only works on slices")
	}

	// Reflect the objects array into generic value
	objectsVal := reflect.ValueOf(objects)

	for {
		var chunkObjects reflect.Value

		if objectsVal.Len() <= chunkSize {
			chunkObjects = objectsVal
			objectsVal = reflect.ValueOf([]interface{}{})
		} else {
			chunkObjects = objectsVal.Slice(0, chunkSize)
			objectsVal = objectsVal.Slice(chunkSize, objectsVal.Len())
		}

		if err := bulkExec(db, chunkObjects); err != nil {
			allErrors = append(allErrors, err)
		}

		// Nothing more to do
		if objectsVal.Len() < 1 {
			break
		}
	}

	if len(allErrors) > 0 {
		return allErrors[0]
	}

	return nil
}

// bulkExec will convert a slice of interfaces to bulk SQL statement.
func bulkExec(db *gorm.DB, objects reflect.Value) error {
	scope, err := scopeFromObjects(db, objects)
	if err != nil {
		return err
	}

	// No scope and no error means nothing to do
	if scope == nil {
		return nil
	}

	rows, err := db.Raw(scope.SQL, scope.SQLVars...).Rows()

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

func scopeFromObjects(db *gorm.DB, objects reflect.Value) (*gorm.Scope, error) {
	if objects.Len() < 1 {
		return nil, nil
	}

	// Retrieve 0th element as an interface
	firstElem := objects.Index(0).Interface()

	var (
		scope = db.NewScope(firstElem)
	)

	// Ensure we set the correct time and reset it after we're done.
	bulkNow = gorm.NowFunc()

	defer func() {
		bulkNow = time.Time{}
	}()

	// Get a map of the first element to calculate field names and number of
	// placeholders.
	firstObjectFields, err := objectToMap(firstElem)
	if err != nil {
		return nil, err
	}

	columnNames, placeholders := buildColumnsAndPlaceholders(firstObjectFields)

	// We must setup quotedColumnNames after sorting columnNames since sorting
	// of quoted fields might differ from sorting without. This way we know that
	// columnNames is the master of the order and will be used both when setting
	// field and values order.
	quotedColumnNames := make([]string, len(columnNames))
	for i := range columnNames {
		quotedColumnNames[i] = scope.Quote(gorm.ToColumnName(columnNames[i]))
	}

	groups := make([]string, objects.Len())
	for i := 0; i < objects.Len(); i++ {
		// Retrieve ith element as an interface
		r := objects.Index(i).Interface()

		row, err := objectToMap(r)
		if err != nil {
			return nil, err
		}

		for _, key := range columnNames {
			scope.AddToVars(row[key])
		}

		groups[i] = fmt.Sprintf("(%s)", strings.Join(placeholders, ", "))
	}

	insertFunc(scope, quotedColumnNames, groups)

	return scope, nil
}

func buildColumnsAndPlaceholders(firstObjectFields map[string]interface{}) ([]string, []string) {
	columnNames := make([]string, 0, len(firstObjectFields))
	placeholders := make([]string, 0, len(firstObjectFields))
	for k := range firstObjectFields {
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

func insertFunc(scope *gorm.Scope, columnNames, groups []string) {
	var (
		extraOptions string
	)

	if insertOption, ok := scope.Get("gorm:insert_option"); ok {
		// Add the extra insert option
		extraOptions = fmt.Sprintf(" %s", insertOption)
	}

	// nolint: gosec
	scope.Raw(fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s %s RETURNING *",
		scope.QuotedTableName(),
		strings.Join(columnNames, ", "),
		strings.Join(groups, ", "),
		extraOptions,
	))
}

// objectToMap takes any object of type <T> and returns a map with the gorm
// field DB name as key and the value as value
func objectToMap(object interface{}) (map[string]interface{}, error) {
	var attributes = map[string]interface{}{}
	var now = bulkNow

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
		now = gorm.NowFunc()
	}

	for _, field := range (&gorm.Scope{Value: object}).Fields() {
		// Exclude relational record because it's not directly contained in database columns
		_, hasForeignKey := field.TagSettingsGet("FOREIGNKEY")
		if hasForeignKey {
			continue
		}

		if field.StructField.Relationship != nil {
			continue
		}

		if field.IsIgnored {
			continue
		}

		// Let the DBM set the default values since these might be meta values such as 'CURRENT_TIMESTAMP'. Has default
		// will be set to true also for 'AUTO_INCREMENT' fields which is not primary keys so we must check that we've
		// ACTUALLY configured a default value and uses the tag before we skip it.
		if field.StructField.HasDefaultValue && field.IsBlank {
			if _, ok := field.TagSettingsGet("DEFAULT"); ok {
				continue
			}
		}

		// Skip blank primary key fields named ID. They're probably coming from
		// `gorm.Model` which doesn't have the AUTO_INCREMENT tag.
		if field.DBName == "id" && field.IsPrimaryKey && field.IsBlank {
			continue
		}

		if field.Struct.Name == "CreatedAt" || field.Struct.Name == "UpdatedAt" {
			if field.IsBlank {
				attributes[field.DBName] = now
				continue
			}
		}

		attributes[field.DBName] = field.Field.Interface()
	}

	return attributes, nil
}

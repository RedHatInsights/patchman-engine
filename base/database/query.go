package database

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"reflect"
	"regexp"
	"strings"
)

type AttrName = string
type AttrQuery = string

// Used to store field name => sql query mapping
type AttrMap = map[AttrName]AttrQuery

var ColumnNameRe = regexp.MustCompile(`column:([\w_]+)`)

func MustGetSelect(t interface{}) string {
	// We must get fields ordered, so we assemble them in proper order for gorm select
	attrs, names, err := GetQueryAttrs(t)
	if err != nil {
		panic(err)
	}
	fields := make([]string, 0, len(names))
	for _, n := range names {
		fields = append(fields, fmt.Sprintf("%v as %v", attrs[n], n))
	}
	return strings.Join(fields, ", ")
}

func getQueryFromTags(v reflect.Type) (AttrMap, []AttrName, error) {
	if v.Kind() != reflect.Struct {
		return nil, nil, errors.New("Only struct kind is supported")
	}
	res := AttrMap{}
	var resNames []AttrName
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		// Parse from nested struct fields
		if field.Type.Kind() == reflect.Struct && field.Name == field.Type.Name() {
			nested, names, err := getQueryFromTags(field.Type)
			if err != nil {
				return nil, nil, err
			}
			resNames = append(resNames, names...)
			for k, v := range nested {
				res[k] = v
			}
		} else {
			columnName := gorm.ToColumnName(field.Name)
			if expr, has := field.Tag.Lookup("gorm"); has {
				match := ColumnNameRe.FindStringSubmatch(expr)
				if len(match) > 0 {
					columnName = match[1]
				}
			}

			if expr, has := field.Tag.Lookup("query"); has {
				res[columnName] = expr
			} else {
				// If we dont have expr, we just use raw column name
				res[columnName] = columnName
			}
			// Result HAS to contain all columns, because gorm loads by index, not by name
			resNames = append(resNames, columnName)
		}
	}
	return res, resNames, nil
}

func GetQueryAttrs(s interface{}) (AttrMap, []string, error) {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Struct {
		return getQueryFromTags(v.Type())
	} else if v.Kind() == reflect.Ptr || v.Kind() == reflect.Slice {
		return getQueryFromTags(v.Type().Elem())
	}
	return nil, nil, errors.New("Invalid type")
}

func MustGetQueryAttrs(s interface{}) AttrMap {
	res, _, err := GetQueryAttrs(s)
	if err != nil {
		panic(err)
	}
	return res
}

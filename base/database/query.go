package database

import (
	"app/base"
	"app/base/utils"
	"fmt"
	"github.com/pkg/errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type AttrParser func(string) (interface{}, error)

type AttrName = string
type AttrInfo struct {
	DataQuery  string
	OrderQuery string
	Parser     AttrParser
}

// Used to store field name => sql query mapping
type AttrMap = map[AttrName]AttrInfo

var ColumnNameRe = regexp.MustCompile(`column:([\w_]+)`)

func MustGetSelect(t interface{}) string {
	// We must get fields ordered, so we assemble them in proper order for gorm select
	attrs, names, err := GetQueryAttrs(t)
	if err != nil {
		panic(err)
	}
	fields := make([]string, 0, len(names))
	for _, n := range names {
		fields = append(fields, fmt.Sprintf("%v as %v", attrs[n].DataQuery, n))
	}
	return strings.Join(fields, ", ")
}

func parserForType(v reflect.Type) (AttrParser, error) {
	switch v.Kind() {
	case reflect.String:
		return func(s string) (i interface{}, err error) {
			return base.RemoveInvalidChars(s), nil
		}, nil
	case reflect.Bool:
		return func(s string) (i interface{}, err error) {
			return strconv.ParseBool(s)
		}, nil
	case reflect.Int64:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int:
		return func(s string) (i interface{}, err error) {
			return strconv.Atoi(s)
		}, nil
	case reflect.Ptr:
		return parserForType(v.Elem())
	case reflect.Struct:
		if reflect.TypeOf(time.Time{}).AssignableTo(v) {
			return func(s string) (i interface{}, err error) {
				return time.Parse(time.RFC3339, s)
			}, nil
		}
		fallthrough
	default:
		utils.Log("attribute", v.Name()).Debug("No query parser found")
		return nil, nil
	}
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
			columnName := field.Name
			if expr, has := field.Tag.Lookup("gorm"); has {
				match := ColumnNameRe.FindStringSubmatch(expr)
				if len(match) > 0 {
					columnName = match[1]
				}
			}

			parser, err := parserForType(field.Type)
			if err != nil {
				return nil, nil, err
			}
			if expr, has := field.Tag.Lookup("query"); has {
				res[columnName] = AttrInfo{
					DataQuery: expr,
					Parser:    parser,
				}
			} else {
				// If we dont have expr, we just use raw column name
				res[columnName] = AttrInfo{
					DataQuery: columnName,
					Parser:    parser,
				}
			}

			info := res[columnName]
			if expr, has := field.Tag.Lookup("order_query"); has {
				info.OrderQuery = expr
			} else {
				info.OrderQuery = info.DataQuery
			}
			res[columnName] = info

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

package templatedb

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/tianxinzizhen/templatedb/template"

	"github.com/tianxinzizhen/templatedb/util"
)

var sqlfunc template.FuncMap = make(template.FuncMap)

var SqlEscapeBytesBackslash = false

func comma(iVal reflect.Value) (string, error) {
	i := util.RefValue(iVal)
	var commaPrint bool
	switch i.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		commaPrint = i.Int() > 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		commaPrint = i.Uint() > 0
	default:
		return "", nil
	}
	if commaPrint {
		return ",", nil
	} else {
		return "", nil
	}
}
func inParam(list reflect.Value, fieldNames ...any) (string, []any, error) {
	list = util.RefValue(list)
	fieldName := fmt.Sprint(fieldNames...)
	if list.Kind() == reflect.Slice || list.Kind() == reflect.Array {
		sb := strings.Builder{}
		sb.WriteString("in (")
		var args []any = make([]any, list.Len())
		for i := 0; i < list.Len(); i++ {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteByte('?')
			item := util.RefValue(list.Index(i))
			if !item.IsValid() {
				continue
			}
			switch item.Kind() {
			case reflect.Struct:
				tField, ok := template.GetFieldByName(item.Type(), fieldName)
				if ok {
					field, err := item.FieldByIndexErr(tField.Index)
					if err != nil {
						return "", nil, err
					}
					args[i] = field.Interface()
				} else {
					return "", nil, fmt.Errorf("in params : The attribute %s was not found in the structure %s.%s", fieldName, item.Type().PkgPath(), item.Type().Name())
				}
			case reflect.Map:
				if item.Type().Key().Kind() == reflect.String {
					fieldValue := item.MapIndex(reflect.ValueOf(fieldName))
					if fieldValue.IsValid() {
						args[i] = fieldValue.Interface()
					} else {
						return "", nil, fmt.Errorf("in params : fieldValue in map[%s] IsValid", fieldName)
					}
				} else {
					return "", nil, fmt.Errorf("in params : Map key Type is not string")
				}
			default:
				args[i] = item.Interface()
			}
		}
		sb.WriteString(")")
		return sb.String(), args, nil
	} else {
		return "", nil, errors.New("in params : variables are not arrays or slices")
	}
}
func params(list ...reflect.Value) (string, []any) {
	sb := strings.Builder{}
	var args []any = make([]any, len(list))
	for i, v := range list {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteByte('?')
		args[i] = v.Interface()
	}
	return sb.String(), args
}

func like(param reflect.Value) (string, []any) {
	var args []any = make([]any, 1)
	p := fmt.Sprint(param)
	lb := strings.Builder{}
	if !strings.HasPrefix(p, "%") {
		lb.WriteByte('%')
	}
	lb.WriteString(p)
	if !strings.HasSuffix(p, "%") {
		lb.WriteByte('%')
	}
	args[0] = lb.String()
	return " like ? ", args
}

func sqlescape(list ...reflect.Value) (string, error) {
	sb := strings.Builder{}
	for i, v := range list {
		if i > 0 {
			sb.WriteByte(',')
		}
		param, err := util.GetNoneEscapeSql(v.Interface(), SqlEscapeBytesBackslash)
		if err != nil {
			return "", err
		}
		sb.WriteString(param)
	}
	return sb.String(), nil
}

func orNull(param reflect.Value) (string, []any) {
	var args []any = make([]any, 1)
	isTure, _ := template.IsTrue(param)
	if isTure {
		args[0] = param.Interface()
	} else {
		args[0] = nil
	}
	return "?", args
}

func init() {
	LoadFunc("comma", comma)
	LoadFunc("in", inParam)
	LoadFunc("like", like)
	LoadFunc("param", params)
	LoadFunc("sqlescape", sqlescape)
	LoadFunc("orNull", orNull)
}

func LoadFunc(key string, funcMethod any) {
	sqlfunc[key] = funcMethod
}

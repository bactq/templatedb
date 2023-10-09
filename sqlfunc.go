package templatedb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/tianxinzizhen/templatedb/template"

	"github.com/tianxinzizhen/templatedb/util"
)

var sqlFunc template.FuncMap = make(template.FuncMap)

var SqlEscapeBytesBackslash = false

func comma(iVal reflect.Value) (string, error) {
	i, isNil := util.Indirect(iVal)
	if isNil {
		return "", fmt.Errorf("comma sql function in paramter is nil")
	}
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

func likeRight(param reflect.Value) (string, []any) {
	var args []any = make([]any, 1)
	p := fmt.Sprint(param)
	lb := strings.Builder{}
	lb.WriteString(p)
	if !strings.HasSuffix(p, "%") {
		lb.WriteByte('%')
	}
	args[0] = lb.String()
	return " like ? ", args
}
func likeLeft(param reflect.Value) (string, []any) {
	var args []any = make([]any, 1)
	p := fmt.Sprint(param)
	lb := strings.Builder{}
	if !strings.HasPrefix(p, "%") {
		lb.WriteByte('%')
	}
	lb.WriteString(p)
	args[0] = lb.String()
	return " like ? ", args
}

func marshal(list ...reflect.Value) (string, []any, error) {
	sb := strings.Builder{}
	var args []any = make([]any, len(list))
	for i, v := range list {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteByte('?')
		vi := v.Interface()
		isTrue, _ := template.IsTrue(vi)
		if isTrue {
			mJson, err := json.Marshal(vi)
			if err != nil {
				return "", nil, err
			}
			args[i] = string(mJson)
		}
	}
	return sb.String(), args, nil
}

func SqlEscape(arg any) (sql string, err error) {
	return util.GetNoneEscapeSql(arg, SqlEscapeBytesBackslash)
}

func SqlInterpolateParams(query string, arg []any) (sql string, err error) {
	return util.InterpolateParams(query, arg, SqlEscapeBytesBackslash)
}

func jsonTagAsFieldName(tag reflect.StructTag, fieldName string) bool {
	if asName, ok := tag.Lookup("json"); ok {
		if asName == "-" {
			return false
		}
		fName, _, _ := strings.Cut(asName, ",")
		if fieldName == fName {
			return true
		}
	}
	if asName, ok := tag.Lookup("as"); ok {
		if fieldName == asName {
			return true
		}
	}
	return false
}

func getFieldByTag(t reflect.Type, fieldName string, scanNum map[string]int) (f reflect.StructField, ok bool) {
	for i := 0; i < t.NumField(); i++ {
		tf := t.Field(i)
		if jsonTagAsFieldName(tf.Tag, fieldName) {
			return tf, true
		}
		if tf.Anonymous && tf.Type.Kind() == reflect.Struct {
			f, ok = getFieldByTag(tf.Type, fieldName, scanNum)
			if ok {
				if scanNum != nil {
					if _, ok := scanNum[f.Name]; ok {
						if i <= scanNum[f.Name] {
							continue
						} else {
							scanNum[f.Name] = i
						}
					} else {
						scanNum[f.Name] = i
					}
				}
				f.Index = append(tf.Index, f.Index...)
				return
			}
		}
	}
	return
}
func DefaultGetFieldByName(t reflect.Type, fieldName string, scanNum map[string]int) (f reflect.StructField, ok bool) {
	tField, ok := t.FieldByName(fieldName)
	if ok {
		return tField, ok
	}
	f, ok = getFieldByTag(t, fieldName, scanNum)
	return
}

func init() {
	//sql 函数的加载
	AddTemplateFunc("comma", comma)
	AddTemplateFunc("like", like)
	AddTemplateFunc("liker", likeRight)
	AddTemplateFunc("likel", likeLeft)
	AddTemplateFunc("param", params)
	AddTemplateFunc("marshal", marshal)
	AddTemplateFunc("json", marshal)
	//模版@#号字符串拼接时对字段值转化成sql字符串函数
	template.SqlEscape = SqlEscape
}

func AddTemplateFunc(key string, funcMethod any) error {
	if _, ok := sqlFunc[key]; ok {
		return fmt.Errorf("add template func[%s] already exists ", key)
	} else {
		sqlFunc[key] = funcMethod
	}
	return nil
}

type TemplateDBContextType int

const (
	TemplateDBKeyString TemplateDBContextType = iota
	TemplateDBFuncName
)

var LogPrintf func(ctx context.Context, info string)

var MaxStackLen = 50

type sqlDB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

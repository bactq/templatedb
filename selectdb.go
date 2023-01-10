package templatedb

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/tianxinzizhen/templatedb/template"
	"github.com/tianxinzizhen/templatedb/util"
)

type AnyDB interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

type SelectDB[T any] struct {
	db       *DefaultDB
	selectdb AnyDB
}

func newScanDest(columns []*sql.ColumnType, t reflect.Type) map[int][]int {
	indexMap := make(map[int][]int, len(columns))
	for i, item := range columns {
		if t.Kind() == reflect.Struct {
			f, ok := template.GetFieldByName(t, item.Name())
			if ok {
				indexMap[i] = f.Index
			} else {
				panic(fmt.Sprintf("类型%v无法扫描字段：%s", t, item.Name()))
			}
		}
	}
	return indexMap
}

func DBSelect[T any](db any) *SelectDB[T] {
	if db, ok := db.(*DefaultDB); ok {
		return &SelectDB[T]{db: db, selectdb: db.sqlDB}
	}
	if db, ok := db.(*TemplateTxDB); ok {
		return &SelectDB[T]{db: db.db, selectdb: db.tx}
	}
	return nil
}

func (sdb *SelectDB[T]) newReceiver(columns []*sql.ColumnType, scanIndex map[int][]int) (*T, []any) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	destSlice := make([]any, 0, len(columns))
	if t.Kind() == reflect.Struct {
		dest := new(T)
		v := reflect.ValueOf(dest).Elem()
		for si := range columns {
			destSlice = append(destSlice, &util.StructScaner{Dest: v, Index: scanIndex[si]})
		}
		return dest, destSlice
	} else if t.Kind() == reflect.Map && t.Key().Kind() == reflect.String {
		var ret *T = new(T)
		dest := reflect.MakeMapWithSize(reflect.MapOf(t.Key(), t.Elem()), len(columns))
		for _, v := range columns {
			destSlice = append(destSlice, &util.MapScaner{Dest: dest, Name: v.Name()})
		}
		*ret = dest.Interface().(T)
		return ret, destSlice
	} else if t.Kind() == reflect.Slice {
		var ret *T = new(T)
		dest := reflect.MakeSlice(reflect.SliceOf(t.Elem()), len(columns), len(columns))
		for i := range columns {
			destSlice = append(destSlice, &util.SliceScaner{Dest: dest, Index: i})
		}
		*ret = dest.Interface().(T)
		return ret, destSlice
	} else {
		dest := new(T)
		return dest, []any{dest}
	}
}

func (sdb *SelectDB[T]) query(params any, name []any) (*sql.Rows, []*sql.ColumnType, error) {
	statement := getSkipFuncName(3, name)
	sql, args, err := sdb.db.templateBuild(statement, params)
	if err != nil {
		return nil, nil, err
	}
	rows, err := sdb.selectdb.Query(sql, args...)
	if err != nil {
		return nil, nil, err
	}
	columns, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, err
	}
	return rows, columns, nil
}

func (sdb *SelectDB[T]) Select(params any, name ...any) []*T {
	rows, columns, err := sdb.query(params, name)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	scanIndex := newScanDest(columns, reflect.TypeOf((*T)(nil)).Elem())
	ret := *(new([]*T))
	for rows.Next() {
		receiver, destSlice := sdb.newReceiver(columns, scanIndex)
		err = rows.Scan(destSlice...)
		if err != nil {
			panic(err)
		}
		ret = append(ret, receiver)
	}
	return ret
}

func (sdb *SelectDB[T]) SelectFirst(params any, name ...any) *T {
	rows, columns, err := sdb.query(params, name)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	scanIndex := newScanDest(columns, reflect.TypeOf((*T)(nil)))
	if rows.Next() {
		receiver, destSlice := sdb.newReceiver(columns, scanIndex)
		err = rows.Scan(destSlice...)
		if err != nil {
			panic(err)
		}
		return receiver
	} else {
		return nil
	}
}

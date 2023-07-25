package templatedb

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/tianxinzizhen/templatedb/template"
)

type DBFuncTemplateDB struct {
	db                    *sql.DB
	leftDelim, rightDelim string
	sqlParamsConvert      func(val reflect.Value) (string, any)
	sqlDebug              bool
	logFunc               func(ctx context.Context, info string)
	sqlParamType          map[reflect.Type]struct{}
}

func (tdb *DBFuncTemplateDB) Delims(leftDelim, rightDelim string) {
	tdb.leftDelim = leftDelim
	tdb.rightDelim = rightDelim
}

func (tdb *DBFuncTemplateDB) SqlParamsConvert(sqlParamsConvert func(val reflect.Value) (string, any)) {
	tdb.sqlParamsConvert = sqlParamsConvert
}

func (tdb *DBFuncTemplateDB) SqlDebug(sqlDebug bool) {
	tdb.sqlDebug = sqlDebug
}

func (tdb *DBFuncTemplateDB) LogFunc(logFunc func(ctx context.Context, info string)) {
	tdb.logFunc = logFunc
}

func (tdb *DBFuncTemplateDB) AddSqlParamType(t reflect.Type) {
	tdb.sqlParamType[t] = struct{}{}
}

func NewDBFuncTemplateDB(sqlDB *sql.DB) *DBFuncTemplateDB {
	tdb := &DBFuncTemplateDB{
		db:        sqlDB,
		leftDelim: "{", rightDelim: "}",
		sqlParamType: make(map[reflect.Type]struct{}),
	}
	for k, v := range sqlParamType {
		tdb.sqlParamType[k] = v
	}
	return tdb
}

type FuncExecOption struct {
	ctx        context.Context
	param      any
	args       []any
	args_Index map[int]any
	result     []reflect.Value
	sql        string
}

func (tdb *DBFuncTemplateDB) templateBuild(templateSql *template.Template, op *FuncExecOption) error {
	var err error
	op.sql, op.args, err = templateSql.ExecuteBuilder(op.param, op.args, op.args_Index)
	if err != nil {
		return err
	}
	if templateSql.NotPrepare {
		op.sql, err = SqlInterpolateParams(op.sql, op.args)
		if err != nil {
			return err
		}
		op.args = nil
	}
	if tdb.sqlDebug && tdb.logFunc != nil {
		interpolateParamsSql, err := SqlInterpolateParams(op.sql, op.args)
		if err != nil {
			tdb.logFunc(op.ctx, fmt.Sprintf("sql not print by error[%v]", err))
		} else {
			tdb.logFunc(op.ctx, interpolateParamsSql)
		}
	}
	return err
}

func (tdb *DBFuncTemplateDB) query(db sqlDB, op *FuncExecOption) error {
	if op.ctx == nil {
		op.ctx = context.Background()
	}
	rows, err := db.QueryContext(op.ctx, op.sql, op.args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	columns, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	rt := reflect.TypeOf(op.result)
	dest, more, err := newScanDestByValues(tdb.sqlParamType, columns, op.result)
	if err != nil {
		return err
	}
	i := 0
	for rows.Next() {
		nextScan(op.result, dest)
		err = rows.Scan(dest...)
		if err != nil {
			return err
		}
		if more {
			if rt.Kind() == reflect.Array && i == rt.Len() {
				break
			}
			i++
		} else {
			break
		}
	}
	return nil
}

func (tdb *DBFuncTemplateDB) exec(db sqlDB, op *FuncExecOption) (ret *Result, err error) {
	if op.ctx == nil {
		op.ctx = context.Background()
	}
	result, err := db.ExecContext(op.ctx, op.sql, op.args...)
	if err != nil {
		return nil, err
	}
	lastInsertId, err := result.LastInsertId()
	if err != nil {
		return &Result{}, nil
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return &Result{}, nil
	}
	return &Result{lastInsertId, rowsAffected}, nil
}

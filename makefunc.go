package templatedb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/tianxinzizhen/templatedb/util"
)

var (
	contextType       = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType         = reflect.TypeOf((*error)(nil)).Elem()
	ResultType        = reflect.TypeOf((*Result)(nil))
	PrepareResultType = reflect.TypeOf((*PrepareResult)(nil))
)

type Operation int

const (
	ExecAction Operation = iota
	PrepareAction
	SelectAction
	SelectScanAction
	ExecNoResultAction
	SelectParamAction
)

type DBFunc[T any] struct {
	Begin      func() (*T, error)
	BeginTx    func(ctx context.Context, opts *sql.TxOptions) (*T, error)
	AutoCommit func(errp *error)
	Recover    func(errp *error)
}

type Result struct {
	LastInsertId int64
	RowsAffected int64
}

type PrepareResult struct {
	RowsAffected int64
}

// 自动初始化构造方法
func DBFuncInit[T any](dbFuncStruct *T, tdb TemplateDB) (*T, error) {
	dv, isNil := util.Indirect(reflect.ValueOf(dbFuncStruct))
	if isNil {
		return nil, errors.New("InitMakeFunc In(0) is nil")
	}
	dt := dv.Type()
	if dt.Kind() != reflect.Struct {
		return nil, errors.New("InitMakeFunc In(0) type is not struct")
	}
	if !dv.FieldByName("DBFunc").IsValid() {
		return nil, errors.New("strcut type need anonymous templatedb.DBFunc")
	}
	dv.FieldByName("Begin").Set(reflect.ValueOf(func() (*T, error) {
		if db, ok := tdb.(*DefaultDB); ok {
			tx, err := db.Begin()
			if err != nil {
				return nil, err
			}
			nt := new(T)
			DBFuncInit(nt, tx)
			return nt, nil
		} else {
			return nil, fmt.Errorf("Begin error: Currently in a transactional state")
		}
	}))
	dv.FieldByName("BeginTx").Set(reflect.ValueOf(func(ctx context.Context, opts *sql.TxOptions) (*T, error) {
		if db, ok := tdb.(*DefaultDB); ok {
			tx, err := db.BeginTx(ctx, opts)
			if err != nil {
				return nil, err
			}
			nt := new(T)
			DBFuncInit(nt, tx)
			return nt, nil
		} else {
			return nil, fmt.Errorf("BeginTx error: Currently in a transactional state")
		}
	}))
	if tx, ok := tdb.(*TemplateTxDB); ok {
		dv.FieldByName("AutoCommit").Set(reflect.ValueOf(tx.AutoCommit))
	} else {
		dv.FieldByName("AutoCommit").Set(reflect.ValueOf(func(errp *error) {}))
	}
	if db, ok := tdb.(*DefaultDB); ok {
		dv.FieldByName("Recover").Set(reflect.ValueOf(db.Recover))
	} else {
		dv.FieldByName("Recover").Set(reflect.ValueOf(func(errp *error) {}))
	}
	for i := 0; i < dt.NumField(); i++ {
		dist := dt.Field(i)
		dit := dist.Type
		div := dv.Field(i)
		if dit.Kind() == reflect.Func {
			if dit.NumIn() > 3 {
				return nil, fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func In Len >3", dt.PkgPath(), dt.Name(), dist.Name)
			}
			if dit.NumOut() == 1 || dit.NumOut() == 2 {
				switch dit.Out(0) {
				case ResultType, ResultType.Elem():
					div.Set(makeDBFunc(dit, tdb, ExecAction, fmt.Sprintf("%s.%s.%s", dt.PkgPath(), dt.Name(), dist.Name), ""))
				case PrepareResultType, PrepareResultType.Elem():
					div.Set(makeDBFunc(dit, tdb, PrepareAction, fmt.Sprintf("%s.%s.%s", dt.PkgPath(), dt.Name(), dist.Name), ""))
				default:
					if dit.Out(0).Implements(errorType) {
						if dit.NumIn() > 0 && dit.In(dit.NumIn()-1).Kind() == reflect.Func {
							div.Set(makeDBFunc(dit, tdb, SelectScanAction, fmt.Sprintf("%s.%s.%s", dt.PkgPath(), dt.Name(), dist.Name), ""))
						} else {
							div.Set(makeDBFunc(dit, tdb, ExecNoResultAction, fmt.Sprintf("%s.%s.%s", dt.PkgPath(), dt.Name(), dist.Name), ""))
						}
					} else {
						if dit.NumOut() == 2 && !dit.Out(1).Implements(errorType) {
							return nil, fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func Out type is not correct", dt.PkgPath(), dt.Name(), dist.Name)
						}
						div.Set(makeDBFunc(dit, tdb, SelectAction, fmt.Sprintf("%s.%s.%s", dt.PkgPath(), dt.Name(), dist.Name), ""))
					}
				}
			} else if dit.NumOut() == 0 {
				if dit.NumIn() > 0 && dit.In(dit.NumIn()-1).Kind() == reflect.Func {
					div.Set(makeDBFunc(dit, tdb, SelectScanAction, fmt.Sprintf("%s.%s.%s", dt.PkgPath(), dt.Name(), dist.Name), ""))
				} else {
					div.Set(makeDBFunc(dit, tdb, ExecNoResultAction, fmt.Sprintf("%s.%s.%s", dt.PkgPath(), dt.Name(), dist.Name), ""))
				}
			} else {
				return nil, fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func In and Out type is not correct", dt.PkgPath(), dt.Name(), dist.Name)
			}
		}
	}
	return dbFuncStruct, nil
}

func makeDBFunc(t reflect.Type, tdb TemplateDB, action Operation, pkg, fieldName string) reflect.Value {
	return reflect.MakeFunc(t, func(args []reflect.Value) (results []reflect.Value) {
		var ctx context.Context
		var param, scanFunc any
		for _, v := range args {
			if v.Type().Implements(contextType) {
				ctx = v.Interface().(context.Context)
			} else if v.Type().Kind() == reflect.Func {
				scanFunc = v.Interface()
			} else {
				param = v.Interface()
			}
		}
		if ctx == nil {
			ctx = context.Background()
		}
		results = make([]reflect.Value, t.NumOut())
		if t.NumOut() == 2 || (t.NumOut() == 1 && t.Out(0).Implements(errorType)) {
			defer func() {
				e := recover()
				if e != nil {
					switch rerr := e.(type) {
					case error:
						if t.NumOut() == 2 {
							results[0] = reflect.Zero(t.Out(0))
						}
						results[len(results)-1] = reflect.ValueOf(rerr)
						recoverPrintf(rerr)
					default:
						panic(e)
					}
				}
			}()
			results[len(results)-1] = reflect.Zero(errorType)
		}
		switch action {
		case ExecAction:
			lastInsertId, rowsAffected := tdb.ExecContext(ctx, param, pkg, fieldName)
			result := reflect.ValueOf(&Result{LastInsertId: lastInsertId, RowsAffected: rowsAffected})
			if t.Out(0) == ResultType || t.Out(0) == ResultType.Elem() {
				if t.Out(0).Kind() == reflect.Pointer {
					results[0] = result
				} else {
					results[0] = result.Elem()
				}
			}
		case PrepareAction:
			pv := reflect.ValueOf(param)
			var pvs []any
			if pv.IsValid() && (pv.Kind() == reflect.Slice || pv.Kind() == reflect.Array) {
				for i := 0; i < pv.Len(); i++ {
					pvs = append(pvs, pv.Index(i).Interface())
				}
			}
			rowsAffected := tdb.PrepareExecContext(ctx, pvs, pkg, fieldName)
			prepareResult := reflect.ValueOf(&PrepareResult{RowsAffected: rowsAffected})
			if t.Out(0) == PrepareResultType || t.Out(0) == PrepareResultType.Elem() {
				if t.Out(0).Kind() == reflect.Pointer {
					results[0] = prepareResult
				} else {
					results[0] = prepareResult.Elem()
				}
			}
		case SelectAction:
			results[0] = tdb.selectByType(ctx, param, t.Out(0), pkg, fieldName)
		case SelectScanAction:
			tdb.SelectScanFuncContext(ctx, param, scanFunc, pkg, fieldName)
		case ExecNoResultAction:
			tdb.ExecContext(ctx, param, pkg, fieldName)
		}
		return results
	})
}

func OptionDBFuncInit[T any](dbFuncStruct *T, tdb TemplateOptionDB) (*T, error) {
	dv, isNil := util.Indirect(reflect.ValueOf(dbFuncStruct))
	if isNil {
		return nil, errors.New("InitMakeFunc In(0) is nil")
	}
	dt := dv.Type()
	if dt.Kind() != reflect.Struct {
		return nil, errors.New("InitMakeFunc In(0) type is not struct")
	}
	if !dv.FieldByName("DBFunc").IsValid() {
		return nil, errors.New("strcut type need anonymous templatedb.DBFunc")
	}
	dv.FieldByName("Begin").Set(reflect.ValueOf(func() (*T, error) {
		if db, ok := tdb.(*OptionDB); ok {
			tx, err := db.Begin()
			if err != nil {
				return nil, err
			}
			nt := new(T)
			OptionDBFuncInit(nt, tx)
			return nt, nil
		} else {
			return nil, fmt.Errorf("Begin error: Currently in a transactional state")
		}
	}))
	dv.FieldByName("BeginTx").Set(reflect.ValueOf(func(ctx context.Context, opts *sql.TxOptions) (*T, error) {
		if db, ok := tdb.(*OptionDB); ok {
			tx, err := db.BeginTx(ctx, opts)
			if err != nil {
				return nil, err
			}
			nt := new(T)
			OptionDBFuncInit(nt, tx)
			return nt, nil
		} else {
			return nil, fmt.Errorf("BeginTx error: Currently in a transactional state")
		}
	}))
	if tx, ok := tdb.(*OptionTxDB); ok {
		dv.FieldByName("AutoCommit").Set(reflect.ValueOf(tx.AutoCommit))
	} else {
		dv.FieldByName("AutoCommit").Set(reflect.ValueOf(func(errp *error) {}))
	}
	if db, ok := tdb.(*OptionDB); ok {
		dv.FieldByName("Recover").Set(reflect.ValueOf(db.Recover))
	} else {
		dv.FieldByName("Recover").Set(reflect.ValueOf(func(errp *error) {}))
	}
	for i := 0; i < dt.NumField(); i++ {
		dist := dt.Field(i)
		dit := dist.Type
		div := dv.Field(i)
		if dit.Kind() == reflect.Func {
			var action Operation = ExecNoResultAction
			for i := 0; i < dit.NumIn(); i++ {
				ditIni := dit.In(i)
				if ditIni.Implements(contextType) {
					continue
				}
				if ditIni.Kind() == reflect.Pointer {
					ditIni = ditIni.Elem()
				}
				if ditIni.Kind() == reflect.Func {
					action = SelectScanAction
					break
				}
				if ditIni.Kind() != reflect.Map && ditIni.Kind() != reflect.Struct && ditIni.Kind() != reflect.Slice {
					action = SelectParamAction
				}
			}
			if dit.NumOut() > 0 {
				if dit.Out(0) == ResultType || dit.Out(0) == ResultType.Elem() {
					action = ExecAction
				} else if !dit.Out(0).Implements(errorType) && action != SelectParamAction {
					action = SelectAction
				}
			}
			div.Set(optionMakeDBFunc(dit, tdb, action, fmt.Sprintf("%s.%s.%s", dt.PkgPath(), dt.Name(), dist.Name)))
		}
	}
	return dbFuncStruct, nil
}

func optionMakeDBFunc(t reflect.Type, tdb TemplateOptionDB, action Operation, funcName string) reflect.Value {
	return reflect.MakeFunc(t, func(args []reflect.Value) (results []reflect.Value) {
		op := NewExecOption()
		var opArgs []any
		for _, v := range args {
			if v.Type().Implements(contextType) {
				op.SetContext(v.Interface().(context.Context))
			} else if v.Type().Kind() == reflect.Func {
				op.SetResult(v.Interface())
			} else if action == SelectParamAction {
				opArgs = append(opArgs, v.Interface())
			} else {
				op.SetParam(v.Interface())
			}
		}
		op.SetArgs(opArgs...)
		op.SetFuncName(funcName)
		results = make([]reflect.Value, t.NumOut())
		if t.NumOut() == 2 || (t.NumOut() == 1 && t.Out(0).Implements(errorType)) {
			defer func() {
				e := recover()
				if e != nil {
					switch rerr := e.(type) {
					case error:
						if t.NumOut() == 2 {
							results[0] = reflect.Zero(t.Out(0))
						}
						results[len(results)-1] = reflect.ValueOf(rerr)
						recoverPrintf(rerr)
					default:
						panic(e)
					}
				}
			}()
			results[len(results)-1] = reflect.Zero(errorType)
		}
		switch action {
		case ExecAction:
			lastInsertId, rowsAffected := tdb.Exec(op)
			result := reflect.ValueOf(&Result{LastInsertId: lastInsertId, RowsAffected: rowsAffected})
			if t.Out(0).Kind() == reflect.Pointer {
				results[0] = result
			} else {
				results[0] = result.Elem()
			}
		case SelectAction:
			op.SetResult(reflect.Zero(t.Out(0)).Interface())
			results[0] = reflect.ValueOf(tdb.Query(op))
		case SelectScanAction:
			tdb.Query(op)
		case ExecNoResultAction:
			tdb.Exec(op)
		}
		return results
	})
}

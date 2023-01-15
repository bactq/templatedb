package templatedb

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/tianxinzizhen/templatedb/util"
)

var (
	templateDBType = reflect.TypeOf((*TemplateDB)(nil)).Elem()
	contextType    = reflect.TypeOf((*context.Context)(nil)).Elem()
)

// 自动初始化构造方法
func DBFuncMake(dbStruct any) error {
	dv, isNil := util.Indirect(reflect.ValueOf(dbStruct))
	if isNil {
		return errors.New("InitMakeFunc In(0) is nil")
	}
	dt := dv.Type()
	if dt.Kind() != reflect.Struct {
		return errors.New("InitMakeFunc In(0) type is not struct")
	}
	for i := 0; i < dt.NumField(); i++ {
		dist := dt.Field(i)
		dit := dist.Type
		div := dv.Field(i)
		if dit.Kind() != reflect.Func {
			return fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] type is not Func", dt.PkgPath(), dt.Name(), dist.Name)
		}
		//need judgment function in parameter type is correct
		switch dit.NumIn() {
		case 1:
			if !(dit.In(0).Implements(templateDBType)) {
				return fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func In(0) types is Implements TemplateDB", dt.PkgPath(), dt.Name(), dist.Name)
			}
		case 2:
			if !(dit.In(0).Implements(templateDBType) ||
				(dit.In(0).Implements(contextType) && dit.In(1).Implements(templateDBType))) {
				return fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func In types is not correct", dt.PkgPath(), dt.Name(), dist.Name)
			}
		case 3:
			if !dit.In(0).Implements(contextType) {
				return fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func In(0) type is not Implements context.Context", dt.PkgPath(), dt.Name(), dist.Name)
			}
			if !dit.In(1).Implements(templateDBType) {
				return fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func In(1) type is not Implements TemplateDB", dt.PkgPath(), dt.Name(), dist.Name)
			}
		default:
			return fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func in type is not correct", dt.PkgPath(), dt.Name(), dist.Name)
		}
		// select exec lastid affected
		if tdb, ok := dist.Tag.Lookup("tdb"); ok {
			if len(tdb) > 0 {
				if tdb[0] == '>' {
					if dit.NumOut() > 2 {
						return fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func Out len > 2 of exec ", dt.PkgPath(), dt.Name(), dist.Name)
					}
					div.Set(makeExecFunc(dit, tdb[1:], fmt.Sprintf("%s.%s", dt.PkgPath(), dt.Name()), dist.Name))
				} else if tdb[0] == '<' {
					if dit.NumOut() > 1 {
						return fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func Out len > 1 of select ", dt.PkgPath(), dt.Name(), dist.Name)
					}
					div.Set(makeSelectFunc(dit, fmt.Sprintf("%s.%s", dt.PkgPath(), dt.Name()), dist.Name))
				} else {
					return fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func tag tdb[0] not '>' exec or '<' select ", dt.PkgPath(), dt.Name(), dist.Name)
				}
			} else {
				return fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func tdb tag format not correct", dt.PkgPath(), dt.Name(), dist.Name)
			}
		} else {
			return fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func not set tdb tag", dt.PkgPath(), dt.Name(), dist.Name)
		}
	}
	return nil
}

func makeSelectFunc(t reflect.Type, pkg, fieldName string) reflect.Value {
	return reflect.MakeFunc(t, func(args []reflect.Value) (results []reflect.Value) {
		var ctx context.Context
		var tdb TemplateDB
		var param any
		if t.NumIn() == 1 {
			tdb = args[0].Interface().(TemplateDB)
		}
		if t.NumIn() == 2 {
			if t.In(1).Implements(templateDBType) {
				ctx = args[0].Interface().(context.Context)
				tdb = args[1].Interface().(TemplateDB)
			} else {
				tdb = args[0].Interface().(TemplateDB)
				param = args[1].Interface()
			}
		}
		if t.NumIn() == 3 {
			ctx = args[0].Interface().(context.Context)
			tdb = args[1].Interface().(TemplateDB)
			param = args[1].Interface()
		}
		if ctx == nil {
			ctx = context.Background()
		}
		return []reflect.Value{tdb.selectByType(ctx, param, t.Out(0), pkg, fieldName)}
	})
}

func makeExecFunc(t reflect.Type, actoin, pkg, fieldName string) reflect.Value {
	return reflect.MakeFunc(t, func(args []reflect.Value) (results []reflect.Value) {
		var ctx context.Context
		var tdb TemplateDB
		var param any
		if t.NumIn() == 1 {
			tdb = args[0].Interface().(TemplateDB)
		}
		if t.NumIn() == 2 {
			if t.In(1).Implements(templateDBType) {
				ctx = args[0].Interface().(context.Context)
				tdb = args[1].Interface().(TemplateDB)
			} else {
				tdb = args[0].Interface().(TemplateDB)
				param = args[1].Interface()
			}
		}
		if t.NumIn() == 3 {
			ctx = args[0].Interface().(context.Context)
			tdb = args[1].Interface().(TemplateDB)
			param = args[1].Interface()
		}
		if ctx == nil {
			ctx = context.Background()
		}
		var lastInsertId, rowsAffected int
		if len(actoin) > 0 && actoin[0] == '>' {
			actoin = actoin[1:]
			pv := reflect.ValueOf(param)
			var pvs []any
			if pv.IsValid() && (pv.Kind() == reflect.Slice || pv.Kind() == reflect.Array) {
				for i := 0; i < pv.Len(); i++ {
					pvs = append(pvs, pv.Index(i).Interface())
				}
			}
			rowsAffected = tdb.PrepareExecContext(ctx, pvs, pkg, fieldName)
		} else {
			lastInsertId, rowsAffected = tdb.ExecContext(ctx, param, pkg, fieldName)
		}
		var ret []reflect.Value
		for i := 0; i < t.NumOut(); i++ {
			if len(actoin) > i && actoin[i] == 'a' {
				ret = append(ret, reflect.ValueOf(rowsAffected))
				continue
			}
			if len(actoin) > i && actoin[i] == 'l' {
				ret = append(ret, reflect.ValueOf(lastInsertId))
				continue
			}
		}
		if len(ret) != t.NumOut() {
			ret = nil
			if t.NumOut() == 1 {
				ret = append(ret, reflect.ValueOf(rowsAffected))
			}
			if t.NumOut() == 2 {
				ret = append(ret, reflect.ValueOf(lastInsertId))
				ret = append(ret, reflect.ValueOf(rowsAffected))
			}
		}
		return ret
	})
}

package templatedb

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/tianxinzizhen/templatedb/util"
)

var (
	anyType        = reflect.TypeOf((*any)(nil)).Elem()
	templateDBType = reflect.TypeOf((*TemplateDB)(nil)).Elem()
	contextType    = reflect.TypeOf((*context.Context)(nil)).Elem()
)

// 自动初始化构造方法
func InitMakeFunc(dbStruct any) error {
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

				} else if tdb[0] == '<' {
					sf := reflect.MakeFunc(dit, func(args []reflect.Value) (results []reflect.Value) {

						return nil
					})
					div.Set(sf)
				} else {
					return fmt.Errorf("InitMakeFunc[%s.%s] Field[%s] Func tag tdb[0] not '>' or '<'", dt.PkgPath(), dt.Name(), dist.Name)
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

func makeSelectFunc(t reflect.Type) reflect.Value {
	return reflect.MakeFunc(t, func(args []reflect.Value) (results []reflect.Value) {
		var context, tdb, param *reflect.Value
		if t.NumIn() == 1 {
			tdb = &args[0]
		}
		if t.NumIn() == 2 {
			if t.In(1).Implements(templateDBType) {
				context = &args[0]
				tdb = &args[1]
			} else {
				tdb = &args[0]
				param = &args[1]
			}
		}
		if t.NumIn() == 3 {
			context = &args[0]
			tdb = &args[2]
			param = &args[3]
		}
		return nil
	})
}

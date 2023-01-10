package util

import "reflect"

type StructScaner struct {
	Dest  reflect.Value
	Index []int
}

func (s *StructScaner) Scan(src any) error {
	if src == nil {
		return nil
	}
	return convertAssign(s.Dest.FieldByIndex(s.Index).Addr().Interface(), src)
}

type MapScaner struct {
	Name string
	Dest reflect.Value
}

func (s *MapScaner) Scan(src any) error {
	if src == nil {
		return nil
	} else {
		sv := reflect.ValueOf(src)
		if sv.CanConvert(s.Dest.Type().Elem()) {
			s.Dest.SetMapIndex(reflect.ValueOf(s.Name), sv.Convert(s.Dest.Type().Elem()))
		}
		return nil
	}
}

type SliceScaner struct {
	Dest  reflect.Value
	Index int
}

func (s *SliceScaner) Scan(src any) error {
	if src == nil {
		return nil
	} else {
		sv := reflect.ValueOf(src)
		if sv.CanConvert(s.Dest.Type().Elem()) {
			s.Dest.Index(s.Index).Set(sv.Convert(s.Dest.Type().Elem()))
		}
		return nil
	}
}

package test

import (
	"fmt"
	"testing"

	"github.com/tianxinzizhen/templatedb"
)

type MTest struct {
	Select      func(templatedb.TemplateDB, map[string]any) *GoodShop `tdb:"<"`
	Exec        func(templatedb.TemplateDB, []GoodShop) int           `tdb:">l"`
	PrepareExec func(templatedb.TemplateDB, []GoodShop) int           `tdb:">>a"`
}

func TestMakeSelectFunc(t *testing.T) {
	dest := &MTest{}
	err := templatedb.DBFuncMake(dest)
	if err != nil {
		t.Error(err)
	}
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	// for _, v := range dest.Select(db) {
	// 	fmt.Printf("%#v", v)
	// }
	fmt.Printf("%#v", dest.Select(db, map[string]any{
		"id": 59,
	}))
}

func TestMakeExecFunc(t *testing.T) {
	dest := &MTest{}
	err := templatedb.DBFuncMake(dest)
	if err != nil {
		t.Error(err)
	}
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	defer db.Recover(&err)
	a := dest.Exec(db, []GoodShop{{
		Name:         "insertOne",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍",
		Avatar:       "aa.jpg",
		Image:        "bb.jpg",
	}})
	fmt.Println(a)
}

func TestMakePrepareExecFunc(t *testing.T) {
	dest := &MTest{}
	err := templatedb.DBFuncMake(dest)
	if err != nil {
		t.Error(err)
	}
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	defer db.Recover(&err)
	a := dest.PrepareExec(db, []GoodShop{{
		Name:         "insertOne",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍",
		Avatar:       "aa.jpg",
		Image:        "bb.jpg",
	}, {
		Name:         "insertOne",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍",
		Avatar:       "aa.jpg",
		Image:        "bb.jpg",
	}})
	fmt.Println(a)
}

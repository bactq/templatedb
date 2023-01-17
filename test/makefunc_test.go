package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/tianxinzizhen/templatedb"
)

type MTest struct {
	templatedb.DBFunc[MTest]
	Select            func(map[string]any, context.Context) ([]GoodShop, error)
	Exec              func([]GoodShop) templatedb.Result
	ExecNoResult      func([]GoodShop)
	ExecNoResultError func([]GoodShop) error
	PrepareExec       func([]GoodShop) templatedb.PrepareResult
}

func TestMakeSelectFunc(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	dest := &MTest{}
	_, err = templatedb.DBFuncInit(dest, db)
	if err != nil {
		t.Error(err)
	}
	defer dest.Recover(&err)
	data, err := dest.Select(nil, context.Background())
	if err != nil {
		t.Error(err)
	}
	for _, v := range data {
		fmt.Printf("%#v\n", v)
	}
	// fmt.Printf("%#v", dest.Select(db, map[string]any{
	// 	"id": 1,
	// }))
}

func TestMakeExecFunc(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	dest := &MTest{}
	_, err = templatedb.DBFuncInit(dest, db)
	if err != nil {
		t.Error(err)
	}
	defer db.Recover(&err)
	dest, _ = dest.Begin()
	defer dest.AutoCommit(&err)
	a := dest.Exec([]GoodShop{{
		Name:         "insertOne",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍1",
		Avatar:       "aa.jpg",
		Image:        "bb.jpg",
	}})
	fmt.Println(a)
}

func TestMakeExecFuncNoResult(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	dest := &MTest{}
	_, err = templatedb.DBFuncInit(dest, db)
	if err != nil {
		t.Error(err)
	}
	defer db.Recover(&err)
	dest, _ = dest.Begin()
	defer dest.AutoCommit(&err)
	dest.ExecNoResult([]GoodShop{{
		Name:         "insertOne",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍1",
		Avatar:       "aa.jpg",
		Image:        "bb.jpg",
	}})
	// fmt.Println(a)
}

func TestMakeExecFuncNoResultError(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	dest := &MTest{}
	_, err = templatedb.DBFuncInit(dest, db)
	if err != nil {
		t.Error(err)
	}
	defer db.Recover(&err)
	dest, _ = dest.Begin()
	defer dest.AutoCommit(&err)
	err = dest.ExecNoResultError([]GoodShop{{
		Name:         "insertOne",
		UserId:       2,
		Phone:        "12345678910",
		Introduction: "一些简单的介绍1",
		Avatar:       "aa.jpg1",
		Image:        "bb.jpg",
	}})
	if err != nil {
		t.Error(err)
	}
}

func TestMakePrepareExecFunc(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	dest := &MTest{}
	_, err = templatedb.DBFuncInit(dest, db)
	if err != nil {
		t.Error(err)
	}
	defer db.Recover(&err)
	a := dest.PrepareExec([]GoodShop{{
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

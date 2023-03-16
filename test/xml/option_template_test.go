package xml

import (
	"embed"
	"fmt"
	"testing"

	"github.com/tianxinzizhen/templatedb"
	"github.com/tianxinzizhen/templatedb/test"
)

//go:embed sql
var sqlDir embed.FS

func GetOptionDB() (*templatedb.OptionDB, error) {
	tdb, err := test.GetOptionDB()
	if err != nil {
		return nil, err
	}
	tdb.LoadSqlOfXml(sqlDir)
	return tdb, nil
}
func TestOptionSelectScan(t *testing.T) {
	db, err := GetOptionDB()
	if err != nil {
		t.Error(err)
	}
	db.Query(&templatedb.ExecOption{
		Sql: "select UserId, Name FROM tbl_test",
		Result: func(id int, name string) {
			fmt.Println(id, name)
		},
	})
}

type Info struct {
	UserId int
	Name   string
}

func TestOptionSelect(t *testing.T) {
	db, err := GetOptionDB()
	if err != nil {
		t.Error(err)
	}
	//查询多条时使用的行来判断不同的sql语句
	rr := &[]**Info{}
	db.Query(&templatedb.ExecOption{
		Sql:    "select UserId, Name FROM tbl_test",
		Result: &rr,
	})
	for _, v := range *rr {
		fmt.Println(*v)
	}
	ret := db.TQuery(&templatedb.ExecOption{
		Sql:    "select Name FROM tbl_test",
		Result: (*[]**Info)(nil),
	}).([]**Info)
	fmt.Println(ret)
	for _, v := range ret {
		fmt.Printf("%#v", *v)
	}
}

func TestOptionSelectArgs(t *testing.T) {
	db, err := GetOptionDB()
	if err != nil {
		t.Error(err)
	}
	sret := []*Info{}
	//查询多条时使用的行来判断不同的sql语句
	db.Query(&templatedb.ExecOption{
		Sql:    "select UserId, Name FROM tbl_test ",
		Result: &sret,
		Args:   []any{1},
		Param:  Info{Name: "dd"},
	})
	for _, v := range sret {
		fmt.Printf("%#v", v)
	}
}

func TestOptionSelectScanFunc(t *testing.T) {
	db, err := GetOptionDB()
	if err != nil {
		t.Error(err)
	}
	//查询多条时使用的行来判断不同的sql语句
	db.Query(&templatedb.ExecOption{
		Sql: "select UserId, Name FROM tbl_test ",
		Result: func(UserId, Name string) {
			fmt.Printf("%s,%s", UserId, Name)
		},
		Args:  []any{1},
		Param: Info{Name: "dd"},
	})
}

func TestOptionSelectFunc(t *testing.T) {
	db, err := GetOptionDB()
	if err != nil {
		t.Error(err)
	}
	var aa func() (UserId, Name string)
	//查询多条时使用的行来判断不同的sql语句
	ret := db.TQuery(&templatedb.ExecOption{
		Sql:    "select UserId, Name FROM tbl_test ",
		Result: aa,
		Args:   []any{1},
		Param:  Info{Name: "dd"},
	}).(func() (UserId, Name string))
	if err != nil {
		t.Error(err)
	}
	UserId, Name := ret()
	fmt.Printf("%s,%s", UserId, Name)
}

func TestOptionSelectXml(t *testing.T) {
	db, err := GetOptionDB()
	if err != nil {
		t.Error(err)
	}
	//查询多条时使用的行来判断不同的sql语句
	ret := db.TQuery(&templatedb.ExecOption{
		FuncPC: templatedb.FuncPC(TestOptionSelectXml),
		Result: []*Info{},
		Args:   []any{1},
		Param:  Info{Name: "dd"},
	}).([]*Info)
	for _, v := range ret {
		fmt.Println(v)
	}
}

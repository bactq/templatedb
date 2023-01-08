package test

import (
	"database/sql"
	"embed"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/tianxinzizhen/templatedb"
)

//go:embed sql
var sqlDir embed.FS

type GoodShop struct {
	Id           int    `json:"id"`
	UserId       int    `json:"userId"`
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	Introduction string `json:"introduction"`
	Avatar       string `json:"avatar"`
	Image        string `json:"image"`
}

func getDB() (*templatedb.DefaultDB, error) {
	sqldb, err := sql.Open("mysql", "lix:lix@/test")
	if err != nil {
		return nil, err
	}
	return templatedb.NewDefaultDB(sqldb, templatedb.LoadSqlByEmbedFS(sqlDir)), nil
}

var testParam = []struct {
	name  string
	param any
}{
	{name: "select", param: GoodShop{
		Name: "0店铺1",
	}},
	{name: "sqlparam", param: GoodShop{
		Name: "0店铺1",
	}},
	{name: "sqlparam", param: GoodShop{
		Name: "0店铺1",
	}},
	{name: "", param: GoodShop{
		Name: "0店铺1",
	}},
}

func TestSelect(t *testing.T) {
	db, err := getDB()
	if err != nil {
		t.Error(err)
	}
	for _, tp := range testParam[3:] {
		ret, err := templatedb.DBSelect[GoodShop](db).Select(templatedb.GetCallerFuncName(tp.name), tp.param)
		if err != nil {
			t.Error(err)
		}
		for _, v := range ret {
			fmt.Printf("%#v", v)
		}
	}
}

package test

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/tianxinzizhen/templatedb"
)

func GetOptionDB() (*templatedb.OptionDB, error) {
	sqldb, err := sql.Open("mysql", "root:lz@3306!@tcp(mysql.local.lezhichuyou.com:3306)/lz_tour?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true")
	if err != nil {
		return nil, err
	}
	templatedb.LogPrintf = func(ctx context.Context, info string) {
		fmt.Println(info)
	}
	return templatedb.NewOptionDB(sqldb), nil
}

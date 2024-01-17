package test

import (
	"context"
	"embed"

	"github.com/tianxinzizhen/templatedb"
)

/*
init sql :
create table test(

	id int,
	name varchar(20)

);
insert into test values(1,"a");
*/
type TestDB struct {
	//sql select * from test where id=?
	Select func(ctx context.Context, id int) ([]*Test, error)

	/*sql
	select * from test where id=? limit 1
	*/
	SelectOne func(ctx context.Context, id int) (*Test, error)
}

//go:embed test_db.go
var testDbSql embed.FS

func NewTestDB(tdb *templatedb.DBFuncTemplateDB) (*TestDB, error) {
	ret := &TestDB{}
	err := templatedb.DBFuncContextInit(tdb, ret, templatedb.LoadComment, testDbSql)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

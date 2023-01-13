package templatedb

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/tianxinzizhen/templatedb/load"
	"github.com/tianxinzizhen/templatedb/load/xml"
	"github.com/tianxinzizhen/templatedb/template"
)

var (
	pkg          = "github.com/tianxinzizhen/templatedb"
	skipFuncName = map[string]any{
		fmt.Sprintf("%s.%s", pkg, "(*SelectDB[...]).Select"):         struct{}{},
		fmt.Sprintf("%s.%s", pkg, "(*SelectDB[...]).SelectFirst"):    struct{}{},
		fmt.Sprintf("%s.%s", pkg, "(*DefaultDB).Transaction"):        struct{}{},
		fmt.Sprintf("%s.%s", pkg, "(*DefaultDB).TransactionContext"): struct{}{},
	}
)

type TemplateDB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

type ActionDB interface {
	query(ctx context.Context, tdb TemplateDB, params any, name []any) (*sql.Rows, []*sql.ColumnType, string, error)
	selectScanFunc(ctx context.Context, tdb TemplateDB, params any, scanFunc any, name []any)
	exec(ctx context.Context, tdb TemplateDB, params any, name []any) (lastInsertId, rowsAffected int)
	prepareExecContext(ctx context.Context, tdb TemplateDB, params []any, name []any) (rowsAffected int)
}

// type TemplateDB interface {
// 	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
// 	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
// }

type DefaultDB struct {
	sqlDB                   *sql.DB
	template                map[string]*template.Template
	delimsLeft, delimsRight string
	tdb                     TemplateDB
	recoverPanic            bool
}

func getSkipFuncName(skip int, name []any) string {
	if len(name) > 0 && reflect.TypeOf(name[0]).Kind() == reflect.Func {
		return fmt.Sprintf("%s:%s", runtime.FuncForPC(reflect.ValueOf(name[0]).Pointer()).Name(), fmt.Sprint(name[1:]...))
	}
	for ; ; skip++ {
		pc, _, _, _ := runtime.Caller(skip)
		funcName := runtime.FuncForPC(pc).Name()
		if _, ok := skipFuncName[funcName]; ok {
			continue
		}
		return fmt.Sprintf("%s:%s", funcName, fmt.Sprint(name...))
	}
}

func Delims(delimsLeft, delimsRight string) func(*DefaultDB) error {
	return func(db *DefaultDB) error {
		db.delimsLeft = delimsLeft
		db.delimsRight = delimsRight
		return nil
	}
}

func RecoverPanic(recoverPanic bool) func(*DefaultDB) error {
	return func(db *DefaultDB) error {
		db.recoverPanic = recoverPanic
		return nil
	}
}

func LoadSqlOfXml(sqlDir embed.FS) func(*DefaultDB) error {
	return func(db *DefaultDB) error {
		if db.template == nil {
			db.template = make(map[string]*template.Template)
		}
		err := xml.LoadTemplateStatements(sqlDir, db.template, db.parse)
		if err != nil {
			return err
		}
		return nil
	}
}

func LoadSqlOfBytes(xmlSql []byte) func(*DefaultDB) error {
	return func(db *DefaultDB) error {
		if db.template == nil {
			db.template = make(map[string]*template.Template)
		}
		err := xml.LoadTemplateStatementsOfBytes(xmlSql, db.template, db.parse)
		if err != nil {
			return err
		}
		return nil
	}
}

func LoadSqlOfString(xmlSql string) func(*DefaultDB) error {
	return func(db *DefaultDB) error {
		if db.template == nil {
			db.template = make(map[string]*template.Template)
		}
		err := xml.LoadTemplateStatementsOfString(xmlSql, db.template, db.parse)
		if err != nil {
			return err
		}
		return nil
	}
}

func NewDefaultDB(SqlDB *sql.DB, options ...func(*DefaultDB) error) (*DefaultDB, error) {
	db := &DefaultDB{
		sqlDB:      SqlDB,
		template:   make(map[string]*template.Template),
		delimsLeft: "{", delimsRight: "}",
		tdb: SqlDB,
	}
	for _, fn := range options {
		err := fn(db)
		if err != nil {
			return nil, err
		}
	}
	return db, nil
}

func (db *DefaultDB) Recover(err *error) {
	if *err == nil {
		*err = recover().(error)
	}
	if *err != nil && db.recoverPanic {
		panic(*err)
	}
}

func (db *DefaultDB) parse(parse string, addParseTrees ...load.AddParseTree) (*template.Template, error) {
	templateSql, err := template.New("").Delims(db.delimsLeft, db.delimsRight).Funcs(sqlfunc).Parse(parse)
	if err != nil {
		return nil, err
	}
	for _, addParseTree := range addParseTrees {
		err = addParseTree(templateSql)
		if err != nil {
			return nil, err
		}
	}
	return templateSql, nil
}

func (db *DefaultDB) templateBuild(query string, params any) (sql string, args []any, err error) {
	if db.template == nil {
		db.template = make(map[string]*template.Template)
	}
	templateSql, templateok := db.template[query]
	if !templateok {
		_, query, _ := strings.Cut(query, ":")
		templateSql, err = db.parse(query)
		if err != nil {
			return
		}
		db.template[query] = templateSql
	}
	return templateSql.ExecuteBuilder(params)
}

func (db *DefaultDB) query(ctx context.Context, tdb TemplateDB, params any, name []any) (*sql.Rows, []*sql.ColumnType, string, error) {
	statement := getSkipFuncName(3, name)
	sql, args, err := db.templateBuild(statement, params)
	if err != nil {
		return nil, nil, statement, err
	}
	rows, err := tdb.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, nil, statement, err
	}
	columns, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, statement, err
	}
	return rows, columns, statement, nil
}

func (db *DefaultDB) selectScanFunc(ctx context.Context, tdb TemplateDB, params any, scanFunc any, name []any) {
	statement := getSkipFuncName(3, name)
	sql, args, err := db.templateBuild(statement, params)
	if err != nil {
		panic(fmt.Errorf("%s->%s", statement, err))
	}
	rows, err := tdb.QueryContext(ctx, sql, args...)
	if err != nil {
		panic(fmt.Errorf("%s->%s", statement, err))
	}
	defer rows.Close()
	columns, err := rows.ColumnTypes()
	if err != nil {
		panic(fmt.Errorf("%s->%s", statement, err))
	}
	if err != nil {
		panic(fmt.Errorf("%s->%s", statement, err))
	}
	st := reflect.TypeOf(scanFunc)
	if st.Kind() != reflect.Func {
		panic("parameter scanFunc is not function")
	}
	if st.NumIn() == 1 {
		t := st.In(0)
		sit := t
		if t.Kind() == reflect.Pointer {
			sit = t.Elem()
		}
		dest := newScanDest(columns, sit)
		for rows.Next() {
			receiver := newReceiver(sit, columns, dest)
			err = rows.Scan(dest...)
			if err != nil {
				panic(fmt.Errorf("%s->%s", statement, err))
			}
			if t.Kind() == reflect.Pointer {
				reflect.ValueOf(scanFunc).Call([]reflect.Value{receiver})
			} else {
				reflect.ValueOf(scanFunc).Call([]reflect.Value{receiver.Elem()})
			}
		}
	} else {
		dest := newScanDest(columns, st)
		for rows.Next() {
			receiver := newReceiver(st, columns, dest)
			err = rows.Scan(dest...)
			if err != nil {
				panic(fmt.Errorf("%s->%s", statement, err))
			}
			reflect.ValueOf(scanFunc).Call(receiver.Interface().([]reflect.Value))
		}
	}
}

func (db *DefaultDB) exec(ctx context.Context, tdb TemplateDB, params any, name []any) (lastInsertId, rowsAffected int) {
	statement := getSkipFuncName(3, name)
	tsql, args, err := db.templateBuild(statement, params)
	if err != nil {
		panic(fmt.Errorf("%s->%s", statement, err))
	}
	result, err := tdb.ExecContext(ctx, tsql, args...)
	if err != nil {
		panic(fmt.Errorf("%s->%s", statement, err))
	}
	lastid, err := result.LastInsertId()
	if err != nil {
		panic(fmt.Errorf("%s->%s", statement, err))
	}
	affected, err := result.RowsAffected()
	if err != nil {
		panic(fmt.Errorf("%s->%s", statement, err))
	}
	return int(lastid), int(affected)
}

func (db *DefaultDB) prepareExecContext(ctx context.Context, tdb TemplateDB, params []any, name []any) (rowsAffected int) {
	statement := getSkipFuncName(3, name)
	var stmtMaps map[string]*sql.Stmt = make(map[string]*sql.Stmt)
	var tempSql string
	for _, param := range params {
		execSql, args, err := db.templateBuild(statement, param)
		if err != nil {
			panic(fmt.Errorf("%s->%s", statement, err))
		}
		var stmt *sql.Stmt
		if tempSql != execSql {
			tempSql = execSql
			if s, ok := stmtMaps[execSql]; !ok {
				stmt, err = tdb.PrepareContext(ctx, execSql)
				if err != nil {
					panic(err)
				}
				stmtMaps[execSql] = stmt
			} else {
				stmt = s
			}
		} else {
			stmt = stmtMaps[execSql]
		}
		result, err := stmt.ExecContext(ctx, args...)
		if err != nil {
			panic(fmt.Errorf("%s->%s", statement, err))
		}
		batchAffected, err := result.RowsAffected()
		if err != nil {
			panic(fmt.Errorf("%s->%s", statement, err))
		}
		rowsAffected += int(batchAffected)
	}
	return
}

func (db *DefaultDB) Exec(params any, name ...any) (lastInsertId, rowsAffected int) {
	return db.exec(context.Background(), db.sqlDB, params, name)
}

func (db *DefaultDB) ExecContext(ctx context.Context, params any, name ...any) (lastInsertId, rowsAffected int) {
	return db.exec(ctx, db.sqlDB, params, name)
}
func (db *DefaultDB) PrepareExec(ctx context.Context, params []any, name ...any) (rowsAffected int) {
	return db.prepareExecContext(context.Background(), db.sqlDB, params, name)
}

func (db *DefaultDB) PrepareExecContext(ctx context.Context, params []any, name ...any) (rowsAffected int) {
	return db.prepareExecContext(ctx, db.sqlDB, params, name)
}

func (db *DefaultDB) SelectScanFunc(params any, scanFunc any, name ...any) {
	db.selectScanFunc(context.Background(), db.sqlDB, params, scanFunc, name)
}
func (db *DefaultDB) SelectScanFuncContext(ctx context.Context, params any, scanFunc any, name ...any) {
	db.selectScanFunc(ctx, db.sqlDB, params, scanFunc, name)
}

func (db *DefaultDB) Transaction(tf func(tx *TemplateTxDB) error) (err error) {
	return db.TransactionContext(context.Background(), tf)
}

func (db *DefaultDB) TransactionContext(ctx context.Context, tf func(tx *TemplateTxDB) error) (err error) {
	tx := db.Begin()
	tx.recoverPanic = true
	defer tx.AutoCommit(&err)
	err = tf(tx)
	if err != nil {
		panic(fmt.Errorf("Transaction function inner error:%s", err))
	}
	return
}

func (db *DefaultDB) Begin() *TemplateTxDB {
	return db.BeginTx(context.Background(), nil)
}

func (db *DefaultDB) BeginTx(ctx context.Context, opts *sql.TxOptions) *TemplateTxDB {
	tx, err := db.sqlDB.BeginTx(ctx, opts)
	if err != nil {
		panic(fmt.Errorf("BeginTx error :%s", err))
	}
	return &TemplateTxDB{ActionDB: db, tx: tx, recoverPanic: db.recoverPanic}
}

type TemplateTxDB struct {
	ActionDB
	tx           *sql.Tx
	recoverPanic bool
}

func (tx *TemplateTxDB) AutoCommit(err *error) {
	if *err == nil {
		*err = recover().(error)
	}
	if *err != nil {
		tx.tx.Rollback()
	} else {
		tx.tx.Commit()
	}
	if *err != nil && tx.recoverPanic {
		panic(*err)
	}
}

func (tx *TemplateTxDB) Exec(params any, name ...any) (lastInsertId, rowsAffected int) {
	return tx.exec(context.Background(), tx.tx, params, name)
}

func (tx *TemplateTxDB) ExecContext(ctx context.Context, params any, name ...any) (lastInsertId, rowsAffected int) {
	return tx.exec(ctx, tx.tx, params, name)
}

func (tx *TemplateTxDB) PrepareExec(ctx context.Context, params []any, name ...any) (rowsAffected int) {
	return tx.prepareExecContext(context.Background(), tx.tx, params, name)
}

func (tx *TemplateTxDB) PrepareExecContext(ctx context.Context, params []any, name ...any) (rowsAffected int) {
	return tx.prepareExecContext(ctx, tx.tx, params, name)
}

func (tx *TemplateTxDB) SelectScanFunc(params any, scanFunc any, name ...any) {
	tx.selectScanFunc(context.Background(), tx.tx, params, scanFunc, name)
}
func (tx *TemplateTxDB) SelectScanFuncContext(ctx context.Context, params any, scanFunc any, name ...any) {
	tx.selectScanFunc(ctx, tx.tx, params, scanFunc, name)
}

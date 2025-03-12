// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime/trace"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	"xorm.io/xorm"
	"xorm.io/xorm/contexts"
	"xorm.io/xorm/names"
	"xorm.io/xorm/schemas"

	_ "github.com/go-sql-driver/mysql" // Needed for the MySQL driver
	_ "github.com/lib/pq"              // Needed for the Postgresql driver
)

var (
	x         *xorm.Engine
	tables    []any
	initFuncs []func() error
)

// Engine represents a xorm engine or session.
// (Our Engine interface remains unchanged.)
type Engine interface {
	Table(tableNameOrBean any) *xorm.Session
	Count(...any) (int64, error)
	Decr(column string, arg ...any) *xorm.Session
	Delete(...any) (int64, error)
	Truncate(...any) (int64, error)
	Exec(...any) (sql.Result, error)
	Find(any, ...any) error
	Get(beans ...any) (bool, error)
	ID(any) *xorm.Session
	In(string, ...any) *xorm.Session
	Incr(column string, arg ...any) *xorm.Session
	Insert(...any) (int64, error)
	Iterate(any, xorm.IterFunc) error
	IsTableExist(any) (bool, error)
	Join(joinOperator string, tablename, condition any, args ...any) *xorm.Session
	SQL(any, ...any) *xorm.Session
	Where(any, ...any) *xorm.Session
	Asc(colNames ...string) *xorm.Session
	Desc(colNames ...string) *xorm.Session
	Limit(limit int, start ...int) *xorm.Session
	NoAutoTime() *xorm.Session
	SumInt(bean any, columnName string) (res int64, err error)
	Sync(...any) error
	Select(string) *xorm.Session
	SetExpr(string, any) *xorm.Session
	NotIn(string, ...any) *xorm.Session
	OrderBy(any, ...any) *xorm.Session
	Exist(...any) (bool, error)
	Distinct(...string) *xorm.Session
	Query(...any) ([]map[string][]byte, error)
	Cols(...string) *xorm.Session
	Context(ctx context.Context) *xorm.Session
	Ping() error
}

// TableInfo remains the same – it will call x.TableInfo on the underlying engine or group.
func TableInfo(v any) (*schemas.Table, error) {
	return x.TableInfo(v)
}

// DumpTables dump tables information
func DumpTables(tables []*schemas.Table, w io.Writer, tp ...schemas.DBType) error {
	return x.DumpTables(tables, w, tp...)
}

// RegisterModel registers model, if initfunc provided, it will be invoked after data model sync.
func RegisterModel(bean any, initFunc ...func() error) {
	tables = append(tables, bean)
	if len(initFuncs) > 0 && initFunc[0] != nil {
		initFuncs = append(initFuncs, initFunc[0])
	}
}

func init() {
	gonicNames := []string{"SSL", "UID"}
	for _, name := range gonicNames {
		names.LintGonicMapper[name] = true
	}
}

// newXORMEngineGroup creates an xorm.EngineGroup (with one master and one or more slaves).
// It assumes you have separate master and slave DSNs defined via the settings package.
func newXORMEngineGroup() (Engine, error) {
	// Retrieve master DSN from settings.
	masterConnStr, err := setting.DBMasterConnStr()
	if err != nil {
		return nil, fmt.Errorf("failed to determine master DSN: %w", err)
	}

	var masterEngine *xorm.Engine
	// For PostgreSQL: if a schema is provided, we use the special “postgresschema” driver.
	if setting.Database.Type.IsPostgreSQL() && len(setting.Database.Schema) > 0 {
		registerPostgresSchemaDriver()
		masterEngine, err = xorm.NewEngine("postgresschema", masterConnStr)
	} else {
		masterEngine, err = xorm.NewEngine(setting.Database.Type.String(), masterConnStr)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create master engine: %w", err)
	}
	if setting.Database.Type.IsMySQL() {
		masterEngine.Dialect().SetParams(map[string]string{"rowFormat": "DYNAMIC"})
	}
	masterEngine.SetSchema(setting.Database.Schema)

	// Get slave DSNs.
	slaveConnStrs, err := setting.DBSlaveConnStrs()
	if err != nil {
		return nil, fmt.Errorf("failed to load slave DSNs: %w", err)
	}

	var slaveEngines []*xorm.Engine
	// Iterate over all slave DSNs and create engines.
	for _, dsn := range slaveConnStrs {
		slaveEngine, err := xorm.NewEngine(setting.Database.Type.String(), dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to create slave engine for dsn %q: %w", dsn, err)
		}
		if setting.Database.Type.IsMySQL() {
			slaveEngine.Dialect().SetParams(map[string]string{"rowFormat": "DYNAMIC"})
		}
		slaveEngine.SetSchema(setting.Database.Schema)
		slaveEngines = append(slaveEngines, slaveEngine)
	}

	// Build load balance policy from user settings.
	var policy xorm.GroupPolicy
	switch setting.Database.LoadBalancePolicy {
	case "WeightRandom":
		var weights []int
		if setting.Database.LoadBalanceWeights != "" {
			for part := range strings.SplitSeq(setting.Database.LoadBalanceWeights, ",") {
				w, err := strconv.Atoi(strings.TrimSpace(part))
				if err != nil {
					w = 1 // use a default weight if conversion fails
				}
				weights = append(weights, w)
			}
		}
		// If no valid weights were provided, default each slave to weight 1.
		if len(weights) == 0 {
			weights = make([]int, len(slaveEngines))
			for i := range weights {
				weights[i] = 1
			}
		}
		policy = xorm.WeightRandomPolicy(weights)
	case "RoundRobin":
		policy = xorm.RoundRobinPolicy()
	default:
		policy = xorm.RandomPolicy()
	}
	// Create the EngineGroup using the selected policy.
	group, err := xorm.NewEngineGroup(masterEngine, slaveEngines, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine group: %w", err)
	}
	return engineGroupWrapper{group}, nil
}

type engineGroupWrapper struct {
	*xorm.EngineGroup
}

func (w engineGroupWrapper) AddHook(hook contexts.Hook) bool {
	w.EngineGroup.AddHook(hook)
	return true
}

// SyncAllTables sync the schemas of all tables.
func SyncAllTables() error {
	_, err := x.StoreEngine("InnoDB").SyncWithOptions(xorm.SyncOptions{
		WarnIfDatabaseColumnMissed: true,
	}, tables...)
	return err
}

// InitEngine initializes the xorm EngineGroup and sets it as db.DefaultContext.
func InitEngine(ctx context.Context) error {
	xormEngine, err := newXORMEngineGroup()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	// Try to cast to the concrete type to access diagnostic methods.
	if eng, ok := xormEngine.(engineGroupWrapper); ok {
		eng.SetMapper(names.GonicMapper{})
		// WARNING: for serv command, MUST remove the output to os.Stdout,
		// so use a log file instead of printing to stdout.
		eng.SetLogger(NewXORMLogger(setting.Database.LogSQL))
		eng.ShowSQL(setting.Database.LogSQL)
		eng.SetMaxOpenConns(setting.Database.MaxOpenConns)
		eng.SetMaxIdleConns(setting.Database.MaxIdleConns)
		eng.SetConnMaxLifetime(setting.Database.ConnMaxLifetime)
		eng.SetConnMaxIdleTime(setting.Database.ConnMaxIdleTime)
		eng.SetDefaultContext(ctx)

		if setting.Database.SlowQueryThreshold > 0 {
			eng.AddHook(&SlowQueryHook{
				Treshold: setting.Database.SlowQueryThreshold,
				Logger:   log.GetLogger("xorm"),
			})
		}

		errorLogger := log.GetLogger("xorm")
		if setting.IsInTesting {
			errorLogger = log.GetLogger(log.DEFAULT)
		}

		eng.AddHook(&ErrorQueryHook{
			Logger: errorLogger,
		})

		eng.AddHook(&TracingHook{})

		SetDefaultEngine(ctx, eng)
	} else {
		// Fallback: if type assertion fails, set default engine without extended diagnostics.
		SetDefaultEngine(ctx, xormEngine)
	}
	return nil
}

// SetDefaultEngine sets the default engine for db.
func SetDefaultEngine(ctx context.Context, eng Engine) {
	if engine, ok := eng.(*xorm.Engine); ok {
		x = engine
	} else if group, ok := eng.(engineGroupWrapper); ok {
		x = group.Master()
	}
	DefaultContext = &Context{
		Context: ctx,
		e:       eng,
	}
}

// UnsetDefaultEngine closes and unsets the default engine.
func UnsetDefaultEngine() {
	if x != nil {
		_ = x.Close()
		x = nil
	}
	DefaultContext = nil
}

// InitEngineWithMigration initializes a new xorm EngineGroup, runs migrations, and sets it as db.DefaultContext.
func InitEngineWithMigration(ctx context.Context, migrateFunc func(Engine) error) (err error) {
	if err = InitEngine(ctx); err != nil {
		return err
	}

	if err = x.Ping(); err != nil {
		return err
	}

	preprocessDatabaseCollation(x)

	// Run migration function.
	if err = migrateFunc(x); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	if err = SyncAllTables(); err != nil {
		return fmt.Errorf("sync database struct error: %w", err)
	}

	for _, initFunc := range initFuncs {
		if err := initFunc(); err != nil {
			return fmt.Errorf("initFunc failed: %w", err)
		}
	}

	return nil
}

// NamesToBean returns a list of beans given names.
func NamesToBean(names ...string) ([]any, error) {
	beans := []any{}
	if len(names) == 0 {
		beans = append(beans, tables...)
		return beans, nil
	}
	// Map provided names to beans.
	beanMap := make(map[string]any)
	for _, bean := range tables {
		beanMap[strings.ToLower(reflect.Indirect(reflect.ValueOf(bean)).Type().Name())] = bean
		beanMap[strings.ToLower(x.TableName(bean))] = bean
		beanMap[strings.ToLower(x.TableName(bean, true))] = bean
	}

	gotBean := make(map[any]bool)
	for _, name := range names {
		bean, ok := beanMap[strings.ToLower(strings.TrimSpace(name))]
		if !ok {
			return nil, fmt.Errorf("no table found that matches: %s", name)
		}
		if !gotBean[bean] {
			beans = append(beans, bean)
			gotBean[bean] = true
		}
	}
	return beans, nil
}

// DumpDatabase dumps all data from database using special SQL syntax to the file system.
func DumpDatabase(filePath, dbType string) error {
	var tbs []*schemas.Table
	for _, t := range tables {
		t, err := x.TableInfo(t)
		if err != nil {
			return err
		}
		tbs = append(tbs, t)
	}

	type Version struct {
		ID      int64 `xorm:"pk autoincr"`
		Version int64
	}
	t, err := x.TableInfo(&Version{})
	if err != nil {
		return err
	}
	tbs = append(tbs, t)

	if len(dbType) > 0 {
		return x.DumpTablesToFile(tbs, filePath, schemas.DBType(dbType))
	}
	return x.DumpTablesToFile(tbs, filePath)
}

// MaxBatchInsertSize returns the table's max batch insert size.
func MaxBatchInsertSize(bean any) int {
	t, err := x.TableInfo(bean)
	if err != nil {
		return 50
	}
	return 999 / len(t.ColumnsSeq())
}

// IsTableNotEmpty returns true if the table has at least one record.
func IsTableNotEmpty(beanOrTableName any) (bool, error) {
	return x.Table(beanOrTableName).Exist()
}

// DeleteAllRecords deletes all records in the given table.
func DeleteAllRecords(tableName string) error {
	_, err := x.Exec(fmt.Sprintf("DELETE FROM %s", tableName))
	return err
}

// GetMaxID returns the maximum id in the table.
func GetMaxID(beanOrTableName any) (maxID int64, err error) {
	_, err = x.Select("MAX(id)").Table(beanOrTableName).Get(&maxID)
	return maxID, err
}

func SetLogSQL(ctx context.Context, on bool) {
	e := GetEngine(ctx)
	if eng, ok := e.(*xorm.Engine); ok {
		eng.ShowSQL(on)
	} else if sess, ok := e.(*xorm.Session); ok {
		sess.Engine().ShowSQL(on)
	}
}

type TracingHook struct{}

var _ contexts.Hook = &TracingHook{}

type sqlTask struct{}

func (TracingHook) BeforeProcess(c *contexts.ContextHook) (context.Context, error) {
	ctx, task := trace.NewTask(c.Ctx, "sql")
	ctx = context.WithValue(ctx, sqlTask{}, task)
	trace.Log(ctx, "query", c.SQL)
	trace.Logf(ctx, "args", "%v", c.Args)
	return ctx, nil
}

func (TracingHook) AfterProcess(c *contexts.ContextHook) error {
	c.Ctx.Value(sqlTask{}).(*trace.Task).End()
	return nil
}

type SlowQueryHook struct {
	Treshold time.Duration
	Logger   log.Logger
}

var _ contexts.Hook = &SlowQueryHook{}

func (SlowQueryHook) BeforeProcess(c *contexts.ContextHook) (context.Context, error) {
	return c.Ctx, nil
}

func (h *SlowQueryHook) AfterProcess(c *contexts.ContextHook) error {
	if c.ExecuteTime >= h.Treshold {
		h.Logger.Log(8, log.WARN, "[Slow SQL Query] %s %v - %v", c.SQL, c.Args, c.ExecuteTime)
	}
	return nil
}

type ErrorQueryHook struct {
	Logger log.Logger
}

var _ contexts.Hook = &ErrorQueryHook{}

func (ErrorQueryHook) BeforeProcess(c *contexts.ContextHook) (context.Context, error) {
	return c.Ctx, nil
}

func (h *ErrorQueryHook) AfterProcess(c *contexts.ContextHook) error {
	if c.Err != nil && !errors.Is(c.Err, context.Canceled) {
		h.Logger.Log(8, log.ERROR, "[Error SQL Query] %s %v - %v", c.SQL, c.Args, c.Err)
	}
	return nil
}

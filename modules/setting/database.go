// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"forgejo.org/modules/log"

	"xorm.io/xorm"
)

var (
	// SupportedDatabaseTypes includes all XORM supported databases type, sqlite3 maybe added by `database_sqlite3.go`
	SupportedDatabaseTypes = []string{"mysql", "postgres"}
	// DatabaseTypeNames contains the friendly names for all database types
	DatabaseTypeNames = map[string]string{"mysql": "MySQL", "postgres": "PostgreSQL", "sqlite3": "SQLite3"}

	// EnableSQLite3 use SQLite3, set by build flag
	EnableSQLite3 bool

	// Database holds the database settings
	Database = DatabaseSettings{
		Timeout:           500,
		IterateBufferSize: 50,
	}
)

type DatabaseSettings struct {
	Type               DatabaseType
	Host               string
	HostPrimary        string
	HostReplica        string
	LoadBalancePolicy  string
	LoadBalanceWeights string
	Name               string
	User               string
	Passwd             string
	Schema             string
	SSLMode            string
	Path               string
	LogSQL             bool
	MysqlCharset       string
	CharsetCollation   string
	Timeout            int // seconds
	SQLiteJournalMode  string
	DBConnectRetries   int
	DBConnectBackoff   time.Duration
	MaxIdleConns       int
	MaxOpenConns       int
	ConnMaxIdleTime    time.Duration
	ConnMaxLifetime    time.Duration
	IterateBufferSize  int
	AutoMigration      bool
	SlowQueryThreshold time.Duration
}

// LoadDBSetting loads the database settings
func LoadDBSetting() {
	loadDBSetting(CfgProvider)
}

func loadDBSetting(rootCfg ConfigProvider) {
	sec := rootCfg.Section("database")
	Database.Type = DatabaseType(sec.Key("DB_TYPE").String())

	Database.Host = sec.Key("HOST").String()
	Database.HostPrimary = sec.Key("HOST_PRIMARY").String()
	Database.HostReplica = sec.Key("HOST_REPLICA").String()
	Database.LoadBalancePolicy = sec.Key("LOAD_BALANCE_POLICY").String()
	Database.LoadBalanceWeights = sec.Key("LOAD_BALANCE_WEIGHTS").String()
	Database.Name = sec.Key("NAME").String()
	Database.User = sec.Key("USER").String()
	if len(Database.Passwd) == 0 {
		Database.Passwd = sec.Key("PASSWD").String()
	}
	Database.Schema = sec.Key("SCHEMA").String()
	Database.SSLMode = sec.Key("SSL_MODE").MustString("disable")
	Database.CharsetCollation = sec.Key("CHARSET_COLLATION").String()

	Database.Path = sec.Key("PATH").MustString(filepath.Join(AppDataPath, "forgejo.db"))
	Database.Timeout = sec.Key("SQLITE_TIMEOUT").MustInt(500)
	Database.SQLiteJournalMode = sec.Key("SQLITE_JOURNAL_MODE").MustString("")

	Database.MaxIdleConns = sec.Key("MAX_IDLE_CONNS").MustInt(2)
	if Database.Type.IsMySQL() {
		Database.ConnMaxLifetime = sec.Key("CONN_MAX_LIFETIME").MustDuration(3 * time.Second)
	} else {
		Database.ConnMaxLifetime = sec.Key("CONN_MAX_LIFETIME").MustDuration(0)
	}
	Database.ConnMaxIdleTime = sec.Key("CONN_MAX_IDLETIME").MustDuration(0)
	Database.MaxOpenConns = sec.Key("MAX_OPEN_CONNS").MustInt(100)

	Database.IterateBufferSize = sec.Key("ITERATE_BUFFER_SIZE").MustInt(50)
	Database.LogSQL = sec.Key("LOG_SQL").MustBool(false)
	Database.DBConnectRetries = sec.Key("DB_RETRIES").MustInt(10)
	Database.DBConnectBackoff = sec.Key("DB_RETRY_BACKOFF").MustDuration(3 * time.Second)
	Database.AutoMigration = sec.Key("AUTO_MIGRATION").MustBool(true)

	deprecatedSetting(rootCfg, "database", "SLOW_QUERY_TRESHOLD", "database", "SLOW_QUERY_THRESHOLD", "1.23")
	if sec.HasKey("SLOW_QUERY_TRESHOLD") && !sec.HasKey("SLOW_QUERY_THRESHOLD") {
		Database.SlowQueryThreshold = sec.Key("SLOW_QUERY_TRESHOLD").MustDuration(5 * time.Second)
	} else {
		Database.SlowQueryThreshold = sec.Key("SLOW_QUERY_THRESHOLD").MustDuration(5 * time.Second)
	}
}

// DBMasterConnStr returns the connection string for the master (primary) database.
// If a primary host is defined in the configuration, it is used;
// otherwise, it falls back to Database.Host.
// Returns an error if no master host is provided but a slave is defined.
func DBMasterConnStr() (string, error) {
	var host string
	if Database.HostPrimary != "" {
		host = Database.HostPrimary
	} else {
		host = Database.Host
	}
	if host == "" && Database.HostReplica != "" {
		return "", errors.New("master host is not defined while slave is defined; cannot proceed")
	}

	// For SQLite, no host is needed
	if host == "" && !Database.Type.IsSQLite3() {
		return "", errors.New("no database host defined")
	}

	return dbConnStrWithHost(host)
}

// DBSlaveConnStrs returns one or more connection strings for the replica databases.
// If a replica host is defined (possibly as a comma-separated list) then those DSNs are returned.
// Otherwise, this function falls back to the master DSN (with a warning log).
func DBSlaveConnStrs() ([]string, error) {
	var dsns []string
	if Database.HostReplica != "" {
		// support multiple replica hosts separated by commas
		replicas := strings.SplitSeq(Database.HostReplica, ",")
		for r := range replicas {
			trimmed := strings.TrimSpace(r)
			if trimmed == "" {
				continue
			}
			dsn, err := dbConnStrWithHost(trimmed)
			if err != nil {
				return nil, err
			}
			dsns = append(dsns, dsn)
		}
	}
	// Fall back to master if no slave DSN was provided.
	if len(dsns) == 0 {
		master, err := DBMasterConnStr()
		if err != nil {
			return nil, err
		}
		log.Debug("Database: No dedicated replica host defined; falling back to primary DSN for replica connections")
		dsns = append(dsns, master)
	}
	return dsns, nil
}

func BuildLoadBalancePolicy(settings *DatabaseSettings, slaveEngines []*xorm.Engine) xorm.GroupPolicy {
	var policy xorm.GroupPolicy
	switch settings.LoadBalancePolicy { // Use the settings parameter directly
	case "WeightRandom":
		var weights []int
		if settings.LoadBalanceWeights != "" { // Use the settings parameter directly
			for part := range strings.SplitSeq(settings.LoadBalanceWeights, ",") {
				w, err := strconv.Atoi(strings.TrimSpace(part))
				if err != nil {
					w = 1 // use a default weight if conversion fails
				}
				weights = append(weights, w)
			}
		}
		// If no valid weights were provided, default each slave to weight 1
		if len(weights) == 0 {
			weights = make([]int, len(slaveEngines))
			for i := range weights {
				weights[i] = 1
			}
		}
		policy = xorm.WeightRandomPolicy(weights)
	case "WeightRoundRobin":
		var weights []int
		if settings.LoadBalanceWeights != "" {
			for part := range strings.SplitSeq(settings.LoadBalanceWeights, ",") {
				w, err := strconv.Atoi(strings.TrimSpace(part))
				if err != nil {
					w = 1 // use a default weight if conversion fails
				}
				weights = append(weights, w)
			}
		}
		// If no valid weights were provided, default each slave to weight 1
		if len(weights) == 0 {
			weights = make([]int, len(slaveEngines))
			for i := range weights {
				weights[i] = 1
			}
		}
		policy = xorm.WeightRoundRobinPolicy(weights)
	case "RoundRobin":
		policy = xorm.RoundRobinPolicy()
	case "LeastConn":
		policy = xorm.LeastConnPolicy()
	default:
		policy = xorm.RandomPolicy()
	}
	return policy
}

// dbConnStrWithHost constructs the connection string, given a host value.
func dbConnStrWithHost(host string) (string, error) {
	var connStr string
	paramSep := "?"
	if strings.Contains(Database.Name, paramSep) {
		paramSep = "&"
	}
	switch Database.Type {
	case "mysql":
		connType := "tcp"
		// if the host starts with '/' it is assumed to be a unix socket path
		if len(host) > 0 && host[0] == '/' {
			connType = "unix"
		}
		tls := Database.SSLMode
		// allow the "disable" value (borrowed from Postgres defaults) to behave as false
		if tls == "disable" {
			tls = "false"
		}
		connStr = fmt.Sprintf("%s:%s@%s(%s)/%s%sparseTime=true&tls=%s",
			Database.User, Database.Passwd, connType, host, Database.Name, paramSep, tls)
	case "postgres":
		connStr = getPostgreSQLConnectionString(host, Database.User, Database.Passwd, Database.Name, Database.SSLMode)
	case "sqlite3":
		if !EnableSQLite3 {
			return "", errors.New("this Gitea binary was not built with SQLite3 support")
		}
		if err := os.MkdirAll(filepath.Dir(Database.Path), os.ModePerm); err != nil {
			return "", fmt.Errorf("failed to create directories: %w", err)
		}
		journalMode := ""
		if Database.SQLiteJournalMode != "" {
			journalMode = "&_journal_mode=" + Database.SQLiteJournalMode
		}
		connStr = fmt.Sprintf("file:%s?cache=shared&mode=rwc&_busy_timeout=%d&_txlock=immediate%s",
			Database.Path, Database.Timeout, journalMode)
	default:
		return "", fmt.Errorf("unknown database type: %s", Database.Type)
	}
	return connStr, nil
}

// parsePostgreSQLHostPort parses given input in various forms defined in
// https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING
// and returns proper host and port number.
func parsePostgreSQLHostPort(info string) (host, port string) {
	if h, p, err := net.SplitHostPort(info); err == nil {
		host, port = h, p
	} else {
		// treat the "info" as "host", if it's an IPv6 address, remove the wrapper
		host = info
		if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
			host = host[1 : len(host)-1]
		}
	}

	// set fallback values
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "5432"
	}
	return host, port
}

func getPostgreSQLConnectionString(dbHost, dbUser, dbPasswd, dbName, dbsslMode string) (connStr string) {
	dbName, dbParam, _ := strings.Cut(dbName, "?")
	host, port := parsePostgreSQLHostPort(dbHost)
	connURL := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbUser, dbPasswd),
		Host:     net.JoinHostPort(host, port),
		Path:     dbName,
		OmitHost: false,
		RawQuery: dbParam,
	}
	query := connURL.Query()
	if strings.HasPrefix(host, "/") { // looks like a unix socket
		query.Add("host", host)
		connURL.Host = ":" + port
	}
	query.Set("sslmode", dbsslMode)
	connURL.RawQuery = query.Encode()
	return connURL.String()
}

type DatabaseType string

func (t DatabaseType) String() string {
	return string(t)
}

func (t DatabaseType) IsSQLite3() bool {
	return t == "sqlite3"
}

func (t DatabaseType) IsMySQL() bool {
	return t == "mysql"
}

func (t DatabaseType) IsPostgreSQL() bool {
	return t == "postgres"
}

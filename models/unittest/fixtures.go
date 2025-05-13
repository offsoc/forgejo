// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//nolint:forbidigo
package unittest

import (
	"fmt"
	"path/filepath"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/modules/auth/password/hash"
	"forgejo.org/modules/setting"

	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

var fixturesLoader *loader

// GetXORMEngine gets the XORM engine
func GetXORMEngine(engine ...*xorm.Engine) (x *xorm.Engine, err error) {
	if len(engine) == 1 {
		return engine[0], nil
	}
	return db.GetMasterEngine(db.DefaultContext.(*db.Context).Engine())
}

func OverrideFixtures(dir string) func() {
	old := fixturesLoader

	opts := FixturesOptions{
		Dir:  filepath.Join(setting.AppWorkPath, "models/fixtures/"),
		Base: setting.AppWorkPath,
		Dirs: []string{dir},
	}
	if err := InitFixtures(opts); err != nil {
		panic(err)
	}

	return func() {
		fixturesLoader = old
	}
}

// InitFixtures initialize test fixtures for a test database
func InitFixtures(opts FixturesOptions, engine ...*xorm.Engine) (err error) {
	e, err := GetXORMEngine(engine...)
	if err != nil {
		return err
	}

	fixturePaths := []string{}
	if opts.Dir != "" {
		fixturePaths = append(fixturePaths, opts.Dir)
	} else {
		fixturePaths = append(fixturePaths, opts.Files...)
	}
	if opts.Dirs != nil {
		for _, dir := range opts.Dirs {
			fixturePaths = append(fixturePaths, filepath.Join(opts.Base, dir))
		}
	}

	var dialect string
	switch e.Dialect().URI().DBType {
	case schemas.POSTGRES:
		dialect = "postgres"
	case schemas.MYSQL:
		dialect = "mysql"
	case schemas.SQLITE:
		dialect = "sqlite3"
	default:
		panic("Unsupported RDBMS for test")
	}

	fixturesLoader, err = newFixtureLoader(e.DB().DB, dialect, fixturePaths)
	if err != nil {
		return err
	}

	// register the dummy hash algorithm function used in the test fixtures
	_ = hash.Register("dummy", hash.NewDummyHasher)

	setting.PasswordHashAlgo, _ = hash.SetDefaultPasswordHashAlgorithm("dummy")

	return err
}

// LoadFixtures load fixtures for a test database
func LoadFixtures(engine ...*xorm.Engine) error {
	e, err := GetXORMEngine(engine...)
	if err != nil {
		return err
	}
	// (doubt) database transaction conflicts could occur and result in ROLLBACK? just try for a few times.
	for range 5 {
		if err = fixturesLoader.Load(); err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		fmt.Printf("LoadFixtures failed after retries: %v\n", err)
	}
	// Now if we're running postgres we need to tell it to update the sequences
	if e.Dialect().URI().DBType == schemas.POSTGRES {
		results, err := e.QueryString(`SELECT 'SELECT SETVAL(' ||
		quote_literal(quote_ident(PGT.schemaname) || '.' || quote_ident(S.relname)) ||
		', COALESCE(MAX(' ||quote_ident(C.attname)|| '), 1) ) FROM ' ||
		quote_ident(PGT.schemaname)|| '.'||quote_ident(T.relname)|| ';'
	 FROM pg_class AS S,
	      pg_depend AS D,
	      pg_class AS T,
	      pg_attribute AS C,
	      pg_tables AS PGT
	 WHERE S.relkind = 'S'
	     AND S.oid = D.objid
	     AND D.refobjid = T.oid
	     AND D.refobjid = C.attrelid
	     AND D.refobjsubid = C.attnum
	     AND T.relname = PGT.tablename
	 ORDER BY S.relname;`)
		if err != nil {
			fmt.Printf("Failed to generate sequence update: %v\n", err)
			return err
		}
		for _, r := range results {
			for _, value := range r {
				_, err = e.Exec(value)
				if err != nil {
					fmt.Printf("Failed to update sequence: %s Error: %v\n", value, err)
					return err
				}
			}
		}
	}
	_ = hash.Register("dummy", hash.NewDummyHasher)
	setting.PasswordHashAlgo, _ = hash.SetDefaultPasswordHashAlgorithm("dummy")

	return err
}

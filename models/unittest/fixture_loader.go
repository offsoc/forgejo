// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package unittest

import (
	"database/sql"
	"encoding/hex"
	"encoding/json" //nolint:depguard
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type insertSQL struct {
	statement string
	values    []any
}

type fixtureFile struct {
	name       string
	insertSQLs []insertSQL
}

type loader struct {
	db      *sql.DB
	dialect string

	fixtureFiles []*fixtureFile
}

func newFixtureLoader(db *sql.DB, dialect string, fixturePaths []string) (*loader, error) {
	l := &loader{
		db:           db,
		dialect:      dialect,
		fixtureFiles: []*fixtureFile{},
	}

	// Load fixtures
	for _, fixturePath := range fixturePaths {
		stat, err := os.Stat(fixturePath)
		if err != nil {
			return nil, err
		}

		// If fixture path is a directory, then read read the files of the directory
		// and use those as fixture files.
		if stat.IsDir() {
			files, err := os.ReadDir(fixturePath)
			if err != nil {
				return nil, err
			}
			for _, file := range files {
				if !file.IsDir() {
					fixtureFile, err := l.buildFixtureFile(filepath.Join(fixturePath, file.Name()))
					if err != nil {
						return nil, err
					}
					l.fixtureFiles = append(l.fixtureFiles, fixtureFile)
				}
			}
		} else {
			fixtureFile, err := l.buildFixtureFile(fixturePath)
			if err != nil {
				return nil, err
			}
			l.fixtureFiles = append(l.fixtureFiles, fixtureFile)
		}
	}

	return l, nil
}

// quoteKeyword returns the quoted string of keyword.
func (l *loader) quoteKeyword(keyword string) string {
	switch l.dialect {
	case "sqlite3":
		return `"` + keyword + `"`
	case "mysql":
		return "`" + keyword + "`"
	case "postgres":
		parts := strings.Split(keyword, ".")
		for i, p := range parts {
			parts[i] = `"` + p + `"`
		}
		return strings.Join(parts, ".")
	default:
		return "invalid"
	}
}

// placeholder returns the placeholder string.
func (l *loader) placeholder(index int) string {
	if l.dialect == "postgres" {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

func (l *loader) buildFixtureFile(fixturePath string) (*fixtureFile, error) {
	f, err := os.Open(fixturePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []map[string]any
	if err := yaml.NewDecoder(f).Decode(&records); err != nil {
		return nil, err
	}

	fixture := &fixtureFile{
		name:       filepath.Base(strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))),
		insertSQLs: []insertSQL{},
	}

	for _, record := range records {
		columns := []string{}
		sqlValues := []string{}
		values := []any{}
		i := 1

		for key, value := range record {
			columns = append(columns, l.quoteKeyword(key))

			switch v := value.(type) {
			case string:
				// Try to decode hex.
				if strings.HasPrefix(v, "0x") {
					value, err = hex.DecodeString(strings.TrimPrefix(v, "0x"))
					if err != nil {
						return nil, err
					}
				}
			case []any:
				// Decode array.
				var bytes []byte
				bytes, err = json.Marshal(v)
				if err != nil {
					return nil, err
				}
				value = string(bytes)
			}

			values = append(values, value)

			sqlValues = append(sqlValues, l.placeholder(i))
			i++
		}

		// Construct the insert SQL.
		fixture.insertSQLs = append(fixture.insertSQLs, insertSQL{
			statement: fmt.Sprintf(
				"INSERT INTO %s (%s) VALUES (%s)",
				l.quoteKeyword(fixture.name),
				strings.Join(columns, ", "),
				strings.Join(sqlValues, ", "),
			),
			values: values,
		})
	}

	return fixture, nil
}

func (l *loader) Load() error {
	// Start transaction.
	tx, err := l.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// Clean the table and re-insert the fixtures.
	tableDeleted := map[string]struct{}{}
	for _, fixture := range l.fixtureFiles {
		if _, ok := tableDeleted[fixture.name]; !ok {
			if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s", l.quoteKeyword(fixture.name))); err != nil {
				return fmt.Errorf("cannot delete table %s: %w", fixture.name, err)
			}
			tableDeleted[fixture.name] = struct{}{}
		}

		for _, insertSQL := range fixture.insertSQLs {
			if _, err := tx.Exec(insertSQL.statement, insertSQL.values...); err != nil {
				return fmt.Errorf("cannot insert %q with values %q: %w", insertSQL.statement, insertSQL.values, err)
			}
		}
	}

	return tx.Commit()
}

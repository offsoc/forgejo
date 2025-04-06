// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package db

import (
	"fmt"
	"strconv"
	"strings"

	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"

	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

// ConvertDatabaseTable converts database and tables from utf8 to utf8mb4 if it's mysql and set ROW_FORMAT=dynamic
func ConvertDatabaseTable() error {
	if x.Dialect().URI().DBType != schemas.MYSQL {
		return nil
	}

	r, err := CheckCollations(x)
	if err != nil {
		return err
	}

	databaseName := strings.SplitN(setting.Database.Name, "?", 2)[0]
	_, err = x.Exec(fmt.Sprintf("ALTER DATABASE `%s` CHARACTER SET utf8mb4 COLLATE %s", databaseName, r.ExpectedCollation))
	if err != nil {
		return err
	}

	tables, err := x.DBMetas()
	if err != nil {
		return err
	}
	for _, table := range tables {
		if _, err := x.Exec(fmt.Sprintf("ALTER TABLE `%s` ROW_FORMAT=dynamic", table.Name)); err != nil {
			return err
		}

		if _, err := x.Exec(fmt.Sprintf("ALTER TABLE `%s` CONVERT TO CHARACTER SET utf8mb4 COLLATE %s", table.Name, r.ExpectedCollation)); err != nil {
			return err
		}
	}

	return nil
}

// Cell2Int64 converts a xorm.Cell type to int64,
// and handles possible irregular cases.
func Cell2Int64(val xorm.Cell) int64 {
	switch (*val).(type) {
	case []uint8:
		log.Trace("Cell2Int64 ([]uint8): %v", *val)

		v, _ := strconv.ParseInt(string((*val).([]uint8)), 10, 64)
		return v
	default:
		return (*val).(int64)
	}
}

// Package vars contains the database variables such as dynamic table names
package vars

import "github.com/flashbots/builder-config-hub/common"

var (
	tablePrefix = common.GetEnv("DB_TABLE_PREFIX", "dev")

	TableMigrations = tablePrefix + "_migrations"
	TableTest       = tablePrefix + "_test"
)
package database_admin //nolint:revive,stylecheck

import (
	"app/base/utils"
)

var (
	// Schema to migrate to (-1 means latest)
	schemaMigration = utils.PodConfig.GetInt("schema_migration", -1)
	// Put this version into schema_migration table and set dirty=false
	forceMigrationVersion = utils.PodConfig.GetInt("force_migration_version", -1)
	// Drop everything and create schema from scratch
	resetSchema = utils.PodConfig.GetBool("reset_schema", false)
	// Create users and update their password
	updateUsers = utils.PodConfig.GetBool("update_users", false)
	// Reset cyndi password
	updateCyndiPasswd = utils.PodConfig.GetBool("update_cyndi_passwd", false)
	// rerun config.sql
	updateDBConfig = utils.PodConfig.GetBool("update_db_config", false)
)

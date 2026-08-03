package database_admin

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
	// Unlock users
	unlockUsers = utils.PodConfig.GetBool("unlock_users", false)
	// rerun config.sql
	updateDBConfig = utils.PodConfig.GetBool("update_db_config", false)
	// Terminate lockUsers sessions after NOLOGIN (for major DDL migrations)
	terminateDBSessions = utils.PodConfig.GetBool("terminate_db_sessions", false)
	// One-off: truncate corrupt system_advisories_0 and clear bucket-0 advisory caches
	repairSystemAdvisories0 = utils.PodConfig.GetBool("repair_system_advisories_0", false)
)

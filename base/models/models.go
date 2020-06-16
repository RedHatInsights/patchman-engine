package models

import (
	"github.com/jinzhu/gorm/dialects/postgres"
	"time"
)

type RhAccount struct {
	ID   int
	Name string
}

func (RhAccount) TableName() string {
	return "rh_account"
}

// nolint: maligned
type SystemPlatform struct {
	ID          int    `gorm:"primary_key"`
	InventoryID string `sql:"unique" gorm:"unique"`
	RhAccountID int
	// All times need to be stored as pointers, since they are set to 0000-00-00 00:00 by gorm if not present
	FirstReported *time.Time `gorm:"default:null"`
	VmaasJSON     string
	JSONChecksum  string

	LastUpdated           *time.Time `gorm:"default:null"`
	UnchangedSince        *time.Time `gorm:"default:null"`
	LastEvaluation        *time.Time `gorm:"default:null"`
	OptOut                bool
	AdvisoryCountCache    int
	AdvisoryEnhCountCache int
	AdvisoryBugCountCache int
	AdvisorySecCountCache int
	LastUpload            *time.Time `gorm:"default:null"`
	StaleTimestamp        *time.Time
	StaleWarningTimestamp *time.Time
	CulledTimestamp       *time.Time
	Stale                 bool
	DisplayName           string

	PackageData *postgres.Jsonb
}

func (SystemPlatform) TableName() string {
	return "system_platform"
}

type SystemPackageData map[string]SystemPackageDataItem
type SystemPackageDataUpdate struct {
	Version  string `json:"version"`
	Advisory string `json:"advisory"`
}
type SystemPackageDataItem struct {
	Version string                    `json:"version"`
	Updates []SystemPackageDataUpdate `json:"updates"`
}

type DeletedSystem struct {
	InventoryID string
	WhenDeleted time.Time
}

func (DeletedSystem) TableName() string {
	return "deleted_system"
}

type AdvisorySeverity struct {
	ID   int
	Name string
}

func (AdvisorySeverity) TableName() string {
	return "advisory_severity"
}

type AdvisoryType struct {
	ID   int
	Name string
}

func (AdvisoryType) TableName() string {
	return "advisory_type"
}

type AdvisoryPackageData map[string]string

type AdvisoryMetadata struct {
	ID             int
	Name           string
	Description    string
	Synopsis       string
	Summary        string
	Solution       string
	AdvisoryTypeID int
	PublicDate     time.Time
	ModifiedDate   time.Time
	URL            *string
	SeverityID     *int
	PackageData    *postgres.Jsonb
	CveList        *string
}

func (AdvisoryMetadata) TableName() string {
	return "advisory_metadata"
}

type AdvisoryMetadataSlice []AdvisoryMetadata

type SystemAdvisories struct {
	SystemID      int `gorm:"primary_key"`
	AdvisoryID    int `gorm:"primary_key"`
	Advisory      AdvisoryMetadata
	FirstReported *time.Time
	WhenPatched   *time.Time
	StatusID      *int
}

func (SystemAdvisories) TableName() string {
	return "system_advisories"
}

type SystemAdvisoriesSlice []SystemAdvisories

type AdvisoryAccountData struct {
	AdvisoryID             int
	RhAccountID            int
	StatusID               int
	SystemsAffected        int
	SystemsStatusDivergent int
}

func (AdvisoryAccountData) TableName() string {
	return "advisory_account_data"
}

type AdvisoryAccountDataSlice []AdvisoryAccountData
type Repo struct {
	ID   int
	Name string
}

func (Repo) TableName() string {
	return "repo"
}

type RepoSlice []Repo

type SystemRepo struct {
	SystemID int
	RepoID   int
}

func (SystemRepo) TableName() string {
	return "system_repo"
}

type SystemRepoSlice []SystemRepo

type TimestampKV struct {
	Name  string
	Value time.Time
}

func (TimestampKV) TableName() string {
	return "timestamp_kv"
}

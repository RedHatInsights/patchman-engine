package models

import (
	"time"
)

type RhAccount struct {
	ID   int
	Name string
}

func (RhAccount) TableName() string {
	return "rh_account"
}

type SystemPlatform struct {
	ID          int
	InventoryID string
	RhAccountID int
	// All times need to be stored as pointers, since they are set to 0000-00-00 00:00 by gorm if not present
	FirstReported         *time.Time
	VmaasJSON             string
	JSONChecksum          string
	LastUpdated           *time.Time
	UnchangedSince        *time.Time
	LastEvaluation        *time.Time
	OptOut                bool
	AdvisoryCountCache    int
	AdvisoryEnhCountCache int
	AdvisoryBugCountCache int
	AdvisorySecCountCache int
	LastUpload            *time.Time
}

func (SystemPlatform) TableName() string {
	return "system_platform"
}

type AdvisoryType struct {
	ID   int
	Name string
}

func (AdvisoryType) TableName() string {
	return "advisory_type"
}

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
}

func (AdvisoryMetadata) TableName() string {
	return "advisory_metadata"
}

type AdvisoryMetadataSlice []AdvisoryMetadata

type SystemAdvisories struct {
	ID            int
	SystemID      int
	AdvisoryID    int
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

type SystemRepo struct {
	SystemID int
	RepoID   int
}

func (SystemRepo) TableName() string {
	return "system_repo"
}

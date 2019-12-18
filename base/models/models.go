package models

import "time"

type RhAccount struct {
	ID   int
	Name string
}

func (RhAccount) TableName() string {
	return "rh_account"
}

type SystemPlatform struct {
	ID                    int
	InventoryID           string
	RhAccountID           int
	FirstReported         time.Time
	VmaasJSON             string
	JsonChecksum          string
	LastUpdated           time.Time
	UnchangedSince        time.Time
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

type AdvisoryMetadata struct {
	ID             int
	Name           string
	Description    string
	Synopsis       string
	Topic          string
	Solution       string
	AdvisoryTypeId int
	PublicDate     time.Time
	ModifiedDate   time.Time
	Url            *string
}

func (AdvisoryMetadata) TableName() string {
	return "advisory_metadata"
}

type SystemAdvisories struct {
	ID            int
	SystemID      int
	AdvisoryID    int
	FirstReported time.Time
	WhenPatched   *time.Time
	StatusId      *int
	StatusText    *string
}

func (SystemAdvisories) TableName() string {
	return "system_advisories"
}

type AdvisoryAccountData struct {
	AdvisoryID  int
	RhAccountID int
}

func (AdvisoryAccountData) TableName() string {
	return "advisory_account_data"
}

type Repo struct {
	ID   int
	Name string
}

func (Repo) TableName() string {
	return "repo"
}

type SystemRepo struct {
	SystemID   int
	RepoId     int
	StatusID   int
	StatusText *string
}

func (SystemRepo) TableName() string {
	return "system_repo"
}

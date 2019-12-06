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
	ID               int
	InventoryID      string
	RhAccountID      int
	FirstReported    time.Time
	S3Url            string
	VmaasJson        string
	JsonChecksum     string
	LastUpdated      time.Time
	UnchangedSince   time.Time
	LastEvaluation   *time.Time
	OptOut           bool
	ErrataCountCache int
	LastUpload       *time.Time
}

func (SystemPlatform) TableName() string {
	return "system_platform"
}

type ErrataMetaData struct {
	ID           int
	Advisory     string
	AdvisoryName string
	Description  string
	Synopsis     string
	Topic        string
	Solution     string
	ErrataTypeId int
	PublicDate   time.Time
	ModifiedDate time.Time
	Url          *string
}

func (ErrataMetaData) TableName() string {
	return "errata_metadata"
}

type SystemAdvisories struct {
	ID            int
	SystemID      int
	ErrataID      int
	FirstReported time.Time
	WhenPatched   *time.Time
	StatusId      *int
	StatusText    *string
}


func (SystemAdvisories) TableName() string {
	return "system_advisories"
}


type ErrataAccountData struct {
	ErrataID    int
	RhAccountID int
}


func (ErrataAccountData) TableName() string {
	return "errata_account_data"
}


type Repo struct {
	ID   int
	Name string
}


func (Repo) TableName() string {
	return "repo"
}


type SystemRepo struct {
	SystemID int
	RepoId   int
	StatusID int
	StatusText *string

}


func (SystemRepo) TableName() string {
	return "system_repo"
}




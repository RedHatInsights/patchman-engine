package models

import (
	"time"
)

type RhAccount struct {
	ID                 int `gorm:"primaryKey"`
	Name               *string
	OrgID              *string
	ValidPackageCache  bool
	ValidAdvisoryCache bool
}

func (RhAccount) TableName() string {
	return "rh_account"
}

type Reporter struct {
	ID   int `gorm:"primaryKey"`
	Name string
}

func (Reporter) TableName() string {
	return "reporter"
}

type Baseline struct {
	ID          int64 `gorm:"primaryKey"`
	RhAccountID int   `gorm:"primaryKey"`
	Name        string
	Config      []byte
	Description *string
	Creator     *string // pointer for compatibility with previous API versions
	Published   *time.Time
	LastEdited  *time.Time
}

func (Baseline) TableName() string {
	return "baseline"
}

// nolint: maligned
type SystemPlatform struct {
	ID                    int64  `gorm:"primaryKey"`
	InventoryID           string `gorm:"unique"`
	RhAccountID           int    `gorm:"primaryKey"`
	VmaasJSON             *string
	JSONChecksum          *string
	LastUpdated           *time.Time `gorm:"default:null"`
	UnchangedSince        *time.Time `gorm:"default:null"`
	LastEvaluation        *time.Time `gorm:"default:null"`
	AdvisoryCountCache    int        `gorm:"column:installable_advisory_count_cache"`
	AdvisoryEnhCountCache int        `gorm:"column:installable_advisory_enh_count_cache"`
	AdvisoryBugCountCache int        `gorm:"column:installable_advisory_bug_count_cache"`
	AdvisorySecCountCache int        `gorm:"column:installable_advisory_sec_count_cache"`
	LastUpload            *time.Time `gorm:"default:null"`
	StaleTimestamp        *time.Time
	StaleWarningTimestamp *time.Time
	CulledTimestamp       *time.Time
	Stale                 bool
	DisplayName           string
	PackagesInstalled     int
	PackagesUpdatable     int
	ThirdParty            bool
	ReporterID            *int
	BaselineID            *int64
	BaselineUpToDate      *bool  `gorm:"column:baseline_uptodate"`
	YumUpdates            []byte `gorm:"column:yum_updates"`
	SatelliteManaged      bool   `gorm:"column:satellite_managed"`
}

func (SystemPlatform) TableName() string {
	return "system_platform"
}

func (s *SystemPlatform) GetInventoryID() string {
	if s == nil {
		return ""
	}
	return s.InventoryID
}

type String struct {
	ID    []byte `gorm:"primaryKey"`
	Value string
}

type PackageName struct {
	ID      int64 `json:"id" gorm:"primaryKey"`
	Name    string
	Summary *string
}

func (PackageName) TableName() string {
	return "package_name"
}

type Package struct {
	ID              int64 `json:"id" gorm:"primaryKey"`
	NameID          int64
	EVRA            string
	DescriptionHash *[]byte
	SummaryHash     *[]byte
	AdvisoryID      *int64
	Synced          bool
}

func (Package) TableName() string {
	return "package"
}

type PackageSlice []Package

type SystemPackage struct {
	RhAccountID int   `gorm:"primaryKey"`
	SystemID    int64 `gorm:"primaryKey"`
	PackageID   int64 `gorm:"primaryKey"`
	// Will contain json in form of [{ "evra": "...", "advisory": "..."}]
	UpdateData []byte
	NameID     int64
}

func (SystemPackage) TableName() string {
	return "system_package"
}

type PackageUpdate struct {
	EVRA     string `json:"evra"`
	Advisory string `json:"-"` // don't show it in API, we can probably remove it completely later
	Status   string `json:"status"`
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
	ID         int `gorm:"primaryKey"`
	Name       string
	Preference int
}

func (AdvisoryType) TableName() string {
	return "advisory_type"
}

type AdvisoryPackageData []string

type AdvisoryMetadata struct {
	ID              int64 `gorm:"primaryKey"`
	Name            string
	Description     string
	Synopsis        string
	Summary         string
	Solution        *string
	AdvisoryTypeID  int
	PublicDate      time.Time
	ModifiedDate    time.Time
	URL             *string
	SeverityID      *int
	PackageData     []byte
	CveList         []byte
	RebootRequired  bool
	ReleaseVersions []byte
	Synced          bool
}

func (AdvisoryMetadata) TableName() string {
	return "advisory_metadata"
}

type AdvisoryMetadataSlice []AdvisoryMetadata

type SystemAdvisories struct {
	RhAccountID   int   `gorm:"primaryKey"`
	SystemID      int64 `gorm:"primaryKey"`
	AdvisoryID    int64 `gorm:"primaryKey"`
	Advisory      AdvisoryMetadata
	FirstReported *time.Time
	StatusID      int
}

func (SystemAdvisories) TableName() string {
	return "system_advisories"
}

type SystemAdvisoriesSlice []SystemAdvisories

type AdvisoryAccountData struct {
	AdvisoryID         int64 `gorm:"primaryKey"`
	RhAccountID        int   `gorm:"primaryKey"`
	SystemsApplicable  int
	SystemsInstallable int
	Notified           *time.Time
}

func (AdvisoryAccountData) TableName() string {
	return "advisory_account_data"
}

type AdvisoryAccountDataSlice []AdvisoryAccountData
type Repo struct {
	ID         int64 `gorm:"primaryKey"`
	Name       string
	ThirdParty bool
}

func (Repo) TableName() string {
	return "repo"
}

type RepoSlice []Repo

type SystemRepo struct {
	RhAccountID int64 `gorm:"primaryKey"`
	SystemID    int64 `gorm:"primaryKey"`
	RepoID      int64 `gorm:"primaryKey"`
}

func (SystemRepo) TableName() string {
	return "system_repo"
}

type SystemRepoSlice []SystemRepo

type TimestampKV struct {
	Name  string `gorm:"unique"`
	Value time.Time
}

func (TimestampKV) TableName() string {
	return "timestamp_kv"
}

type PackageAccountData struct {
	AccID          int   `gorm:"column:rh_account_id;primaryKey"`
	PkgNameID      int64 `gorm:"column:package_name_id;primaryKey"`
	SysInstalled   int   `gorm:"column:systems_installed"`
	SysInstallable int   `gorm:"column:systems_installable"`
	SysApplicable  int   `gorm:"column:systems_applicable"`
}

func (PackageAccountData) TableName() string {
	return "package_account_data"
}

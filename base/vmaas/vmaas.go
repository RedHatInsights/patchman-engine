package vmaas

import (
	"app/base/types"
	"sort"
	"strings"
)

type UpdatesV3Request struct {
	PackageList    []string                       `json:"package_list"`
	RepositoryList []string                       `json:"repository_list"`
	ModulesList    *[]UpdatesV3RequestModulesList `json:"modules_list,omitempty"`
	Releasever     *string                        `json:"releasever,omitempty"`
	Basearch       *string                        `json:"basearch,omitempty"`
	SecurityOnly   *bool                          `json:"security_only,omitempty"`
	LatestOnly     *bool                          `json:"latest_only,omitempty"`
	// Include content from \"third party\" repositories into the response, disabled by default.
	ThirdParty *bool `json:"third_party,omitempty"`
	// Search for updates of unknown package EVRAs.
	OptimisticUpdates *bool `json:"optimistic_updates,omitempty"`
	// VMaaS will check package_list and return error if we provide package_list without epochs
	EpochRequired *bool `json:"epoch_required,omitempty"`
}

type UpdatesV3RequestModulesList struct {
	ModuleName   string `json:"module_name"`
	ModuleStream string `json:"module_stream"`
}

func (o *UpdatesV3Request) GetModulesList() []UpdatesV3RequestModulesList {
	if o == nil || o.ModulesList == nil {
		var ret []UpdatesV3RequestModulesList
		return ret
	}
	return *o.ModulesList
}

func (o *UpdatesV3Request) SetReleasever(v string) {
	o.Releasever = &v
}

type UpdatesV3Response struct {
	UpdateList     *map[string]UpdatesV3ResponseUpdateList `json:"update_list,omitempty"`
	RepositoryList *[]string                               `json:"repository_list,omitempty"`
	ModulesList    *[]UpdatesV3RequestModulesList          `json:"modules_list,omitempty"`
	Releasever     *string                                 `json:"releasever,omitempty"`
	Basearch       *string                                 `json:"basearch,omitempty"`
	LastChange     *string                                 `json:"last_change,omitempty"`
	BuildPkgcache  *bool                                   `json:"build_pkgcache,omitempty"`
}

// GetUpdateList returns the UpdateList field value if set, zero value otherwise.
func (o *UpdatesV3Response) GetUpdateList() map[string]UpdatesV3ResponseUpdateList {
	if o == nil || o.UpdateList == nil {
		var ret map[string]UpdatesV3ResponseUpdateList
		return ret
	}
	return *o.UpdateList
}

// GetBuildPkgcache returns the boolean value for the `build_pkgcache` field of yum_updates
func (o *UpdatesV3Response) GetBuildPkgcache() bool {
	if o == nil || o.BuildPkgcache == nil {
		return false
	}
	return *o.BuildPkgcache
}

type UpdatesV3ResponseUpdateList struct {
	AvailableUpdates *[]UpdatesV3ResponseAvailableUpdates `json:"available_updates,omitempty"`
}

func (o *UpdatesV3ResponseUpdateList) GetAvailableUpdates() []UpdatesV3ResponseAvailableUpdates {
	if o == nil || o.AvailableUpdates == nil {
		var ret []UpdatesV3ResponseAvailableUpdates
		return ret
	}
	updates := *o.AvailableUpdates
	sort.Slice(updates, func(i, j int) bool {
		// `less` function
		updatesI := updates[i]
		updatesJ := updates[j]
		if updatesI.Package == nil && updatesJ.Package != nil {
			return true
		}
		if updatesJ.Package == nil && updatesI.Package != nil {
			return false
		}
		if updatesJ.Package == nil && updatesI.Package == nil {
			return true
		}
		return *updatesI.Package < *updatesJ.Package
	})
	return updates
}

type UpdatesV3ResponseAvailableUpdates struct {
	Repository  *string `json:"repository,omitempty"`
	Releasever  *string `json:"releasever,omitempty"`
	Basearch    *string `json:"basearch,omitempty"`
	Erratum     *string `json:"erratum,omitempty"`
	Package     *string `json:"package,omitempty"`
	PackageName *string `json:"package_name,omitempty"`
	EVRA        *string `json:"evra,omitempty"`
	// helper column to diferentiate installable/applicable
	StatusID int `json:"-"`
}

func (o *UpdatesV3ResponseAvailableUpdates) GetPackage() string {
	if o == nil || o.Package == nil {
		var ret string
		return ret
	}
	return *o.Package
}

func (o *UpdatesV3ResponseAvailableUpdates) GetPackageName() string {
	if o == nil || o.PackageName == nil {
		var ret string
		return ret
	}
	return *o.PackageName
}

func (o *UpdatesV3ResponseAvailableUpdates) GetEVRA() string {
	if o == nil || o.EVRA == nil {
		var ret string
		return ret
	}
	return *o.EVRA
}

func (o *UpdatesV3ResponseAvailableUpdates) GetErratum() string {
	if o == nil || o.Erratum == nil {
		var ret string
		return ret
	}
	return *o.Erratum
}

func (o *UpdatesV3ResponseAvailableUpdates) GetBasearch() string {
	if o == nil || o.Basearch == nil {
		var ret string
		return ret
	}
	return *o.Basearch
}

func (o *UpdatesV3ResponseAvailableUpdates) GetReleasever() string {
	if o == nil || o.Releasever == nil {
		var ret string
		return ret
	}
	return *o.Releasever
}

func (o *UpdatesV3ResponseAvailableUpdates) GetRepository() string {
	if o == nil || o.Repository == nil {
		var ret string
		return ret
	}
	return *o.Repository
}

func (o *UpdatesV3ResponseAvailableUpdates) Cmp(b *UpdatesV3ResponseAvailableUpdates) int {
	if cmp := strings.Compare(o.GetPackage(), b.GetPackage()); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(o.GetErratum(), b.GetErratum()); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(o.GetRepository(), b.GetRepository()); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(o.GetBasearch(), b.GetBasearch()); cmp != 0 {
		return cmp
	}
	return strings.Compare(o.GetReleasever(), b.GetReleasever())
}

type ErrataRequest struct {
	Page          int      `json:"page,omitempty"`
	PageSize      int      `json:"page_size,omitempty"`
	ErrataList    []string `json:"errata_list"`
	ModifiedSince *string  `json:"modified_since,omitempty"`
	// Include content from \"third party\" repositories into the response, disabled by default.
	ThirdParty *bool     `json:"third_party,omitempty"`
	Type       *[]string `json:"type,omitempty"`
	Severity   *[]string `json:"severity,omitempty"`
}

type ErrataResponse struct {
	Page       int                                 `json:"page,omitempty"`
	PageSize   int                                 `json:"page_size,omitempty"`
	Pages      int                                 `json:"pages,omitempty"`
	ErrataList map[string]ErrataResponseErrataList `json:"errata_list,omitempty"`
	Type       []string                            `json:"type,omitempty"`
	Severity   []string                            `json:"severity,omitempty"`
	LastChange string                              `json:"last_change,omitempty"`
}

type ErrataResponseErrataList struct {
	Updated           string    `json:"updated,omitempty"`
	Severity          string    `json:"severity,omitempty"`
	ReferenceList     *[]string `json:"reference_list,omitempty"`
	Issued            string    `json:"issued,omitempty"`
	Description       string    `json:"description,omitempty"`
	Solution          *string   `json:"solution,omitempty"`
	Summary           string    `json:"summary,omitempty"`
	URL               *string   `json:"url,omitempty"`
	Synopsis          string    `json:"synopsis,omitempty"`
	CveList           *[]string `json:"cve_list,omitempty"`
	BugzillaList      *[]string `json:"bugzilla_list,omitempty"`
	PackageList       []string  `json:"package_list,omitempty"`
	SourcePackageList *[]string `json:"source_package_list,omitempty"`
	Type              string    `json:"type,omitempty"`
	ThirdParty        *bool     `json:"third_party,omitempty"`
	RequiresReboot    bool      `json:"requires_reboot,omitempty"`
	ReleaseVersions   *[]string `json:"release_versions,omitempty"`
}

type PkgListRequest struct {
	Page          int     `json:"page,omitempty"`
	PageSize      int     `json:"page_size,omitempty"`
	ModifiedSince *string `json:"modified_since,omitempty"`
	// Include 'modified' package attribute into the response
	ReturnModified *bool `json:"return_modified,omitempty"`
}

type PkgListResponse struct {
	Page        int           `json:"page,omitempty"`
	PageSize    int           `json:"page_size,omitempty"`
	Pages       int           `json:"pages,omitempty"`
	LastChange  *string       `json:"last_change,omitempty"`
	PackageList []PkgListItem `json:"package_list,omitempty"`
	// Total number of packages to return.
	Total int `json:"total,omitempty"`
}

type PkgListItem struct {
	Nevra       string `json:"nevra,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
	Modified    string `json:"modified,omitempty"`
}

type ReposRequest struct {
	Page           int      `json:"page,omitempty"`
	PageSize       int      `json:"page_size,omitempty"`
	RepositoryList []string `json:"repository_list"`
	// Return only repositories changed after the given date
	ModifiedSince *string `json:"modified_since,omitempty"`
	// Include content from \"third party\" repositories into the response, disabled by default.
	ThirdParty *bool `json:"third_party,omitempty"`
}

type ReposResponse struct {
	Page           int                                 `json:"page,omitempty"`
	PageSize       int                                 `json:"page_size,omitempty"`
	Pages          int                                 `json:"pages,omitempty"`
	RepositoryList map[string][]map[string]interface{} `json:"repository_list,omitempty"`
	LastChange     *string                             `json:"last_change,omitempty"`
}

type DBChangeResponse struct {
	ErrataChanges     *types.Rfc3339Timestamp `json:"errata_changes,omitempty"`
	CVEChanges        *types.Rfc3339Timestamp `json:"cve_changes,omitempty"`
	RepositoryChanges *types.Rfc3339Timestamp `json:"repository_changes,omitempty"`
	LastChange        *types.Rfc3339Timestamp `json:"last_change,omitempty"`
	Exported          *types.Rfc3339Timestamp `json:"exported,omitempty"`
}

func (o *DBChangeResponse) GetExported() *types.Rfc3339Timestamp {
	if o == nil {
		return nil
	}
	return o.Exported
}

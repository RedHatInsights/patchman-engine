package vmaas

type UpdatesV3Request struct {
	PackageList    []string                       `json:"package_list"`
	RepositoryList *[]string                      `json:"repository_list,omitempty"`
	ModulesList    *[]UpdatesV3RequestModulesList `json:"modules_list,omitempty"`
	Releasever     *string                        `json:"releasever,omitempty"`
	Basearch       *string                        `json:"basearch,omitempty"`
	SecurityOnly   *bool                          `json:"security_only,omitempty"`
	LatestOnly     *bool                          `json:"latest_only,omitempty"`
	// Include content from \"third party\" repositories into the response, disabled by default.
	ThirdParty *bool `json:"third_party,omitempty"`
	// Search for updates of unknown package EVRAs.
	OptimisticUpdates *bool `json:"optimistic_updates,omitempty"`
}

type UpdatesV3RequestModulesList struct {
	ModuleName   string `json:"module_name"`
	ModuleStream string `json:"module_stream"`
}

type UpdatesV2Response struct {
	UpdateList     *map[string]UpdatesV2ResponseUpdateList `json:"update_list,omitempty"`
	RepositoryList *[]string                               `json:"repository_list,omitempty"`
	ModulesList    *[]UpdatesV3RequestModulesList          `json:"modules_list,omitempty"`
	Releasever     *string                                 `json:"releasever,omitempty"`
	Basearch       *string                                 `json:"basearch,omitempty"`
	LastChange     *string                                 `json:"last_change,omitempty"`
}

// GetUpdateList returns the UpdateList field value if set, zero value otherwise.
func (o *UpdatesV2Response) GetUpdateList() map[string]UpdatesV2ResponseUpdateList {
	if o == nil || o.UpdateList == nil {
		var ret map[string]UpdatesV2ResponseUpdateList
		return ret
	}
	return *o.UpdateList
}

type UpdatesV2ResponseUpdateList struct {
	AvailableUpdates *[]UpdatesV2ResponseAvailableUpdates `json:"available_updates,omitempty"`
}

func (o *UpdatesV2ResponseUpdateList) GetAvailableUpdates() []UpdatesV2ResponseAvailableUpdates {
	if o == nil || o.AvailableUpdates == nil {
		var ret []UpdatesV2ResponseAvailableUpdates
		return ret
	}
	return *o.AvailableUpdates
}

type UpdatesV2ResponseAvailableUpdates struct {
	Repository *string `json:"repository,omitempty"`
	Releasever *string `json:"releasever,omitempty"`
	Basearch   *string `json:"basearch,omitempty"`
	Erratum    *string `json:"erratum,omitempty"`
	Package    *string `json:"package,omitempty"`
}

func (o *UpdatesV2ResponseAvailableUpdates) GetPackage() string {
	if o == nil || o.Package == nil {
		var ret string
		return ret
	}
	return *o.Package
}

func (o *UpdatesV2ResponseAvailableUpdates) GetErratum() string {
	if o == nil || o.Erratum == nil {
		var ret string
		return ret
	}
	return *o.Erratum
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
	Solution          string    `json:"solution,omitempty"`
	Summary           string    `json:"summary,omitempty"`
	URL               string    `json:"url,omitempty"`
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

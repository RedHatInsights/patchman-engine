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

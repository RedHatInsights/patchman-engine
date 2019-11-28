package vmaas_client

type VMaaSUpdatesResponse struct {
	UpdateList     map[string]UpdatingPackage `json:"update_list"`
	Releasever     string                     `json:"releasever"`
	Basearch       string                     `json:"basearch"`
	RepositoryList []string                   `json:"repository_list"`
	ModulesList    []string                   `json:"modules_list"`
	StatusCode     int
}

type UpdatingPackage struct {
	Summary          string   `json:"summary"`
	Description      string   `json:"description"`
	AvailableUpdates []Update `json:"available_updates"`
}

type Update struct {
	Package       string   `json:"package"`
	Erratum       string   `json:"erratum"`
	Repository    string   `json:"repository"`
	Basearch      string   `json:"basearch"`
	Releasever    string   `json:"releasever"`
}

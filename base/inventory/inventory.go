package inventory

type SystemProfile struct {
	Arch              *string          `json:"arch,omitempty"`
	InstalledPackages *[]string        `json:"installed_packages,omitempty"`
	OperatingSystem   *OperatingSystem `json:"operating_system,omitempty"`
	YumRepos          *[]YumRepo       `json:"yum_repos,omitempty"`
	DnfModules        *[]DnfModule     `json:"dnf_modules,omitempty"`
	Rhsm              Rhsm             `json:"rhsm,omitempty"`
}

type OperatingSystem struct {
	Major *int32  `json:"major,omitempty"`
	Minor *int32  `json:"minor,omitempty"`
	Name  *string `json:"name,omitempty"`
}

type YumRepo struct {
	ID      *string `json:"id,omitempty"`
	Name    *string `json:"name,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}

type DnfModule struct {
	Name   *string `json:"name,omitempty"`
	Stream *string `json:"stream,omitempty"`
}

type Rhsm struct {
	Version string `json:"version,omitempty"`
}

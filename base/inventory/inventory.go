package inventory

import (
	"app/base/types"

	"github.com/google/uuid"
)

type SystemProfile struct {
	Arch              *string         `json:"arch,omitempty"`
	HostType          string          `json:"host_type,omitempty"`
	InstalledPackages *[]string       `json:"installed_packages,omitempty"`
	YumRepos          *[]YumRepo      `json:"yum_repos,omitempty"`
	DnfModules        *[]DnfModule    `json:"dnf_modules,omitempty"`
	OperatingSystem   OperatingSystem `json:"operating_system,omitempty"`
	Rhsm              Rhsm            `json:"rhsm,omitempty"`
	Releasever        *string         `json:"releasever,omitempty"`
	SatelliteManaged  bool            `json:"satellite_managed,omitempty"`
	BootcStatus       Bootc           `json:"bootc_status,omitempty"`
	OwnerID           uuid.UUID       `json:"owner_id,omitempty"`
	Workloads         Workloads       `json:"workloads,omitempty"`
}

func (t *SystemProfile) GetInstalledPackages() []string {
	if t == nil || t.InstalledPackages == nil {
		return []string{}
	}
	return *t.InstalledPackages
}

func (t *SystemProfile) GetDnfModules() []DnfModule {
	if t == nil || t.DnfModules == nil {
		return []DnfModule{}
	}
	return *t.DnfModules
}

func (t *SystemProfile) GetYumRepos() []YumRepo {
	if t == nil || t.YumRepos == nil {
		return []YumRepo{}
	}
	return *t.YumRepos
}

type OperatingSystem struct {
	Major int16  `json:"major,omitempty"`
	Minor int16  `json:"minor,omitempty"`
	Name  string `json:"name,omitempty"`
}

type YumRepo struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	Enabled    bool   `json:"enabled,omitempty"`
	Mirrorlist string `json:"mirrorlist,omitempty"`
	BaseURL    string `json:"base_url,omitempty"`
}

type DnfModule struct {
	Name   string `json:"name,omitempty"`
	Stream string `json:"stream,omitempty"`
}

type Rhsm struct {
	Version      string   `json:"version,omitempty"`
	Environments []string `json:"environment_ids,omitempty"`
}

type Bootc struct {
	Booted BootcBooted `json:"booted,omitempty"`
}

type BootcBooted struct {
	Image string `json:"image,omitempty"`
}

type ReporterStaleness struct {
	LastCheckIn types.Rfc3339TimestampWithZ `json:"last_check_in"`
}

type Group struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Workloads struct {
	Sap     SapWorkload     `json:"sap,omitempty"`
	Ansible AnsibleWorkload `json:"ansible,omitempty"`
	Mssql   MssqlWorkload   `json:"mssql,omitempty"`
}

type SapWorkload struct {
	SapSystem bool     `json:"sap_system,omitempty"`
	Sids      []string `json:"sids,omitempty"`
}

type AnsibleWorkload struct {
	ControllerVersion string `json:"controller_version,omitempty"`
}

type MssqlWorkload struct {
	Version string `json:"version,omitempty"`
}

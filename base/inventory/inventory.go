package inventory

type SystemProfile struct {
	Arch            *string   `json:"arch,omitempty"`
	BiosReleaseDate *string   `json:"bios_release_date,omitempty"`
	BiosVendor      *string   `json:"bios_vendor,omitempty"`
	BiosVersion     *string   `json:"bios_version,omitempty"`
	CapturedDate    *string   `json:"captured_date,omitempty"`
	CloudProvider   *string   `json:"cloud_provider,omitempty"`
	CoresPerSocket  *int32    `json:"cores_per_socket,omitempty"`
	CpuFlags        *[]string `json:"cpu_flags,omitempty"`
	// The cpu model name
	CpuModel               *string   `json:"cpu_model,omitempty"`
	EnabledServices        *[]string `json:"enabled_services,omitempty"`
	GpgPubkeys             *[]string `json:"gpg_pubkeys,omitempty"`
	InfrastructureType     *string   `json:"infrastructure_type,omitempty"`
	InfrastructureVendor   *string   `json:"infrastructure_vendor,omitempty"`
	InsightsClientVersion  *string   `json:"insights_client_version,omitempty"`
	InsightsEggVersion     *string   `json:"insights_egg_version,omitempty"`
	InstalledPackages      *[]string `json:"installed_packages,omitempty"`
	InstalledPackagesDelta *[]string `json:"installed_packages_delta,omitempty"`
	InstalledServices      *[]string `json:"installed_services,omitempty"`
	// Indicates whether the host is part of a marketplace install from AWS, Azure, etc.
	IsMarketplace       *bool     `json:"is_marketplace,omitempty"`
	KatelloAgentRunning *bool     `json:"katello_agent_running,omitempty"`
	KernelModules       *[]string `json:"kernel_modules,omitempty"`
	LastBootTime        *string   `json:"last_boot_time,omitempty"`
	NumberOfCpus        *int32    `json:"number_of_cpus,omitempty"`
	NumberOfSockets     *int32    `json:"number_of_sockets,omitempty"`
	// The kernel version represented with a three, optionally four, number scheme.
	OsKernelVersion *string          `json:"os_kernel_version,omitempty"`
	OsRelease       *string          `json:"os_release,omitempty"`
	OperatingSystem *OperatingSystem `json:"operating_system,omitempty"`
	// A UUID associated with the host's RHSM certificate
	OwnerId *string `json:"owner_id,omitempty"`
	// A UUID associated with a cloud_connector
	RhcClientId *string `json:"rhc_client_id,omitempty"`
	// A UUID associated with the config manager state
	RhcConfigState   *string   `json:"rhc_config_state,omitempty"`
	RunningProcesses *[]string `json:"running_processes,omitempty"`
	// The instance number of the SAP HANA system (a two-digit number between 00 and 99)
	SapInstanceNumber *string   `json:"sap_instance_number,omitempty"`
	SapSids           *[]string `json:"sap_sids,omitempty"`
	// Indicates if SAP is installed on the system
	SapSystem *bool `json:"sap_system,omitempty"`
	// The version of the SAP HANA lifecycle management program
	SapVersion       *string `json:"sap_version,omitempty"`
	SatelliteManaged *bool   `json:"satellite_managed,omitempty"`
	// The SELinux mode provided in the config file
	SelinuxConfigFile *string `json:"selinux_config_file,omitempty"`
	// The current SELinux mode, either enforcing, permissive, or disabled
	SelinuxCurrentMode     *string `json:"selinux_current_mode,omitempty"`
	SubscriptionAutoAttach *string `json:"subscription_auto_attach,omitempty"`
	SubscriptionStatus     *string `json:"subscription_status,omitempty"`
	SystemMemoryBytes      *int64  `json:"system_memory_bytes,omitempty"`
	// Current profile resulting from command tuned-adm active
	TunedProfile *string      `json:"tuned_profile,omitempty"`
	YumRepos     *[]YumRepo   `json:"yum_repos,omitempty"`
	DnfModules   *[]DnfModule `json:"dnf_modules,omitempty"`
	Rhsm         Rhsm         `json:"rhsm,omitempty"`
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

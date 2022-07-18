package utils

var emptyStruct = []byte(`{}`)
var emptyUpdate = []byte(`
	{
		"available_updates": []
	}
`)

var pkgA1 = []byte(`
	{
		"available_updates": [
			{
				"package": "pkgA-0:1.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "8"
			}
		]
	}
`)

var pkgA2 = []byte(`
	{
		"available_updates": [
			{
				"package": "pkgA-0:2.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "8"
			}
		]
	}
`)

var pkgA12 = []byte(`
	{
		"available_updates": [
			{
				"package": "pkgA-0:1.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "8"
			},
			{
				"package": "pkgA-0:2.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "8"
			}
		]
	}
`)

var pkgA123 = []byte(`
	{
		"available_updates": [
			{
				"package": "pkgA-0:1.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "8"
			},
			{
				"package": "pkgA-0:2.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "8"
			},
			{
				"package": "pkgA-0:3.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "8"
			}
		]
	}
`)

var pkgA1Xattrs = []byte(`
	{
		"available_updates": [
			{
				"package": "pkgA-0:1.0-1.x86_64",
				"erratum": "XXXX-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "8"
			},
			{
				"package": "pkgA-0:1.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "XXXX-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "8"
			},
			{
				"package": "pkgA-0:1.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "XXX_64",
				"releasever": "8"
			},
			{
				"package": "pkgA-0:1.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "XXX"
			}
		]
	}
`)

var pkgA123Xattrs = []byte(`
	{
		"available_updates": [
			{
				"package": "pkgA-0:1.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "8"
			},
			{
				"package": "pkgA-0:1.0-1.x86_64",
				"erratum": "XXXX-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "8"
			},
			{
				"package": "pkgA-0:1.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "XXXX-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "8"
			},
			{
				"package": "pkgA-0:1.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "XXX_64",
				"releasever": "8"
			},
			{
				"package": "pkgA-0:1.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "XXX"
			},
			{
				"package": "pkgA-0:2.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "8"
			},
			{
				"package": "pkgA-0:3.0-1.x86_64",
				"erratum": "RHSA-9999:0001",
				"repository": "rhel-8-for-x86_64-baseos-rpms",
				"basearch": "x86_64",
				"releasever": "8"
			}
		]
	}
`)

var kernel3101 = []byte(`
	{
		"update_list": {
			"kernel-0:3.10.0-1160.42.2.el7.x86_64": {
				"available_updates": [
					{
						"erratum": "RHSA-2021:3801",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.10.1-1160.42.2.el7.x86_64"
					},
					{
						"erratum": "RHSA-2021:4777",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.10.4-1160.42.2.el7.x86_64"
					},
					{
						"erratum": "RHSA-2022:0063",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.10.7-1160.42.2.el7.x86_64"
					},
					{
						"erratum": "RHSA-2022:0620",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.10.9-1160.42.2.el7.x86_64"
					}
				]
			}
		},
		"repository_list": ["rhel7"],
		"modules_list": [],
		"basearch": "x86_64",
		"releasever": "7Server"
	}`,
)

var kernel3102 = []byte(`
	{
		"update_list": {
			"kernel-0:3.10.0-1160.42.2.el7.x86_64": {
				"available_updates": [
					{
						"erratum": "RHSA-2021:3801",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.10.1-1160.42.2.el7.x86_64"
					},
					{
						"erratum": "RHSA-2021:4777",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.10.9-1160.42.2.el7.x86_64"
					},
					{
						"erratum": "RHSA-2022:0063",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.11.0-1160.42.2.el7.x86_64"
					},
					{
						"erratum": "RHSA-2022:0620",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.12.8-1160.42.2.el7.x86_64"
					}
				]
			}
		},
		"repository_list": ["rhel7"],
		"modules_list": [],
		"basearch": "x86_64",
		"releasever": "7Server"
	}`,
)

var kernel3101and3102 = []byte(`
	{
		"update_list": {
			"kernel-0:3.10.0-1160.42.2.el7.x86_64": {
				"available_updates": [
					{
						"erratum": "RHSA-2021:3801",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.10.1-1160.42.2.el7.x86_64"
					},
					{
						"erratum": "RHSA-2021:4777",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.10.4-1160.42.2.el7.x86_64"
					},
					{
						"erratum": "RHSA-2022:0063",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.10.7-1160.42.2.el7.x86_64"
					},
					{
						"erratum": "RHSA-2021:4777",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.10.9-1160.42.2.el7.x86_64"
					},
					{
						"erratum": "RHSA-2022:0620",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.10.9-1160.42.2.el7.x86_64"
					},
					{
						"erratum": "RHSA-2022:0063",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.11.0-1160.42.2.el7.x86_64"
					},
					{
						"erratum": "RHSA-2022:0620",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.12.8-1160.42.2.el7.x86_64"
					}
				]
			}
		},
		"repository_list": ["rhel7"],
		"modules_list": [],
		"basearch": "x86_64",
		"releasever": "7Server"
	}`,
)

var kernel3111 = []byte(`
	{
		"update_list": {
			"kernel-0:3.11.0-1160.42.2.el7.x86_64": {
				"available_updates": [
					{
						"erratum": "RHSA-2022:0620",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.11.2-1160.42.2.el7.x86_64"
					}
				]
			}
		},
		"repository_list": ["rhel7"],
		"modules_list": [],
		"basearch": "x86_64",
		"releasever": "7Server"
	}`,
)

var kernel3121 = []byte(`
	{
		"update_list": {
			"kernel-0:3.12.0-1160.42.2.el7.x86_64": {
				"available_updates": [
					{
						"erratum": "RHSA-2022:0620",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.12.4-1160.42.2.el7.x86_64"
					}
				]
			}
		},
		"repository_list": ["rhel7"],
		"modules_list": [],
		"basearch": "x86_64",
		"releasever": "7Server"
	}`,
)

var kernel3111AndKernel3121 = []byte(`
	{
		"update_list": {
			"kernel-0:3.12.0-1160.42.2.el7.x86_64": {
				"available_updates": [
					{
						"erratum": "RHSA-2022:0620",
						"basearch": "x86_64",
						"releasever": "7Server",
						"repository": "rhel-7-server-rpms",
						"package": "kernel-0:3.12.4-1160.42.2.el7.x86_64"
					}
				]
			}
		},
		"repository_list": ["rhel7"],
		"modules_list": [],
		"basearch": "x86_64",
		"releasever": "7Server"
	}`,
)

var bash44201 = []byte(`
	{
		"update_list": {
			"bash-0:4.4.20-1.el8_4.x86_64": {
				"available_updates": [
					{
						"package": "bash-0:4.4.23-1.el8_4.x86_64",
						"repository": "rhel-8-for-x86_64-baseos-rpms",
						"basearch": "x86_64",
						"releasever": "8"
					}
				]
			}
		},
		"releasever": "8",
		"basearch": "x86_64"
	}
`)

var bash44202 = []byte(`
	{
		"update_list": {
			"bash-0:4.4.20-1.el8_4.x86_64": {
				"available_updates": [
					{
						"package": "bash-0:4.5.20-1.el8_4.x86_64",
						"repository": "ubi-8-baseos",
						"basearch": "x86_64",
						"releasever": "8"
					}
				]
			}
		},
		"releasever": "8",
		"basearch": "x86_64"
	}
`)

var bash44201AndBash44202 = []byte(`
	{
		"update_list": {
			"bash-0:4.4.20-1.el8_4.x86_64": {
				"available_updates": [
					{
						"package": "bash-0:4.4.23-1.el8_4.x86_64",
						"repository": "rhel-8-for-x86_64-baseos-rpms",
						"basearch": "x86_64",
						"releasever": "8"
					},
					{
						"package": "bash-0:4.5.20-1.el8_4.x86_64",
						"repository": "ubi-8-baseos",
						"basearch": "x86_64",
						"releasever": "8"
					}
				]
			}
		},
		"releasever": "8",
		"basearch": "x86_64"
	}
`)

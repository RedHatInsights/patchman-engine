package platform

import (
	"app/base/inventory"
	"app/base/utils"
	"crypto/rand"
	"math/big"
)

var pkgs = []string{
	"kernel-debug-devel-2.6.32-220.el6.i686",
	"bogl-debuginfo-0.1.18-11.2.1.el5.1.i386",
	"tetex-latex-3.0-33.13.el5.x86_64",
	"openssh-clients-5.3p1-20.el6_0.3.i686",
	"httpd-debuginfo-2.2.3-43.el5.i386",
	"openoffice.org-langpack-tn_ZA-1:3.2.1-19.6.el6_0.5.i686",
	"mod_nss-debuginfo-1.0.8-8.el5_10.i386",
	"java-1.5.0-ibm-demo-1:1.5.0.16.9-1jpp.1.el5.i386",
	"openoffice.org-calc-1:3.2.1-19.6.el6_0.5.i686",
	"rubygem-foreman_api-0.1.11-6.el6sat.noarch",
	"bluez-libs-debuginfo-3.7-1.1.i386",
	"java-1.6.0-sun-demo-1:1.6.0.27-1jpp.2.el5.x86_64",
	"thunderbird-debuginfo-2.0.0.24-6.el5.x86_64",
	"chkconfig-debuginfo-1.3.30.2-2.el5.i386",
	"PackageKit-device-rebind-0.5.8-20.el6.i686",
	"java-1.7.0-oracle-devel-1:1.7.0.25-1jpp.1.el5_9.i386",
	"xulrunner-debuginfo-1.9.0.7-3.el5.i386",
	"mysql-server-5.1.66-2.el6_3.i686",
	"iproute-2.6.18-13.el5.i386",
	"libbonobo-2.24.2-5.el6.i686"}

// Create bare system profile
func makeSystemProfile(id string, randomPkgs bool) inventory.SystemProfile {
	_pkgs := pkgs
	if id == "TEST-NO-PKGS" {
		_pkgs = []string{}
	} else if randomPkgs {
		nPkgs, _ := rand.Int(rand.Reader, big.NewInt(int64(len(pkgs))))
		_pkgs = pkgs[0:nPkgs.Int64()]
	}

	yumRepos := []inventory.YumRepo{
		{
			ID:      "repo1",
			Name:    "Debug packages",
			Enabled: true,
		},
	}
	dnfModules := []inventory.DnfModule{
		{
			Name:   "firefox",
			Stream: "60",
		},
	}
	return inventory.SystemProfile{
		Arch:              utils.PtrString("i686"),
		InstalledPackages: &_pkgs,
		YumRepos:          &yumRepos,
		DnfModules:        &dnfModules,
	}
}

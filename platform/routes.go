package platform

import (
	"app/base/utils"
	"app/listener"
	"context"
	"encoding/json"
	"github.com/RedHatInsights/patchman-clients/inventory"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"net/http"
)

// Init routes.
func Init(app *gin.Engine) {
	// public routes
	app.GET("/api/inventory/v1/:host_id/system_profile", SystemProfileHandler)
	app.POST("/api/mock_upload", MockUploadHandler)
}

func SystemProfileHandler(c *gin.Context) {
	c.JSON(http.StatusOK, makeSystemProfile(c.Param("host_id")))
}
func MockUploadHandler(c *gin.Context) {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    "platform.upload.available",
		Balancer: &kafka.LeastBytes{},
	})

	identity, err := utils.Identity{
		Entitlements: map[string]utils.Entitlement{
			"smart_management": {Entitled: true},
		},
		Identity: utils.IdentityDetail{
			AccountNumber: "0",
			Type:          "User",
		},
	}.Encode()

	if err != nil {
		panic(err)
	}

	event := listener.PlatformEvent{
		Id:          "bblabla",
		B64Identity: &identity,
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}

	err = writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte{},
		Value: encoded,
	})
	if err != nil {
		panic(err)
	}

	c.Status(http.StatusOK)
}

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

func makeSystemProfile(Id string) inventory.SystemProfileByHostOut {

	profile := inventory.HostSystemProfileOut{
		Id: Id,
		SystemProfile: inventory.SystemProfileIn{
			Arch:              "i686",
			InstalledPackages: pkgs,
			// TODO : Add repo id after https://github.com/RedHatInsights/insights-host-inventory/pull/536
			YumRepos: []inventory.YumRepo{
				{
					Name:     "Debug packages",
					Baseurl:  "http://repo.com/$arch/$releasever/$product/repo",
					Enabled:  true,
					Gpgcheck: false,
				},
			},
			// TODO: Add modules
		},
	}

	return inventory.SystemProfileByHostOut{
		Total:   1,
		Count:   1,
		Page:    0,
		PerPage: 1,
		Results: []inventory.HostSystemProfileOut{profile},
	}
}

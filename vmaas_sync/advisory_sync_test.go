package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	utils.TestLoadEnv("conf/vmaas_sync.env")
}

//nolint:lll,misspell
const testAdvisories = `
{
   "errata_list":{
      "RHBA-2004:391":{
         "synopsis":"Updated perl packages",
         "summary":"Updated perl packages that fix a UTF-8 support bug are now available.",
         "type":"bugfix",
         "severity":"None",
         "description":"Perl is a high-level programming language with roots in C, sed, awk\nand shell scripting.  Perl is good at handling processes and files,\nand is especially good at handling text.\n\nPerl-5.8.0 now includes default support for UTF-8 character encoding for\nRed Hat Enterprise Linux 3.  Some interactions between UTF-8 support and\nperl could result in corrupted data.  This update fixes an issue in regards\nto regular expression handling.\n\nAll users of perl should upgrade to these updated packages, which resolve\nthis issue.",
         "solution":"Before applying this update, make sure that all previously-released\nerrata relevant to your system have been applied.  Use Red Hat\nNetwork to download and update your packages.  To launch the Red Hat\nUpdate Agent, use the following command:\n\n    up2date\n\nFor information on how to install packages manually, refer to the\nfollowing Web page for the System Administration or Customization\nguide specific to your system:\n\n    http://www.redhat.com/docs/manuals/enterprise/",
         "issued":"2004-09-02T00:00:00+00:00",
         "updated":"2004-09-02T00:00:00+00:00",
         "cve_list":[

         ],
         "package_list":[

         ],
         "source_package_list":[

         ],
         "bugzilla_list":[
            "112339"
         ],
         "reference_list":[

         ],
         "modules_list":[

         ],
         "url":"https://access.redhat.com/errata/RHBA-2004:391",
         "requires_reboot": true
      },
      "RHBA-2004:403":{
         "synopsis":"Updated rusers packages",
         "summary":"Updated rusers packages that remove the requirement for procps are now\navailable.",
         "type":"bugfix",
         "severity":"None",
         "description":"The rusers program allows users to find out who is logged into certain\nmachines on the local network. The 'rusers' command produces output\nsimilar to 'who', but for a specified list of hosts or for all machines\non the local network.\n\nPrevious versions of the rusers package, and the included rstatd\napplication, had a requirement such that the procps package and the\nlibraries therein were required for rusers to function properly. This\ncaused problems when updated versions of procps were released. These\nupdated rusers packages contain a fix that removes the procps package\ndependancy.\n\nAll users of rusers and rstatd should upgrade to these updated packages,\nwhich resolve this issue.",
         "solution":"Before applying this update, make sure that all previously-released\nerrata relevant to your system have been applied.  Use Red Hat\nNetwork to download and update your packages.  To launch the Red Hat\nUpdate Agent, use the following command:\n\n    up2date\n\nFor information on how to install packages manually, refer to the\nfollowing Web page for the System Administration or Customization\nguide specific to your system:\n\n    http://www.redhat.com/docs/manuals/enterprise/",
         "issued":"2004-09-02T00:00:00+00:00",
         "updated":"2004-09-02T00:00:00+00:00",
         "cve_list":[

         ],
         "package_list":[

         ],
         "source_package_list":[

         ],
         "bugzilla_list":[

         ],
         "reference_list":[

         ],
         "modules_list":[

         ],
         "url":"https://access.redhat.com/errata/RHBA-2004:403",
         "requires_reboot": true
      }
	},
   "page":0,
   "page_size":10,
   "pages":1
}
`

func TestParseAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	data := map[string]vmaas.ErrataResponseErrataList{
		"ER1": {
			Updated:           "2004-09-02T00:00:00+00:00",
			Severity:          "",
			ReferenceList:     &[]string{},
			Issued:            "2004-09-02T00:00:00+00:00",
			Description:       "DESC",
			Solution:          utils.PtrString("SOL"),
			Summary:           "SUM",
			URL:               utils.PtrString("URL"),
			Synopsis:          "SYN",
			CveList:           utils.PtrSliceString([]string{"CVE-1", "CVE-2", "CVE-3"}),
			BugzillaList:      &[]string{},
			PackageList:       []string{},
			SourcePackageList: &[]string{},
			Type:              "bugfix",
			ThirdParty:        new(bool),
			RequiresReboot:    true,
			ReleaseVersions:   utils.PtrSliceString([]string{"8.0", "8.1"}),
		},
	}

	res, err := parseAdvisories(data)
	assert.Nil(t, err)
	assert.Equal(t, len(res), 1)
	adv := res[0]

	time, err := time.Parse(base.Rfc3339NoTz, "2004-09-02T00:00:00+00:00")
	assert.Nil(t, err)
	assert.Equal(t, time, adv.PublicDate)
	assert.Equal(t, time, adv.ModifiedDate)
	assert.Equal(t, "DESC", adv.Description)
	assert.Equal(t, "SOL", *adv.Solution)
	assert.Equal(t, "SUM", adv.Summary)
	assert.Equal(t, "URL", *adv.URL)
	assert.Equal(t, "SYN", adv.Synopsis)
	assert.Equal(t, 2, adv.AdvisoryTypeID)
	assert.Equal(t, 2, adv.AdvisoryTypeID)
	assert.Equal(t, true, adv.RebootRequired)
	js := json.RawMessage(string(adv.CveList))
	cves, _ := json.Marshal(js)
	assert.Equal(t, string(cves), `["CVE-1","CVE-2","CVE-3"]`)
	js = json.RawMessage(string(adv.ReleaseVersions))
	relvers, _ := json.Marshal(js)
	assert.Equal(t, string(relvers), `["8.0","8.1"]`)
}

func TestSaveAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	var data vmaas.ErrataResponse

	assert.Nil(t, json.Unmarshal([]byte(testAdvisories), &data))
	errataList := data.ErrataList
	testString := "TEST"
	for i, v := range errataList {
		v.URL = &testString
		errataList[i] = v
	}

	// Repeatedly storing erratas should just overwrite them
	for i := 0; i < 2; i++ {
		err := storeAdvisories(data.ErrataList)
		assert.NoError(t, err)
		var count int64

		assert.Nil(t, database.Db.Model(&models.AdvisoryMetadata{}).Where("url = ?", "TEST").Count(&count).Error)

		assert.Equal(t, count, int64(len(data.ErrataList)))
	}

	assert.Nil(t, database.Db.Unscoped().Where("url = ?", "TEST").Delete(&models.AdvisoryMetadata{}).Error)
}

func TestSyncAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	err := syncAdvisories(time.Now(), nil)
	assert.NoError(t, err)

	expected := []string{"RH-100"}
	database.CheckAdvisoriesInDB(t, expected)

	database.DeleteNewlyAddedPackages(t)
	database.DeleteNewlyAddedAdvisories(t)
}

func TestSyncAdvisoriesCheck(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	modifiedSince := "2020-01-01T00:00:00+00:00"
	err := syncAdvisories(time.Now(), &modifiedSince)
	assert.NoError(t, err)

	expected := []string{"RH-100"}
	database.CheckAdvisoriesInDB(t, expected)

	var am models.AdvisoryMetadata
	assert.Nil(t, database.Db.Model(&models.AdvisoryMetadata{}).Where("name in (?)", expected).Find(&am).Error)
	assert.Equal(t, "adv-100-des", am.Description)
	assert.Equal(t, "adv-100-sum", am.Summary)
	assert.Equal(t, "adv-100-syn", am.Synopsis)
	assert.Equal(t, "adv-100-sol", *am.Solution)
	assert.Equal(t, true, am.RebootRequired)
	assert.Equal(t, 3, am.AdvisoryTypeID)
	assert.Equal(t, "2020-01-02 08:04:05 +0000 UTC", am.PublicDate.String())
	assert.Equal(t, "2020-01-02 08:04:05 +0000 UTC", am.ModifiedDate.String())
	assert.Equal(t, true, am.Synced)

	database.DeleteNewlyAddedPackages(t)
	database.DeleteNewlyAddedAdvisories(t)
}

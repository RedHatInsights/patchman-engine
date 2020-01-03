package vmaas_sync

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"encoding/json"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

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
         "url":"https://access.redhat.com/errata/RHBA-2004:391"
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
         "url":"https://access.redhat.com/errata/RHBA-2004:403"
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
			Updated:     "2004-09-02T00:00:00+00:00",
			Issued:      "2004-09-02T00:00:00+00:00",
			Description: "DESC",
			Solution:    "SOL",
			Summary:     "SUM",
			Url:         "URL",
			Synopsis:    "SYN",
			Type:        "bugfix",
		},
	}

	res, err := parseAdvisories(data)
	assert.Nil(t, err)
	assert.Equal(t, len(res), 1)
	adv := res[0]

	time, err := time.Parse(base.RFC_3339_NO_TZ, "2004-09-02T00:00:00+00:00")
	assert.Nil(t, err)
	assert.Equal(t, adv.PublicDate, time)
	assert.Equal(t, adv.ModifiedDate, time)
	assert.Equal(t, adv.Description, "DESC")
	assert.Equal(t, adv.Solution, "SOL")
	assert.Equal(t, adv.Summary, "SUM")
	assert.Equal(t, *adv.Url, "URL")
	assert.Equal(t, adv.Synopsis, "SYN")
	assert.Equal(t, adv.AdvisoryTypeId, 2)

}

func TestSaveAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	var data vmaas.ErrataResponse

	assert.Nil(t, json.Unmarshal([]byte(testAdvisories), &data))
	for i, v := range data.ErrataList {
		v.Url = "TEST"
		data.ErrataList[i] = v
	}

	// Repeatedly storing erratas should just overwrite them
	for i := 0; i < 2; i++ {
		assert.Nil(t, storeAdvisories(data.ErrataList))
		var count int

		assert.Nil(t, database.Db.Model(&models.AdvisoryMetadata{}).Where("url = ?", "TEST").Count(&count).Error)

		assert.Equal(t, count, len(data.ErrataList))
	}

	assert.Nil(t, database.Db.Unscoped().Where("url = ?", "TEST").Delete(&models.AdvisoryMetadata{}).Error)
}

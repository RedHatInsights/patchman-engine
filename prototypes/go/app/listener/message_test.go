package listener

import (
	"github.com/bmizerany/assert"
	"testing"
)

func TestFilter(t *testing.T)  {
	msg := Message{
		Arch: "i686",
		Packages: &[]string{
			"kdepimlibs-akonadi-4.3.4-4.el6.i686",
			"lohit-oriya-fonts-2.4.3-6.el6.noarch",
			"bzip2-debuginfo-1.0.3-4.el5_2.i386",
			"upstart-0.6.5-6.1.el6_0.1.i686",
		}}
	msg.FilterPackages()
	assert.Equal(t, 2, len(*msg.Packages))
	assert.Equal(t, "kdepimlibs-akonadi-4.3.4-4.el6.i686", (*msg.Packages)[0])
	assert.Equal(t, "upstart-0.6.5-6.1.el6_0.1.i686", (*msg.Packages)[1])
}

func TestToJSON(t *testing.T)  {
	msg := Message{
		Arch: "i686",
		Packages: &[]string{
			"pkg1.i686",
			"pkg2.noarch",
		}}
	js := msg.ToJSON()
	assert.Equal(t, `{"id":0,"arch":"i686","packages":["pkg1.i686","pkg2.noarch"]}`, string(js))
}

func TestJSONChecksum(t *testing.T)  {
	msg := Message{
		Arch: "i686",
		Packages: &[]string{
			"pkg1.i686",
			"pkg2.noarch",
		}}
	checksum := msg.JSONChecksum()
	assert.Equal(t, `8bcb19df5adec337262b6b4c5502554bfb624e20973990146dd8443dea51e1d5`, checksum)
}

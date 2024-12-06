package middlewares

import (
	"app/base/database"
	"app/base/utils"
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func testSetup(t *testing.T) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	utils.SkipWithoutDB(t)
	utils.SetDefaultEnvOrFail("LOG_LEVEL", "debug")
	utils.ConfigureLogging()
	database.Configure()
	return c
}

func TestFindAccount(t *testing.T) {
	c := testSetup(t)
	orgID := "9876543"
	// account does not exist
	assert.NotContains(t, AccountIDCache.Values, orgID)
	// account does not exist but it is created
	firstFind := findAccount(c, orgID)
	assert.True(t, firstFind)
	assert.Contains(t, AccountIDCache.Values, orgID)
	accID1 := AccountIDCache.Values[orgID]
	// account exists
	secondFind := findAccount(c, orgID)
	assert.True(t, secondFind)
	assert.Contains(t, AccountIDCache.Values, orgID)
	accID2 := AccountIDCache.Values[orgID]
	// rhAccount.ID should be the same
	// second find should not create another record
	assert.Equal(t, accID1, accID2)
}

func TestUpdateOrgID(t *testing.T) {
	c := testSetup(t)
	orgID := "7654321"
	found := findAccount(c, orgID)
	assert.True(t, found)
	updatedID := AccountIDCache.Values[orgID]
	assert.NotEqual(t, updatedID, 0)
}

func TestFindDNoAccNr(t *testing.T) {
	c := testSetup(t)
	orgID := "2222222"
	// create account
	found := findAccount(c, orgID)
	assert.True(t, found)
	assert.Contains(t, AccountIDCache.Values, orgID)
	accID := AccountIDCache.Values[orgID]
	// account exists
	found = findAccount(c, orgID)
	assert.True(t, found)
	assert.Contains(t, AccountIDCache.Values, orgID)
	updatedID := AccountIDCache.Values[orgID]
	assert.Equal(t, updatedID, accID)
	// new account without AccountNumber
	// test that account `accID` without AccountNumber is not overwritten
	orgID = "3333333"
	found = findAccount(c, orgID)
	assert.True(t, found)
	assert.Contains(t, AccountIDCache.Values, orgID)
	newID := AccountIDCache.Values[orgID]
	assert.NotEqual(t, newID, accID)
}

var (
	identityHeaderTmpl, _ = template.New("x-rh-identity").Parse(`{
    "identity": {
        "org_id": "{{.OrgID}}",
        "auth_type": "{{.AuthType}}",
        "type": "{{.Type}}",
		"user": {
            "username": "jdoe@acme.com",
            "email": "jdoe@acme.com",
            "first_name": "john",
            "last_name": "doe",
            "is_active": true,
            "is_org_admin": false,
            "is_internal": false,
            "locale": "en_US"
        },
        "system": {
            "cert_type": "{{.CertType}}",
            "cn": "{{.SystemCN}}"
        },
        "internal": {
            "org_id": "{{.OrgID}}",
            "auth_type": "{{.AuthType}}",
            "auth_time": 6300
        }
    },
    "entitlements": {
        "insights": {
            "is_entitled": true
        }
    }
}`)
	systemCN = "cccccccc-0000-0000-0001-000000000004"
)

func TestPublicAuthenticator(t *testing.T) {
	c := testSetup(t)
	var identityHeader bytes.Buffer
	err := identityHeaderTmpl.Execute(&identityHeader,
		map[string]string{"OrgID": "org_1", "Type": "User", "AuthType": "basic-auth"})
	assert.Nil(t, err)
	c.Request = &http.Request{Header: http.Header{}}
	c.Request.Header.Add("x-rh-identity", base64.StdEncoding.EncodeToString(identityHeader.Bytes()))
	PublicAuthenticator()(c)
	assert.Equal(t, 1, c.GetInt(utils.KeyAccount))
	assert.Equal(t, "john doe", c.GetString(utils.KeyUser))
}

func TestPublicAuthenticatorMissingHeader(t *testing.T) {
	c := testSetup(t)
	// no x-rh-identity header
	c.Request = &http.Request{Header: http.Header{}}

	PublicAuthenticator()(c)
	assert.Nil(t, c.Keys)
}

func TestTurnpikeAuthenticator(t *testing.T) {
	c := testSetup(t)
	var identityHeader bytes.Buffer
	err := identityHeaderTmpl.Execute(&identityHeader,
		map[string]string{"OrgID": "org_1", "Type": "Associate", "AuthType": "basic-auth"})
	assert.Nil(t, err)
	c.Request = &http.Request{Header: http.Header{}}
	c.Request.Header.Add("x-rh-identity", base64.StdEncoding.EncodeToString(identityHeader.Bytes()))
	TurnpikeAuthenticator()(c)
	assert.False(t, c.IsAborted())
}

func TestTurnpikeAuthenticatorAbort(t *testing.T) {
	c := testSetup(t)
	var identityHeader bytes.Buffer
	err := identityHeaderTmpl.Execute(&identityHeader,
		map[string]string{"OrgID": "org_1", "Type": "User", "AuthType": "basic-auth"})
	assert.Nil(t, err)
	c.Request = &http.Request{Header: http.Header{}}
	c.Request.Header.Add("x-rh-identity", base64.StdEncoding.EncodeToString(identityHeader.Bytes()))
	TurnpikeAuthenticator()(c)
	assert.True(t, c.IsAborted())
}

func TestSystemCertAuthenticator(t *testing.T) {
	c := testSetup(t)
	var systemIdentityHeader bytes.Buffer
	err := identityHeaderTmpl.Execute(&systemIdentityHeader,
		map[string]string{"OrgID": "org_1", "SystemCN": systemCN, "Type": "System",
			"CertType": "system", "AuthType": "cert-type"})
	assert.Nil(t, err)
	c.Request = &http.Request{Header: http.Header{}}
	c.Request.Header.Add("x-rh-identity", base64.StdEncoding.EncodeToString(systemIdentityHeader.Bytes()))
	SystemCertAuthenticator()(c)
	assert.Equal(t, 1, c.GetInt(utils.KeyAccount))
	assert.Equal(t, systemCN, c.GetString(utils.KeySystem))
}

func TestSystemCertAuthenticatorDeny(t *testing.T) {
	c := testSetup(t)
	c.Request = &http.Request{Header: http.Header{}}
	var systemIdentityHeader bytes.Buffer

	id1 := map[string]string{"OrgID": "org_1", "SystemCN": systemCN, "Type": "User",
		"CertType": "system", "AuthType": "cert-type"}
	id2 := map[string]string{"OrgID": "org_1", "SystemCN": systemCN, "Type": "System",
		"CertType": "not-valid", "AuthType": "cert-type"}
	id3 := map[string]string{"OrgID": "org_1", "SystemCN": "not-valid", "Type": "System",
		"CertType": "system", "AuthType": "cert-type"}
	identities := []map[string]string{id1, id2, id3}

	for _, id := range identities {
		err := identityHeaderTmpl.Execute(&systemIdentityHeader, id)
		assert.Nil(t, err)
		c.Request.Header.Add("x-rh-identity", base64.StdEncoding.EncodeToString(systemIdentityHeader.Bytes()))
		SystemCertAuthenticator()(c)
		assert.Nil(t, c.Keys)
	}
}

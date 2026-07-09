package utils

import (
	"testing"

	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
	"github.com/stretchr/testify/assert"
)

var identityStringWithUserID = "ewogICAgImVudGl0bGVtZW50cyI6IHsKICAgICAgICAiaW5zaWdodHMiOiB7CiAgICAgICAgICAgICJpc19lbnRpdGxlZCI6IHRydWUKICAgICAgICB9LAogICAgICAgICJjb3N0X21hbmFnZW1lbnQiOiB7CiAgICAgICAgICAgICJpc19lbnRpdGxlZCI6IHRydWUKICAgICAgICB9LAogICAgICAgICJhbnNpYmxlIjogewogICAgICAgICAgICAiaXNfZW50aXRsZWQiOiB0cnVlCiAgICAgICAgfSwKICAgICAgICAib3BlbnNoaWZ0IjogewogICAgICAgICAgICAiaXNfZW50aXRsZWQiOiB0cnVlCiAgICAgICAgfSwKICAgICAgICAic21hcnRfbWFuYWdlbWVudCI6IHsKICAgICAgICAgICAgImlzX2VudGl0bGVkIjogdHJ1ZQogICAgICAgIH0sCiAgICAgICAgIm1pZ3JhdGlvbnMiOiB7CiAgICAgICAgICAgICJpc19lbnRpdGxlZCI6IHRydWUKICAgICAgICB9CiAgICB9LAogICAgImlkZW50aXR5IjogewogICAgICAgICJpbnRlcm5hbCI6IHsKICAgICAgICAgICAgImF1dGhfdGltZSI6IDI5OSwKICAgICAgICAgICAgImF1dGhfdHlwZSI6ICJiYXNpYy1hdXRoIiwKICAgICAgICAgICAgIm9yZ19pZCI6ICIxMTc4OTc3MiIKICAgICAgICB9LAogICAgICAgICJhY2NvdW50X251bWJlciI6ICI2MDg5NzE5IiwKICAgICAgICAidXNlciI6IHsKICAgICAgICAgICAgImZpcnN0X25hbWUiOiAiSW5zaWdodHMiLAogICAgICAgICAgICAiaXNfYWN0aXZlIjogdHJ1ZSwKICAgICAgICAgICAgImlzX2ludGVybmFsIjogZmFsc2UsCiAgICAgICAgICAgICJsYXN0X25hbWUiOiAiUUEiLAogICAgICAgICAgICAibG9jYWxlIjogImVuX1VTIiwKICAgICAgICAgICAgImlzX29yZ19hZG1pbiI6IHRydWUsCiAgICAgICAgICAgICJ1c2VybmFtZSI6ICJpbnNpZ2h0cy1xYSIsCiAgICAgICAgICAgICJlbWFpbCI6ICJqbmVlZGxlK3FhQHJlZGhhdC5jb20iLAogICAgICAgICAgICAidXNlcl9pZCI6ICI2MDg5NzE5IgogICAgICAgIH0sCiAgICAgICAgInR5cGUiOiAiVXNlciIKICAgIH0KfQ==" //nolint:lll
var identityStringWithServiceAccountUserID = "ewogICAgImVudGl0bGVtZW50cyI6IHsKICAgICAgICAiaW5zaWdodHMiOiB7CiAgICAgICAgICAgICJpc19lbnRpdGxlZCI6IHRydWUKICAgICAgICB9LAogICAgICAgICJjb3N0X21hbmFnZW1lbnQiOiB7CiAgICAgICAgICAgICJpc19lbnRpdGxlZCI6IHRydWUKICAgICAgICB9LAogICAgICAgICJhbnNpYmxlIjogewogICAgICAgICAgICAiaXNfZW50aXRsZWQiOiB0cnVlCiAgICAgICAgfSwKICAgICAgICAib3BlbnNoaWZ0IjogewogICAgICAgICAgICAiaXNfZW50aXRsZWQiOiB0cnVlCiAgICAgICAgfSwKICAgICAgICAic21hcnRfbWFuYWdlbWVudCI6IHsKICAgICAgICAgICAgImlzX2VudGl0bGVkIjogdHJ1ZQogICAgICAgIH0sCiAgICAgICAgIm1pZ3JhdGlvbnMiOiB7CiAgICAgICAgICAgICJpc19lbnRpdGxlZCI6IHRydWUKICAgICAgICB9CiAgICB9LAogICAgImlkZW50aXR5IjogewogICAgICAgICJpbnRlcm5hbCI6IHsKICAgICAgICAgICAgImF1dGhfdGltZSI6IDI5OSwKICAgICAgICAgICAgImF1dGhfdHlwZSI6ICJiYXNpYy1hdXRoIiwKICAgICAgICAgICAgIm9yZ19pZCI6ICIxMTc4OTc3MiIKICAgICAgICB9LAogICAgICAgICJhY2NvdW50X251bWJlciI6ICI2MDg5NzE5IiwKICAgICAgICAic2VydmljZV9hY2NvdW50IjogewogICAgICAgICAgICAidXNlcl9pZCI6ICI2MDg5NzE5IgogICAgICAgIH0sCiAgICAgICAgInR5cGUiOiAiVXNlciIKICAgIH0KfQ=="                                                                                                                                                                                                                                                                                                                                                               //nolint:lll

func TestParseIdentity(t *testing.T) {
	xrhid, err := ParseXRHID(identityStringWithUserID)

	assert.Equal(t, nil, err)
	assert.Equal(t, "6089719", xrhid.Identity.AccountNumber)
	assert.Equal(t, "6089719", xrhid.Identity.User.UserID)

	xrhid, err = ParseXRHID(identityStringWithServiceAccountUserID)

	assert.Equal(t, nil, err)
	assert.Equal(t, "6089719", xrhid.Identity.AccountNumber)
	assert.Equal(t, "6089719", xrhid.Identity.ServiceAccount.UserId)
}

func TestEncodeXRHID(t *testing.T) {
	testIdentityString := "eyJpZGVudGl0eSI6eyJvcmdfaWQiOiJvcmdfMSIsImludGVybmFsIjp7Im9yZ19pZCI6Im9yZ18xIn0sInR5cGUiOiJVc2VyIn0sImVudGl0bGVtZW50cyI6bnVsbH0=" //nolint:lll
	orgID := "org_1"
	var xrhid identity.Identity
	xrhid.OrgID = orgID
	xrhid.Internal.OrgID = orgID
	xrhid.Type = "User"
	identityString, err := EncodeXRHID(xrhid)

	assert.Equal(t, nil, err)
	assert.Equal(t, testIdentityString, identityString)
}

func TestContentSourcesXRHIDForOrg(t *testing.T) {
	const testUser = "test-user"
	orgID := "1234"
	xrhid := XRHIDForOrg(orgID, testUser)

	assert.Equal(t, orgID, xrhid.Identity.OrgID)
	assert.Equal(t, orgID, xrhid.Identity.Internal.OrgID)
	assert.Equal(t, "User", xrhid.Identity.Type)
	assert.Equal(t, testUser, xrhid.Identity.User.Username)
}

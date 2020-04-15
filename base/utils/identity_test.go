package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseIdentity(t *testing.T) {
	str := "eyJlbnRpdGxlbWVudHMiOnsiaW5zaWdodHMiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJjb3N0X21hbmFnZW1lbnQiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJhbnNpYmxlIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwib3BlbnNoaWZ0Ijp7ImlzX2VudGl0bGVkIjp0cnVlfSwic21hcnRfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1ZX0sIm1pZ3JhdGlvbnMiOnsiaXNfZW50aXRsZWQiOnRydWV9fSwiaWRlbnRpdHkiOnsiaW50ZXJuYWwiOnsiYXV0aF90aW1lIjoyOTksImF1dGhfdHlwZSI6ImJhc2ljLWF1dGgiLCJvcmdfaWQiOiIxMTc4OTc3MiJ9LCJhY2NvdW50X251bWJlciI6IjYwODk3MTkiLCJ1c2VyIjp7ImZpcnN0X25hbWUiOiJJbnNpZ2h0cyIsImlzX2FjdGl2ZSI6dHJ1ZSwiaXNfaW50ZXJuYWwiOmZhbHNlLCJsYXN0X25hbWUiOiJRQSIsImxvY2FsZSI6ImVuX1VTIiwiaXNfb3JnX2FkbWluIjp0cnVlLCJ1c2VybmFtZSI6Imluc2lnaHRzLXFhIiwiZW1haWwiOiJqbmVlZGxlK3FhQHJlZGhhdC5jb20ifSwidHlwZSI6IlVzZXIifX0=" //nolint:lll
	identity, err := ParseIdentity(str)

	entitlements := map[string]Entitlement{
		"smart_management": {Entitled: true},
		"migrations":       {Entitled: true},
		"insights":         {Entitled: true},
		"cost_management":  {Entitled: true},
		"ansible":          {Entitled: true},
		"openshift":        {Entitled: true},
	}

	assert.Equal(t, nil, err)
	assert.Equal(t, entitlements, identity.Entitlements)
	assert.Equal(t, "6089719", identity.Identity.AccountNumber)
	assert.Equal(t, true, identity.IsSmartEntitled())
	identity.Entitlements["smart_management"] = Entitlement{Entitled: false}
	assert.Equal(t, false, identity.IsSmartEntitled())
}

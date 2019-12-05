package utils

import (
	"encoding/base64"
	"encoding/json"
)

type Entitlement struct {
	Entitled bool `json:"is_entitled"`
}

type IdentityDetail struct {
	AccountNumber string `json:"account_number"`
	Type          string `json:"type"`
	// Additional information, we don't parse this
	Internal map[string]interface{}
}

type Identity struct {
	Entitlements map[string]Entitlement `json:"entitlements"`
	Identity     IdentityDetail         `json:"identity"`
}

func ParseIdentity(identityString string) (*Identity, error) {
	var ident Identity
	decoded, err := base64.StdEncoding.DecodeString(identityString)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(decoded, &ident)
	if err != nil {
		return nil, err
	}
	return &ident, nil
}

func (this *Identity) IsSmartEntitled() bool {
	mgmt, contains := this.Entitlements["smart_management"]
	if contains {
		return mgmt.Entitled
	}
	return false
}

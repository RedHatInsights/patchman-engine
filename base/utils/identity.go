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

func (t Identity) Encode() (string, error) {
	data, err := json.Marshal(&t)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

func (t *Identity) IsSmartEntitled() bool {
	mgmt, contains := t.Entitlements["smart_management"]
	if contains {
		return mgmt.Entitled
	}
	return false
}

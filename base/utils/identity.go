package utils

import (
	"encoding/base64"
	"encoding/json"

	"github.com/redhatinsights/identity"
)

func ParseXRHID(identityString string) (*identity.XRHID, error) {
	var xrhid identity.XRHID

	decoded, err := base64.StdEncoding.DecodeString(identityString)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(decoded, &xrhid)
	if err != nil {
		return nil, err
	}
	return &xrhid, nil
}

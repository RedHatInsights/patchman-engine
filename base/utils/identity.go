package utils

import (
	"encoding/base64"
	"encoding/json"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

func ParseIdentity(identityString string) (*identity.Identity, error) {
	var ident identity.XRHID

	decoded, err := base64.StdEncoding.DecodeString(identityString)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(decoded, &ident)
	if err != nil {
		return nil, err
	}
	return &ident.Identity, nil
}

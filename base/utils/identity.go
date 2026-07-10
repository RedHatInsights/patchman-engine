package utils

import (
	"encoding/base64"

	"github.com/bytedance/sonic"
	"github.com/pkg/errors"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
)

var ERRUserIDNotFound = errors.New("user_id not found in identity string")

func ParseXRHID(identityString string) (*identity.XRHID, error) {
	var xrhid identity.XRHID

	decoded, err := base64.StdEncoding.DecodeString(identityString)
	if err != nil {
		return nil, err
	}
	err = sonic.Unmarshal(decoded, &xrhid)
	if err != nil {
		return nil, err
	}
	return &xrhid, nil
}

func EncodeXRHID(id identity.Identity) (string, error) {
	js, err := sonic.Marshal(identity.XRHID{Identity: id})
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(js), nil
}

func XRHIDForOrg(orgID string, username string) identity.XRHID {
	return identity.XRHID{
		Identity: identity.Identity{
			Type: "User",
			User: &identity.User{
				Username: username,
			},
			OrgID: orgID,
			Internal: identity.Internal{
				OrgID: orgID,
			},
		},
	}
}

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

package utils

import (
	"encoding/base64"
	"encoding/json"
	"reflect"

	"github.com/redhatinsights/platform-go-middlewares/identity"
)

type Identity identity.Identity
type XRHID struct {
	Identity Identity `json:"identity"`
}

func ParseIdentity(identityString string) (*Identity, error) {
	var ident XRHID

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

// Fallback if platform removes AccountNumber from Identity struct
func (ident Identity) GetAccountNumber() *string {
	val := reflect.ValueOf(&ident).Elem().FieldByName("AccountNumber")
	if val.IsValid() {
		res := val.Interface().(string)
		return &res
	}
	return nil
}

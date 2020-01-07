package utils

import (
	"github.com/pkg/errors"
	"regexp"
)

var (
	nevraRegex *regexp.Regexp
)

func init() {
	nevraRegex = regexp.MustCompile(
		`((?P<e1>[0-9]+):)?(?P<pn>[^:]+)-((?P<e2>[0-9]+):)?(?P<ver>[^-:]+)-(?P<rel>[^-:]+)\.(?P<arch>[a-z0-9_]+)`)
}

type Nevra struct {
	Name    string
	Epoch   string
	Version string
	Release string
	Arch    string
}

// parse package components
func ParseNevra(nevra string) (*Nevra, error) {
	parsed := nevraRegex.FindStringSubmatch(nevra)
	if len(parsed) != 9 {
		return nil, errors.New("unable to parse nevra")
	}
	res := Nevra{
		Name: parsed[3],
		Epoch: parsed[2],
		Version: parsed[6],
		Release: parsed[7],
		Arch: parsed[8],
	}
	return &res, nil
}

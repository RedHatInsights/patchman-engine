package utils

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
)

var (
	nevraRegex = regexp.MustCompile(
		`((?P<e1>[0-9]+):)?(?P<pn>[^:]+)-((?P<e2>[0-9]+):)?(?P<ver>[^-:]+)-(?P<rel>[^-:]+)\.(?P<arch>[a-z0-9_]+)`)
	nevraRegexIndices map[string]int
)

func init() {
	nevraRegexIndices = make(map[string]int)
	for i, name := range nevraRegex.SubexpNames() {
		if i != 0 && name != "" {
			nevraRegexIndices[name] = i
		}
	}
}

type Nevra struct {
	Name    string
	Epoch   int
	Version string
	Release string
	Arch    string
}

func ParseNevra(nevra string) (*Nevra, error) {
	parsed := nevraRegex.FindStringSubmatch(nevra)

	if len(parsed) != 9 {
		return nil, errors.Errorf("unable to parse (%s)", nevra)
	}
	var err error
	epoch := 0
	if parsed[2] != "" {
		epoch, err = strconv.Atoi(parsed[2])
		if err != nil {
			return nil, err
		}
	} else if parsed[5] != "" {
		epoch, err = strconv.Atoi(parsed[5])
		if err != nil {
			return nil, err
		}
	}
	res := Nevra{
		Name:    parsed[3],
		Epoch:   epoch,
		Version: parsed[6],
		Release: parsed[7],
		Arch:    parsed[8],
	}
	return &res, nil
}

func ParseNameEVRA(name, evra string) (*Nevra, error) {
	return ParseNevra(fmt.Sprintf("%s-%s", name, evra))
}

func (n Nevra) StringE(showEpoch bool) string {
	if n.Epoch != 0 || showEpoch {
		return fmt.Sprintf("%s-%d:%s-%s.%s", n.Name, n.Epoch, n.Version, n.Release, n.Arch)
	}
	return fmt.Sprintf("%s-%s-%s.%s", n.Name, n.Version, n.Release, n.Arch)
}

func (n Nevra) String() string {
	return n.StringE(false)
}

func (n Nevra) EVRStringE(showEpoch bool) string {
	if n.Epoch != 0 || showEpoch {
		return fmt.Sprintf("%d:%s-%s", n.Epoch, n.Version, n.Release)
	}
	return fmt.Sprintf("%s-%s", n.Version, n.Release)
}

func (n Nevra) EVRString() string {
	return n.EVRStringE(false)
}

func (n Nevra) EVRAStringE(showEpoch bool) string {
	if n.Epoch != 0 || showEpoch {
		return fmt.Sprintf("%d:%s-%s.%s", n.Epoch, n.Version, n.Release, n.Arch)
	}
	return fmt.Sprintf("%s-%s.%s", n.Version, n.Release, n.Arch)
}

func (n Nevra) EVRAString() string {
	return n.EVRAStringE(false)
}

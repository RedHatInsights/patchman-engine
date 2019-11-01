package utils

import "strings"
import "errors"

type Nevra struct {
	Name    string
	Epoch   string
	Version string
	Release string
	Arch    string
}

func ParseRpmName(rpmName string) (*Nevra, error) {
	chunks := strings.Split(rpmName, ".")
	if len(chunks) == 0 {
		return nil, errors.New("unable to parse arch using .")
	}
	res := Nevra{Arch: chunks[len(chunks)-1]}
	return &res, nil
}

package base

import (
	"errors"
)

var (
	ErrDatabase   = errors.New("database error")
	ErrBadRequest = errors.New("bad request")
	ErrNotFound   = errors.New("not found")
)

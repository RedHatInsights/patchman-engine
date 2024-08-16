package base

import (
	stdErrors "errors"

	"github.com/pkg/errors"
)

var (
	ErrDatabase   = errors.New("database error")
	ErrKafka      = errors.New("kafka error")
	ErrBadRequest = errors.New("bad request")
	ErrNotFound   = errors.New("not found")
	ErrFatal      = errors.New("fatal error restarting pod")
)

func WrapFatalError(err error, message string) error {
	return wrapErrors(err, message)
}

func WrapFatalDBError(err error, message string) error {
	return wrapErrors(err, message, ErrFatal, ErrDatabase)
}

func WrapFatalKafkaError(err error, message string) error {
	return wrapErrors(err, message, ErrFatal, ErrKafka)
}

func wrapErrors(err error, message string, errs ...error) error {
	if err == nil {
		return nil
	}
	errsJoined := stdErrors.Join(errs...)
	err = stdErrors.Join(errsJoined, err)
	return errors.Wrap(err, message)
}

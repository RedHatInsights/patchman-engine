package utils

import (
	"errors"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestFindSentryExceptionUsesErrField(t *testing.T) {
	err := errors.New("boom")

	got := findErrorInFieldsOrMsg(log.Fields{"err": err}, "processing failed")

	assert.Same(t, err, got)
}

func TestFindSentryExceptionFallsBackToMessage(t *testing.T) {
	err := errors.New("boom")

	got := findErrorInFieldsOrMsg(log.Fields{}, err)

	assert.Same(t, err, got)
}

func TestFindSentryExceptionIgnoresStringifiedError(t *testing.T) {
	got := findErrorInFieldsOrMsg(log.Fields{"err": "boom"}, "processing failed")

	assert.Nil(t, got)
}

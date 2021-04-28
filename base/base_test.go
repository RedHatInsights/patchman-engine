package base

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTimeFormats(t *testing.T) {
	_, err := ParseTime("2021-03-09T12:35:50")
	assert.NoError(t, err)
	_, err = ParseTime("2021-04-28T08:26:54.629140+00:00")
	assert.NoError(t, err)
}

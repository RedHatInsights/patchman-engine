package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const myTestEnv = "MY_TEST_CONFIG"

func TestPodConfig(t *testing.T) {
	os.Setenv(myTestEnv, "bool_val1=true;bool_val2=false;no_val;int_val=123;int64_val=17179869184;str_val=a b c")
	pc := ReadPodConfig(myTestEnv)
	assert.True(t, pc.GetBool("bool_val1", false))
	assert.False(t, pc.GetBool("bool_val2", true))
	assert.True(t, pc.GetBool("no_val", false))
	assert.Equal(t, 123, pc.GetInt("int_val", 999))
	assert.Equal(t, int64(17179869184), pc.GetInt64("int64_val", 999))
	assert.Equal(t, "a b c", pc.GetString("str_val", "nope"))

	// check defaults
	assert.True(t, pc.GetBool("undefined", true))
	assert.Equal(t, 1, pc.GetInt("undefined", 1))
	assert.Equal(t, int64(1), pc.GetInt64("undefined", 1))
	assert.Equal(t, "nope", pc.GetString("undefined", "nope"))
}

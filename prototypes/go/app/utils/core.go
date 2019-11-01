package utils

import (
	"fmt"
	"os"
	"strconv"
)

// load environment variable or return default value
func Getenv(key, defaultt string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultt
}

// load environment variable or fail
func GetenvOrFail(envname string) string {
	value := os.Getenv(envname)
	if value == "" {
		panic(fmt.Sprintf("Set %s env variable!", envname))
	}

	return value
}

// load int environment variable or fail
func GetIntEnvOrFail(envname string) int {
	valueStr := os.Getenv(envname)
	if valueStr == "" {
		panic(fmt.Sprintf("Set %s env variable!", envname))
	}

	value, err:= strconv.Atoi(valueStr)
	if err != nil {
		panic(fmt.Sprintf("Unable convert '%s' env var '%s' to int!", envname, valueStr))
	}

	return value
}

// set environment variable or fail
func SetenvOrFail(envname, value string) string {
	err := os.Setenv(envname, value)
	if err != nil {
		panic(err)
	}

	return value
}

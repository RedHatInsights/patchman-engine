package utils

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
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

// parse bool value from env variable
func GetBoolEnvOrFail(envname string) bool {
	value := os.Getenv(envname)
	if value == "" {
		panic(fmt.Sprintf("Set %s env variable!", envname))
	}

	parsedBool, err := strconv.ParseBool(value)
	if err != nil {
		panic(err)
	}

	return parsedBool
}

// parse bool value from env variable
func GetBoolEnvOrDefault(envname string, defval bool) bool {
	value := os.Getenv(envname)
	if value == "" {
		return defval
	}

	parsedBool, err := strconv.ParseBool(value)
	if err != nil {
		panic(err)
	}

	return parsedBool
}

// load int environment variable or fail
func GetIntEnvOrFail(envname string) int {
	valueStr := os.Getenv(envname)
	if valueStr == "" {
		panic(fmt.Sprintf("Set %s env variable!", envname))
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		panic(fmt.Sprintf("Unable convert '%s' env var '%s' to int!", envname, valueStr))
	}

	return value
}

// load int environment variable or load default
func GetIntEnvOrDefault(envname string, defval int) int {
	valueStr := os.Getenv(envname)
	if valueStr == "" {
		return defval
	}

	value, err := strconv.Atoi(valueStr)
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

func TestLoadEnv(files ...string) {
	// go test changes working directory to package's location, we utilize env variable to project working directory
	dir := Getenv("TEST_WD", ".")
	for i, f := range files {
		files[i] = path.Join(dir, f)
	}
	err := godotenv.Overload(files...)

	Log("files", files).Debug("Loading new env file")
	Log("dbuser", Getenv("DB_USER", ""), "passwd", Getenv("DB_PASSWD", "")).Debug("Db auth info")
	if err != nil {
		Log().Panic("Could not load env file")
	}
}

// Catches panics, and logs them to stderr, then exit conditionally
func LogPanics(exitAfterLogging bool) {
	if obj := recover(); obj != nil {
		stack := string(debug.Stack())
		stackLine := strings.Replace(stack, "\n", "|", -1)
		Log("err", obj, "stack", stackLine).Error("Panicked")
		FlushLogs()
		if exitAfterLogging {
			os.Exit(1)
		}
	}
}

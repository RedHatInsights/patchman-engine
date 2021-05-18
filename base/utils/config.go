package utils

import (
	"fmt"
	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
)

// IsClowderEnabled Check env variable CLOWDER_ENABLED = "true".
func IsClowderEnabled() bool {
	clowderEnabled := GetBoolEnvOrDefault("CLOWDER_ENABLED", false)
	return clowderEnabled
}

// PrintClowderParams Print Clowder params to export environment variables.
func PrintClowderParams() {
	fmt.Println("Trying to export variables from Clowder")
	if IsClowderEnabled() {
		fmt.Println("Clowder config enabled, exporting variables..")
		// Database
		fmt.Printf("DB_ADMIN_USER=%s\n", clowder.LoadedConfig.Database.AdminUsername)
		fmt.Printf("DB_ADMIN_PASSWD=%s\n", clowder.LoadedConfig.Database.AdminPassword)
		fmt.Printf("DB_HOST=%s\n", clowder.LoadedConfig.Database.Hostname)
		fmt.Printf("DB_NAME=%s\n", clowder.LoadedConfig.Database.Name)
		fmt.Printf("DB_PORT=%d\n", clowder.LoadedConfig.Database.Port)
		fmt.Printf("DB_SSLMODE=%s\n", clowder.LoadedConfig.Database.SslMode)
		// API
		fmt.Printf("PUBLIC_PORT=%d\n", *clowder.LoadedConfig.PublicPort)
		fmt.Printf("PRIVATE_PORT=%d\n", *clowder.LoadedConfig.PrivatePort)
		fmt.Printf("METRICS_PORT=%d\n", clowder.LoadedConfig.MetricsPort)
		fmt.Printf("METRICS_PATH=%s\n", clowder.LoadedConfig.MetricsPath)
		fmt.Println("...done")
	} else {
		fmt.Println("Clowder not enabled")
	}
}

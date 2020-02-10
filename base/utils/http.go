package utils

import (
	"fmt"
	"net/http"
)

// If possible return some details about response for logging purposes.
// E.g. response status code. Return empty string on nil input.
func TryGetResponseDetails(response *http.Response) string {
	details := ""
	if response != nil {
		details = fmt.Sprintf(", status code: %d", response.StatusCode)
	}
	return details
}

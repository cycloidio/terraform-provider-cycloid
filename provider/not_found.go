package provider

import (
	"errors"
	"net/http"
	"strings"

	cycloidapiclient "github.com/cycloidio/cycloid-cli/cmd/apiclient"
)

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// A typed 404 is an unambiguous not-found signal.
	var apiErr *cycloidapiclient.APIResponseError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		return true
	}
	// Some backends (e.g. plugin-manager) report a missing object as a 422
	// validation error with a "was not found" message instead of a 404. Fall
	// back to matching the message so those still resolve to state removal
	// instead of a hard error.
	errMessage := strings.ToLower(err.Error())
	return strings.Contains(errMessage, " not found") ||
		strings.Contains(errMessage, "notfound") ||
		(strings.Contains(errMessage, "404") && strings.Contains(errMessage, "returned"))
}

// isConflictError returns true when the Cycloid API responds with a
// 409 Conflict status (e.g. resource already exists or still in use).
// Only matches typed *APIResponseError; plain errors are not matched
func isConflictError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *cycloidapiclient.APIResponseError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusConflict
	}
	return false
}

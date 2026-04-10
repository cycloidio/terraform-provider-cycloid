package provider

import (
	"errors"
	"net/http"
	"strings"

	cycloidmiddleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
)

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	errMessage := strings.ToLower(err.Error())
	return strings.Contains(errMessage, " not found") ||
		strings.Contains(errMessage, "notfound") ||
		(strings.Contains(errMessage, "404") && strings.Contains(errMessage, "returned"))
}

// isCredentialInUseError returns true when the Cycloid API refuses a credential
// deletion because the credential is still referenced by another resource
// (e.g. a config repository that was just deleted but whose deletion has not
// yet propagated). A 409 Conflict status is used as the canonical signal.
func isCredentialInUseError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *cycloidmiddleware.APIResponseError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusConflict
	}
	return false
}

package provider

import "strings"

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	errMessage := strings.ToLower(err.Error())
	return strings.Contains(errMessage, " not found") ||
		strings.Contains(errMessage, "notfound") ||
		(strings.Contains(errMessage, "404") && strings.Contains(errMessage, "returned"))
}

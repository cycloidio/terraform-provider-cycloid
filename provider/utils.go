package provider

import (
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// normalizeURL ensures the URL has a valid protocol scheme (http:// or https://).
// If no scheme is provided, it defaults to http://.
func normalizeURL(url string) string {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	return "http://" + url
}

var allowedCanonicalCharRegex = regexp.MustCompile("[^a-zA-Z0-9-_]")

func getOrganizationCanonical(provider CycloidProvider, dataOrgCan types.String) string {
	if org := dataOrgCan.ValueString(); org != "" {
		return org
	}

	return provider.DefaultOrganization
}

// ToCanonical convert a name to a valid canonical
func ToCanonical(name string) string {
	replacer := strings.NewReplacer(
		" ", "_",
	)

	replaced := replacer.Replace(name)
	filtered := allowedCanonicalCharRegex.ReplaceAllString(replaced, "")
	trimmed := strings.Trim(filtered, "-_")
	return strings.ToLower(trimmed)
}

// NameOrCanonical will process name and canonical argument and return both
// if name is set, the canonical will be inferred from it
// if canonical is set, name will be a Capitalized version of the canonical
// if both are empty, you will get an error
func NameOrCanonical(name, canonical string) (string, string, error) {
	if canonical == "" {
		return name, ToCanonical(name), nil
	}

	if name == "" {
		return strings.ToUpper(canonical[:1]) + canonical[1:], canonical, nil
	}

	return name, canonical, nil
}

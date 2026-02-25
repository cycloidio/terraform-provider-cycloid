package provider

import (
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

func getOrganizationCanonical(provider CycloidProvider, dataOrgCan types.String) string {
	if org := dataOrgCan.ValueString(); org != "" {
		return org
	}

	return provider.DefaultOrganization
}

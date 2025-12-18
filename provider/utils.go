package provider

import (
	"errors"
	"strings"

	"github.com/cycloidio/cycloid-cli/cmd/cycloid/common"
	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
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

func getOrganizationCanonical(pv provider_cycloid.CycloidModel, dataOrgCan types.String) string {
	orgCan := pv.OrganizationCanonical.ValueString()

	if dOrgCan := dataOrgCan.ValueString(); dOrgCan != "" {
		orgCan = dOrgCan
	}
	return orgCan
}

func getDefaultApi(provider provider_cycloid.CycloidModel) (*common.APIClient, error) {
	if provider.Jwt.IsNull() || provider.Jwt.IsUnknown() {
		return nil, errors.New("Cycloid API key not set in provider")
	}
	return common.NewAPI(common.WithURL(provider.Url.ValueString()), common.WithToken(provider.Jwt.ValueString())), nil
}

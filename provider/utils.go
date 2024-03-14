package provider

import (
	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func getOrganizationCanonical(pv provider_cycloid.CycloidModel, dataOrgCan types.String) string {
	orgCan := pv.OrganizationCanonical.ValueString()

	if dOrgCan := dataOrgCan.ValueString(); dOrgCan != "" {
		orgCan = dOrgCan
	}
	return orgCan
}

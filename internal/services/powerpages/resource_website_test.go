// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package powerpages_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/microsoft/terraform-provider-power-platform/internal/mocks"
)

func TestAccPowerPagesWebsiteResource_Validate_Create(t *testing.T) {
	t.Setenv("TF_ACC", "1")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: mocks.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				
				resource "powerplatform_powerpages_website" "powerpage" {
					environment_id = "0eaf704f-8c3a-e49f-b64d-69e85d0a5b52"
					name           = "my power page1"
					language_lcid  = 1031
					subdomain      = "mwpage1"
					template_name  = "DefaultPortalTemplate"
				}`,

				Check: resource.ComposeAggregateTestCheckFunc(),
			},
		},
	})
}

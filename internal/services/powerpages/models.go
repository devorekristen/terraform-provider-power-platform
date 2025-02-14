// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package powerpages

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/microsoft/terraform-provider-power-platform/internal/helpers"
)

type WebsiteResource struct {
	helpers.TypeInfo
	PowerPagesClient client
}

type WebsiteResourceModel struct {
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
	Id            types.String   `tfsdk:"id"`
	EnvironmentId types.String   `tfsdk:"environment_id"`
	Name          types.String   `tfsdk:"name"`
	LanguageLCID  types.Int32    `tfsdk:"language_lcid"`
	Subdomain     types.String   `tfsdk:"subdomain"`
	TemplateName  types.String   `tfsdk:"template_name"`
}

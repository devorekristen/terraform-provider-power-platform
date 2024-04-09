// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package powerplatform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	api "github.com/microsoft/terraform-provider-power-platform/internal/powerplatform/api"
	helpers "github.com/microsoft/terraform-provider-power-platform/internal/powerplatform/helpers"
)

var _ resource.Resource = &ApplicationUserResource{}
var _ resource.ResourceWithImportState = &ApplicationUserResource{}

func NewApplicationUserResource() resource.Resource {
	return &ApplicationUserResource{
		ProviderTypeName: "powerplatform",
		TypeName:         "_application_user",
	}
}

type ApplicationUserResource struct {
	ApplicationUserClient ApplicationUserClient
	ProviderTypeName      string
	TypeName              string
}

type ApplicationUserResourceModel struct {
	Id              types.String `tfsdk:"Applicationid"`
	ApplicationName types.String `tfsdk:"first_name"`
	EnvironmentId   types.String `tfsdk:"environment_id"`
	BusinessUnitId  types.String `tfsdk:"business_unit_id"`
	SecurityRoles   []string     `tfsdk:"security_roles"`
	DisableDelete   types.Bool   `tfsdk:"disable_delete"`
}

func (r *ApplicationUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.TypeName
}

func (r *ApplicationUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {

	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource associates a application user to a Power Platform environment. Additional Resources:\n\n* [Add application users to an environment](https://learn.microsoft.com/power-platform/admin/manage-application-users)\n\n* [Overview of User Security](https://learn.microsoft.com/en-us/power-platform/admin/grant-users-access)",
		Description:         "This resource associates a application user to a Power Platform environment",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique application id (guid)",
				Description:         "Unique application id (guid)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Unique environment id (guid)",
				Description:         "Unique environment id (guid)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"business_unit_id": schema.StringAttribute{
				Description: "Id of the business unit to which the user belongs",
				Computed:    true,
			},
			"security_roles": schema.SetAttribute{
				MarkdownDescription: "Security roles Ids assigned to the user",
				Description:         "Security roles Ids assigned to the user",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
			},
			"application_name": schema.StringAttribute{
				MarkdownDescription: "User principal name",
				Description:         "User principal name",
				Computed:            true,
			},
			"disable_delete": schema.BoolAttribute{
				MarkdownDescription: "Disable. When set to `True` is expects that (Disable)[https://learn.microsoft.com/power-platform/admin/manage-application-users#activate-or-deactivate-an-application-user] feature to be enabled.",
				Description:         "Disable. Disable application user from Dataverse if it was already removed from Entra.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
		},
	}
}

func (r *ApplicationUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clientApi := req.ProviderData.(*api.ProviderClient).Api

	if clientApi == nil {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.ApplicationUserClient = NewApplicationUserClient(clientApi)
}

func (r *ApplicationUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan *ApplicationUserResourceModel

	tflog.Debug(ctx, fmt.Sprintf("CREATE RESOURCE START: %s", r.ProviderTypeName))

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ApplicationUserDto, err := r.ApplicationUserClient.CreateApplicationUser(ctx, plan.EnvironmentId.ValueString(), plan.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Client error when creating %s_%s", r.ProviderTypeName, r.TypeName), err.Error())
		return
	}

	ApplicationUserDto, err = r.ApplicationUserClient.AddSecurityRoles(ctx, plan.EnvironmentId.ValueString(), ApplicationUserDto.Id, plan.SecurityRoles)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Client error when creating %s_%s", r.ProviderTypeName, r.TypeName), err.Error())
		return
	}

	model := ConvertFromApplicationUserDto(ApplicationUserDto, plan.DisableDelete.ValueBool())

	plan.Id = model.Id
	req.Plan.SetAttribute(ctx, path.Root("security_roles"), model.SecurityRoles)
	plan.ApplicationName = model.ApplicationName
	plan.DisableDelete = model.DisableDelete
	plan.BusinessUnitId = model.BusinessUnitId

	tflog.Trace(ctx, fmt.Sprintf("created a resource with ID %s", plan.Id.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Debug(ctx, fmt.Sprintf("CREATE RESOURCE END: %s", r.ProviderTypeName))
}

func (r *ApplicationUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state *ApplicationUserResourceModel

	tflog.Debug(ctx, fmt.Sprintf("READ RESOURCE START: %s", r.ProviderTypeName))

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ApplicationUserDto, err := r.ApplicationUserClient.GetApplicationUserBySystemApplicationUserId(ctx, state.EnvironmentId.ValueString(), state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Client error when reading %s_%s", r.ProviderTypeName, r.TypeName), err.Error())
		return
	}

	model := ConvertFromApplicationUserDto(ApplicationUserDto, state.DisableDelete.ValueBool())

	state.Id = model.Id
	state.SecurityRoles = model.SecurityRoles
	state.ApplicationName = model.ApplicationName
	state.BusinessUnitId = model.BusinessUnitId
	state.DisableDelete = model.DisableDelete

	tflog.Debug(ctx, fmt.Sprintf("READ: %s_environment with id %s", r.ProviderTypeName, state.Id.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Debug(ctx, fmt.Sprintf("READ RESOURCE END: %s", r.ProviderTypeName))
}

func (r *ApplicationUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan *ApplicationUserResourceModel

	tflog.Debug(ctx, fmt.Sprintf("UPDATE RESOURCE START: %s", r.ProviderTypeName))

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var state *ApplicationUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	addedSecurityRoles, removedSecurityRoles := helpers.DiffArrays(plan.SecurityRoles, state.SecurityRoles)

	ApplicationUser, err := r.ApplicationUserClient.GetApplicationUserBySystemApplicationUserId(ctx, plan.EnvironmentId.ValueString(), state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Client error when reading %s_%s", r.ProviderTypeName, r.TypeName), err.Error())
		return
	}

	if len(addedSecurityRoles) > 0 {
		ApplicationUserDto, err := r.ApplicationUserClient.AddSecurityRoles(ctx, plan.EnvironmentId.ValueString(), state.Id.ValueString(), addedSecurityRoles)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Client error when adding security roles %s_%s", r.ProviderTypeName, r.TypeName), err.Error())
			return
		}
		ApplicationUser = ApplicationUserDto
	}
	if len(removedSecurityRoles) > 0 {
		ApplicationUserDto, err := r.ApplicationUserClient.RemoveSecurityRoles(ctx, plan.EnvironmentId.ValueString(), state.Id.ValueString(), removedSecurityRoles)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Client error when removing security roles %s_%s", r.ProviderTypeName, r.TypeName), err.Error())
			return
		}
		ApplicationUser = ApplicationUserDto
	}

	model := ConvertFromApplicationUserDto(ApplicationUser, plan.DisableDelete.ValueBool())

	plan.Id = model.Id
	req.Plan.SetAttribute(ctx, path.Root("security_roles"), model.SecurityRoles)
	plan.ApplicationName = model.ApplicationName
	plan.DisableDelete = model.DisableDelete
	plan.BusinessUnitId = model.BusinessUnitId

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Debug(ctx, fmt.Sprintf("UPDATE RESOURCE END: %s", r.ProviderTypeName))
}

func (r *ApplicationUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state *ApplicationUserResourceModel

	tflog.Debug(ctx, fmt.Sprintf("DELETE RESOURCE START: %s", r.ProviderTypeName))

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if state.DisableDelete.ValueBool() {
		err := r.ApplicationUserClient.DeleteApplicationUser(ctx, state.EnvironmentId.ValueString(), state.Id.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Client error when deleting %s_%s", r.ProviderTypeName, r.TypeName), err.Error())
			return
		}

	} else {
		tflog.Debug(ctx, fmt.Sprintf("Disable delete is set to false. Skipping delete of systemuser with id %s", state.Id.ValueString()))
	}
	tflog.Debug(ctx, fmt.Sprintf("DELETE RESOURCE END: %s", r.ProviderTypeName))
}

func (r *ApplicationUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

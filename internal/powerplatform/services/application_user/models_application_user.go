// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package powerplatform

import "github.com/hashicorp/terraform-plugin-framework/types"

//"github.com/google/uuid"

//"github.com/hashicorp/terraform-plugin-framework/types"
//"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

type ApplicationUserDto struct {
	Id              string            `json:"systemapplicationuserid"`
	ApplicationName string            `json:"applicationname"`
	BusinessUnitId  string            `json:"_businessunitid_value"`
	SecurityRoles   []SecurityRoleDto `json:"systemuserroles_association,omitempty"`
}

type SecurityRoleDto struct {
	RoleId         string `json:"roleid"`
	Name           string `json:"name"`
	IsManaged      bool   `json:"ismanaged"`
	BusinessUnitId string `json:"_businessunitid_value"`
}

type SecurityRoleDtoArray struct {
	Value []SecurityRoleDto `json:"value"`
}

func (u *ApplicationUserDto) SecurityRolesArray() []string {
	if len(u.SecurityRoles) == 0 {
		return []string{}
	} else {
		var roles []string
		for _, role := range u.SecurityRoles {
			roles = append(roles, role.RoleId)
		}
		return roles
	}
}

type ApplicationUserDtoArray struct {
	Value []ApplicationUserDto `json:"value"`
}

type EnvironmentIdDto struct {
	Id         string                     `json:"id"`
	Name       string                     `json:"name"`
	Properties EnvironmentIdPropertiesDto `json:"properties"`
}

type EnvironmentIdPropertiesDto struct {
	LinkedEnvironmentMetadata LinkedEnvironmentIdMetadataDto `json:"linkedEnvironmentMetadata"`
}

type LinkedEnvironmentIdMetadataDto struct {
	InstanceURL string
}

func ConvertFromApplicationUserDto(applicationuserDto *ApplicationUserDto, disableDelete bool) ApplicationUserResourceModel {
	model := ApplicationUserResourceModel{
		Id:              types.StringValue(applicationuserDto.Id),
		SecurityRoles:   applicationuserDto.SecurityRolesArray(),
		ApplicationName: types.StringValue(applicationuserDto.ApplicationName),
		BusinessUnitId:  types.StringValue(applicationuserDto.BusinessUnitId),
	}
	model.DisableDelete = types.BoolValue(disableDelete)
	return model
}

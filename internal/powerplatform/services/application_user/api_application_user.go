// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package powerplatform

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	api "github.com/microsoft/terraform-provider-power-platform/internal/powerplatform/api"
)

func NewApplicationUserClient(api *api.ApiClient) ApplicationUserClient {
	return ApplicationUserClient{
		Api: api,
	}
}

type ApplicationUserClient struct {
	Api *api.ApiClient
}

func (client *ApplicationUserClient) GetApplicationUsers(ctx context.Context, environmentId string) ([]ApplicationUserDto, error) {
	environmentUrl, err := client.GetEnvironmentUrlById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	apiUrl := &url.URL{
		Scheme: "https",
		Host:   strings.TrimPrefix(environmentUrl, "https://"),
		Path:   "/api/data/v9.2/systemusers",
	}
	applicationuserArray := ApplicationUserDtoArray{}
	_, err = client.Api.Execute(ctx, "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &applicationuserArray)
	if err != nil {
		return nil, err
	}
	return applicationuserArray.Value, nil
}

func (client *ApplicationUserClient) GetApplicationUserBySystemApplicationUserId(ctx context.Context, environmentId, systemApplicationUserId string) (*ApplicationUserDto, error) {
	environmentUrl, err := client.GetEnvironmentUrlById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	apiUrl := &url.URL{
		Scheme: "https",
		Host:   strings.TrimPrefix(environmentUrl, "https://"),
		Path:   "/api/data/v9.2/systemusers(" + systemApplicationUserId + ")",
	}
	values := url.Values{}
	values.Add("$expand", "systemuserroles_association($select=roleid,name,ismanaged,_businessunitid_value)")
	apiUrl.RawQuery = values.Encode()

	applicationuser := ApplicationUserDto{}
	_, err = client.Api.Execute(ctx, "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &applicationuser)
	if err != nil {
		return nil, err
	}
	return &applicationuser, nil

}

/*
	func (client *ApplicationUserClient) GetUserByAadObjectId(ctx context.Context, environmentId, aadObjectId string) (*ApplicationUserDto, error) {
		environmentUrl, err := client.GetEnvironmentUrlById(ctx, environmentId)
		if err != nil {
			return nil, err
		}
		apiUrl := &url.URL{
			Scheme: "https",
			Host:   strings.TrimPrefix(environmentUrl, "https://"),
			Path:   "/api/data/v9.2/systemusers",
		}
		values := url.Values{}
		values.Add("$filter", fmt.Sprintf("azureactivedirectoryobjectid eq %s", aadObjectId))
		values.Add("$expand", "systemapplicationuserroles_association($select=roleid,name,ismanaged,_businessunitid_value)")
		apiUrl.RawQuery = values.Encode()

		user := ApplicationUserDtoArray{}
		_, err = client.Api.Execute(ctx, "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &user)
		if err != nil {
			return nil, err
		}
		return &user.Value[0], nil
	}
*/
func (client *ApplicationUserClient) CreateApplicationUser(ctx context.Context, environmentId, systemApplicationUserId string) (*ApplicationUserDto, error) {
	apiUrl := &url.URL{
		Scheme: "https",
		Host:   client.Api.GetConfig().Urls.BapiUrl,
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/addUser", environmentId),
	}
	values := url.Values{}
	values.Add("api-version", "2023-06-01")
	apiUrl.RawQuery = values.Encode()

	applicationuserToCreate := map[string]interface{}{
		"objectId": systemApplicationUserId,
	}

	retryCount := 6 * 9 // 9 minutes of retries
	err := fmt.Errorf("")
	for retryCount > 0 {
		_, err = client.Api.Execute(ctx, "POST", apiUrl.String(), nil, applicationuserToCreate, []int{http.StatusOK}, nil)
		//the license assignment in Entra is async, so we need to wait for that to happen if a user is created in the same terraform run
		if err == nil || !strings.Contains(err.Error(), "userNotLicensed") {
			break
		}
		tflog.Debug(ctx, fmt.Sprintf("Error creating application user: %s", err.Error()))
		//lintignore:R018
		time.Sleep(10 * time.Second)
		retryCount--
	}
	if err != nil {
		return nil, err
	}

	applicationuser, err := client.GetApplicationUserBySystemApplicationUserId(ctx, environmentId, systemApplicationUserId)
	if err != nil {
		return nil, err
	}

	return applicationuser, nil
}

func (client *ApplicationUserClient) UpdateApplicationUser(ctx context.Context, environmentId, systemApplicationUserId string, applicationuserUpdate *ApplicationUserDto) (*ApplicationUserDto, error) {
	environmentUrl, err := client.GetEnvironmentUrlById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	apiUrl := &url.URL{
		Scheme: "https",
		Host:   strings.TrimPrefix(environmentUrl, "https://"),
		Path:   "/api/data/v9.2/systemusers(" + systemApplicationUserId + ")",
	}

	_, err = client.Api.Execute(ctx, "PATCH", apiUrl.String(), nil, applicationuserUpdate, []int{http.StatusOK}, nil)
	if err != nil {
		return nil, err
	}

	applicationuser, err := client.GetApplicationUserBySystemApplicationUserId(ctx, environmentId, systemApplicationUserId)
	if err != nil {
		return nil, err
	}
	return applicationuser, nil
}

func (client *ApplicationUserClient) DeleteApplicationUser(ctx context.Context, environmentId, systemApplicationUserId string) error {
	environmentUrl, err := client.GetEnvironmentUrlById(ctx, environmentId)
	if err != nil {
		return err
	}
	apiUrl := &url.URL{
		Scheme: "https",
		Host:   strings.TrimPrefix(environmentUrl, "https://"),
		Path:   "/api/data/v9.2/systemusers(" + systemApplicationUserId + ")",
	}

	_, err = client.Api.Execute(ctx, "DELETE", apiUrl.String(), nil, nil, []int{http.StatusNoContent}, nil)
	if err != nil {
		return err
	}
	return nil
}

func (client *ApplicationUserClient) RemoveSecurityRoles(ctx context.Context, environmentId, systemApplicationUserId string, securityRolesIds []string) (*ApplicationUserDto, error) {
	environmentUrl, err := client.GetEnvironmentUrlById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	for _, roleId := range securityRolesIds {
		apiUrl := &url.URL{
			Scheme: "https",
			Host:   strings.TrimPrefix(environmentUrl, "https://"),
			Path:   "/api/data/v9.2/systemusers(" + systemApplicationUserId + ")/systemuserroles_association/$ref",
		}
		values := url.Values{}
		values.Add("$id", fmt.Sprintf("%s/api/data/v9.2/roles(%s)", environmentUrl, roleId))
		apiUrl.RawQuery = values.Encode()

		_, err = client.Api.Execute(ctx, "DELETE", apiUrl.String(), nil, nil, []int{http.StatusNoContent}, nil)
		if err != nil {
			return nil, err
		}
	}

	applicationuser, err := client.GetApplicationUserBySystemApplicationUserId(ctx, environmentId, systemApplicationUserId)
	if err != nil {
		return nil, err
	}
	return applicationuser, nil
}

func (client *ApplicationUserClient) AddSecurityRoles(ctx context.Context, environmentId, systemApplicationUserId string, securityRolesIds []string) (*ApplicationUserDto, error) {
	environmentUrl, err := client.GetEnvironmentUrlById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	apiUrl := &url.URL{
		Scheme: "https",
		Host:   strings.TrimPrefix(environmentUrl, "https://"),
		Path:   "/api/data/v9.2/systemusers(" + systemApplicationUserId + ")/systemuserroles_association/$ref",
	}

	for _, roleId := range securityRolesIds {
		roleToassociate := map[string]interface{}{
			"@odata.id": fmt.Sprintf("%s/api/data/v9.2/roles(%s)", environmentUrl, roleId),
		}
		_, err = client.Api.Execute(ctx, "POST", apiUrl.String(), nil, roleToassociate, []int{http.StatusNoContent}, nil)
		if err != nil {
			return nil, err
		}
	}
	applicationuser, err := client.GetApplicationUserBySystemApplicationUserId(ctx, environmentId, systemApplicationUserId)
	if err != nil {
		return nil, err
	}
	return applicationuser, nil
}

func (client *ApplicationUserClient) GetEnvironmentUrlById(ctx context.Context, environmentId string) (string, error) {
	env, err := client.getEnvironment(ctx, environmentId)
	if err != nil {
		return "", err
	}
	environmentUrl := strings.TrimSuffix(env.Properties.LinkedEnvironmentMetadata.InstanceURL, "/")
	return environmentUrl, nil
}

func (client *ApplicationUserClient) getEnvironment(ctx context.Context, environmentId string) (*EnvironmentIdDto, error) {

	apiUrl := &url.URL{
		Scheme: "https",
		Host:   client.Api.GetConfig().Urls.BapiUrl,
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s", environmentId),
	}
	values := url.Values{}
	values.Add("$expand", "permissions,properties.capacity,properties/billingPolicy")
	values.Add("api-version", "2023-06-01")
	apiUrl.RawQuery = values.Encode()

	env := EnvironmentIdDto{}
	_, err := client.Api.Execute(ctx, "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &env)
	if err != nil {
		return nil, err
	}

	return &env, nil
}

func (client *ApplicationUserClient) GetSecurityRoles(ctx context.Context, environmentId, businessUnitId string) ([]SecurityRoleDto, error) {
	environmentUrl, err := client.GetEnvironmentUrlById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	apiUrl := &url.URL{
		Scheme: "https",
		Host:   strings.TrimPrefix(environmentUrl, "https://"),
		Path:   "/api/data/v9.2/roles",
	}
	if businessUnitId != "" {
		var values = url.Values{}
		values.Add("$filter", fmt.Sprintf("_businessunitid_value eq %s", businessUnitId))
		apiUrl.RawQuery = values.Encode()
	}
	securityRoleArray := SecurityRoleDtoArray{}
	_, err = client.Api.Execute(ctx, "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &securityRoleArray)
	if err != nil {
		return nil, err
	}
	return securityRoleArray.Value, nil
}

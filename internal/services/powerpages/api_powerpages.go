// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package powerpages

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

func newPowerPagesClient(apiClient *api.Client) client {
	return client{
		Api: apiClient,
		//environmentClient: environment.NewEnvironmentClient(apiClient),
	}
}

type client struct {
	Api *api.Client
	//environmentClient environment.Client
}

func (client *client) CreateWebsite(ctx context.Context, website *WebsiteCreateDto) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.Api.Config.Urls.PowerPlatformUrl,
		Path:   fmt.Sprintf("/powerpages/environments/%s/websites", website.DataverseOrganizationId),
	}
	values := url.Values{}
	values.Add("api-version", "2022-03-01-preview")
	apiUrl.RawQuery = values.Encode()

	resp, err := client.Api.Execute(ctx, nil, "POST", apiUrl.String(), nil, website, []int{http.StatusUnauthorized, http.StatusBadRequest, http.StatusAccepted, http.StatusNotFound}, nil)
	if err != nil {
		return err
	}

	if resp.HttpResponse.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %s", string(resp.BodyAsBytes))
	}
	return nil
}

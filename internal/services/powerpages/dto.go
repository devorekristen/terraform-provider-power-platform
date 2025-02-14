// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package powerpages

type WebsiteCreateDto struct {
	DataverseOrganizationId string `json:"dataverseOrganizationId"`
	Name                    string `json:"name"`
	SelectedBaseLanguage    int32  `json:"selectedBaseLanguage"`
	Subdomain               string `json:"subdomain"`
	TemplateName            string `json:"templateName"`
	WebsiteRecordId         string `json:"websiteRecordId"`
}

type WebsiteDto struct {
	ApplicationUserAadAppId        string   `json:"applicationUserAadAppId"`
	CreatedOn                      string   `json:"createdOn"`
	CustomHostNames                []string `json:"customHostNames"`
	DataverseInstanceUrl           string   `json:"dataverseInstanceUrl"`
	DataverseOrganizationId        string   `json:"dataverseOrganizationId"`
	EnvironmentId                  string   `json:"environmentId"`
	EnvironmentName                string   `json:"environmentName"`
	Id                             string   `json:"id"`
	IsCustomErrorEnabled           bool     `json:"isCustomErrorEnabled"`
	IsEarlyUpgradeEnabled          bool     `json:"isEarlyUpgradeEnabled"`
	Name                           string   `json:"name"`
	OwnerId                        string   `json:"ownerId"`
	PackageInstallStatus           string   `json:"packageInstallStatus"`
	PackageVersion                 string   `json:"packageVersion"`
	SelectedBaseLanguage           int      `json:"selectedBaseLanguage"`
	SiteVisibility                 string   `json:"siteVisibility"`
	Status                         string   `json:"status"`
	Subdomain                      string   `json:"subdomain"`
	SuspendedWebsiteDeletingInDays int      `json:"suspendedWebsiteDeletingInDays"`
	TemplateName                   string   `json:"templateName"`
	TenantId                       string   `json:"tenantId"`
	TrialExpiringInDays            int      `json:"trialExpiringInDays"`
	Type                           string   `json:"type"`
	WebsiteRecordId                string   `json:"websiteRecordId"`
	WebsiteUrl                     string   `json:"websiteUrl"`
}

package controller

import (
	"net/http"

	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

// credExample is an example value + description for a credential key.
type credExample struct{ example, description string }

// credentialExamples maps canonical CSP name -> credential key -> example. Keys
// are cb-spider's generic credential keys (what GetCloudOSMetaInfo returns).
// Example values are illustrative (from cb-tumblebug's template.credentials.yaml
// / cb-spider samples), NOT real credentials.
// NOTE: example values are deliberately obvious placeholders (NOT real, and not
// matching provider secret formats) so they are safe to commit and to display.
var credentialExamples = map[string]map[string]credExample{
	"AWS": {
		"ClientId":     {"<AWS_ACCESS_KEY_ID>", "AWS Access Key ID (aws_access_key_id)"},
		"ClientSecret": {"<AWS_SECRET_ACCESS_KEY>", "AWS Secret Access Key (aws_secret_access_key)"},
	},
	"AZURE": {
		"ClientId":       {"<AZURE_CLIENT_ID>", "Azure Client ID (clientId), a GUID"},
		"ClientSecret":   {"<AZURE_CLIENT_SECRET>", "Azure Client Secret (clientSecret)"},
		"TenantId":       {"<AZURE_TENANT_ID>", "Azure Tenant ID (tenantId), a GUID"},
		"SubscriptionId": {"<AZURE_SUBSCRIPTION_ID>", "Azure Subscription ID (subscriptionId), a GUID"},
	},
	"GCP": {
		"PrivateKey":  {"-----BEGIN PRIVATE KEY-----\\n<base64>\\n-----END PRIVATE KEY-----\\n", "GCP service account private key (private_key, inline with \\n)"},
		"ProjectID":   {"cloud-barista", "GCP Project ID (project_id)"},
		"ClientEmail": {"user01@cloud-barista.com", "GCP service account client email (client_email)"},
	},
	"ALIBABA": {
		"ClientId":     {"<ALIBABA_ACCESS_KEY_ID>", "Alibaba Access Key ID (AccessKeyId)"},
		"ClientSecret": {"<ALIBABA_ACCESS_KEY_SECRET>", "Alibaba Access Key Secret (AccessKeySecret)"},
	},
	"TENCENT": {
		"ClientId":     {"<TENCENT_SECRET_ID>", "Tencent SecretId"},
		"ClientSecret": {"<TENCENT_SECRET_KEY>", "Tencent SecretKey"},
	},
	"IBM": {
		"ApiKey": {"<IBM_CLOUD_API_KEY>", "IBM Cloud API key"},
	},
	"NCP": {
		"ClientId":     {"<NCLOUD_ACCESS_KEY>", "NCP Access Key (ncloud_access_key)"},
		"ClientSecret": {"<NCLOUD_SECRET_KEY>", "NCP Secret Key (ncloud_secret_key)"},
	},
	"OPENSTACK": {
		"IdentityEndpoint": {"http://openstack-host:5000/v3", "Keystone identity endpoint"},
		"Username":         {"demo", "OpenStack username"},
		"Password":         {"<OPENSTACK_PASSWORD>", "OpenStack password"},
		"DomainName":       {"Default", "OpenStack domain name"},
		"ProjectID":        {"<OPENSTACK_PROJECT_ID>", "OpenStack project ID"},
	},
}

// buildCredentialFields pairs each required credential key with its example.
func buildCredentialFields(canonical string, keys []string) []model.CSPCredentialField {
	exMap := credentialExamples[canonical]
	fields := make([]model.CSPCredentialField, 0, len(keys))
	for _, k := range keys {
		f := model.CSPCredentialField{Key: k}
		if ex, ok := exMap[k]; ok {
			f.Example = ex.example
			f.Description = ex.description
		}
		fields = append(fields, f)
	}
	return fields
}

// ListCSP godoc
//
//	@ID				list-csp
//	@Summary		List supported CSPs
//	@Description	Return the list of CSPs supported by the connected cb-spider.
//	@Tags			[CSP] Metadata
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	model.ListCSPRes		"List of CSP names"
//	@Failure		500	{object}	common.ErrorResponse	"Failed to query cb-spider"
//	@Router			/csp [get]
func ListCSP(c echo.Context) error {
	list, err := spider.ListCloudOS()
	if err != nil {
		return common.ReturnInternalError(c, err, "failed to list CSPs from cb-spider")
	}
	return c.JSONPretty(http.StatusOK, model.ListCSPRes{CSP: list}, " ")
}

// GetCSP godoc
//
//	@ID				get-csp
//	@Summary		Get CSP metadata
//	@Description	Return the credential keys, regions, and other metadata for the given CSP. Name is matched case-insensitively.
//	@Tags			[CSP] Metadata
//	@Accept			json
//	@Produce		json
//	@Param			name path string true "CSP name (case-insensitive, e.g. aws or AWS)"
//	@Success		200	{object}	model.CSPInfo			"CSP metadata"
//	@Failure		400	{object}	common.ErrorResponse	"Unsupported or missing CSP name"
//	@Failure		500	{object}	common.ErrorResponse	"Failed to query cb-spider"
//	@Router			/csp/{name} [get]
func GetCSP(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return common.ReturnErrorMsg(c, "Please provide the CSP name.")
	}

	canonical, err := spider.NormalizeProvider(name)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	meta, err := spider.GetCloudOSMetaInfo(canonical)
	if err != nil {
		return common.ReturnInternalError(c, err, "failed to get CSP metainfo from cb-spider")
	}

	defaultRegion := ""
	if len(meta.DefaultRegionToQuery) > 0 {
		defaultRegion = meta.DefaultRegionToQuery[0]
	}

	return c.JSONPretty(http.StatusOK, model.CSPInfo{
		Name:           canonical,
		CredentialKeys: meta.Credential,
		Credentials:    buildCredentialFields(canonical, meta.Credential),
		Regions:        meta.Region,
		DefaultRegion:  defaultRegion,
	}, " ")
}

// ListSourceGroupRegions godoc
//
//	@ID				list-source-group-regions
//	@Summary		List CSP regions for a source group
//	@Description	Return the CSP's available regions and zones, queried live via the source group's stored credential. CSP-type source groups only.
//	@Tags			[CSP] Metadata
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup (csp type)"
//	@Success		200	{object}	model.ListRegionRes		"Regions and zones"
//	@Failure		400	{object}	common.ErrorResponse	"Not a csp-type source group / missing sgId"
//	@Failure		500	{object}	common.ErrorResponse	"Failed to query cb-spider"
//	@Router			/source_group/{sgId}/region [get]
func ListSourceGroupRegions(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if sourceGroup.Type != serverCommon.SourceGroupTypeCSP {
		return common.ReturnErrorMsg(c, "regions are only available for csp-type source groups")
	}

	regions := make([]model.CSPRegion, 0)
	err = withSpiderCredential(sourceGroup, func(driverName, credName string) error {
		list, err := spider.ListRegionZonePreConfig(driverName, credName)
		if err != nil {
			return err
		}
		for _, r := range list {
			zones := make([]string, 0, len(r.ZoneList))
			for _, z := range r.ZoneList {
				zones = append(zones, z.Name)
			}
			regions = append(regions, model.CSPRegion{
				Name:        r.Name,
				DisplayName: r.DisplayName,
				Zones:       zones,
			})
		}
		return nil
	})
	if err != nil {
		return common.ReturnInternalError(c, err, "failed to list regions from cb-spider")
	}

	return c.JSONPretty(http.StatusOK, model.ListRegionRes{
		Provider: sourceGroup.ProviderName,
		Regions:  regions,
	}, " ")
}

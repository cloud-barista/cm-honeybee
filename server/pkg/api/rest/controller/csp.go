package controller

import (
	"errors"
	"net/http"
	"strings"

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
		"aws_access_key_id":     {"<AWS_ACCESS_KEY_ID>", "AWS Access Key ID"},
		"aws_secret_access_key": {"<AWS_SECRET_ACCESS_KEY>", "AWS Secret Access Key"},
	},
	"AZURE": {
		"clientId":       {"<AZURE_CLIENT_ID>", "Azure Client ID, a GUID"},
		"clientSecret":   {"<AZURE_CLIENT_SECRET>", "Azure Client Secret"},
		"tenantId":       {"<AZURE_TENANT_ID>", "Azure Tenant ID, a GUID"},
		"subscriptionId": {"<AZURE_SUBSCRIPTION_ID>", "Azure Subscription ID, a GUID"},
	},
	"GCP": {
		"private_key":  {"-----BEGIN PRIVATE KEY-----\\n<base64>\\n-----END PRIVATE KEY-----\\n", "GCP service account private key (inline with \\n)"},
		"project_id":   {"cloud-barista", "GCP Project ID"},
		"client_email": {"user01@cloud-barista.com", "GCP service account client email"},
	},
	"ALIBABA": {
		"AccessKeyId":     {"<ALIBABA_ACCESS_KEY_ID>", "Alibaba Cloud Access Key ID"},
		"AccessKeySecret": {"<ALIBABA_ACCESS_KEY_SECRET>", "Alibaba Cloud Access Key Secret"},
	},
	"TENCENT": {
		"SecretId":  {"<TENCENT_SECRET_ID>", "Tencent Cloud SecretId"},
		"SecretKey": {"<TENCENT_SECRET_KEY>", "Tencent Cloud SecretKey"},
	},
	"IBM": {
		"ApiKey": {"<IBM_CLOUD_API_KEY>", "IBM Cloud API key"},
	},
	"NCP": {
		"ncloud_access_key": {"<NCLOUD_ACCESS_KEY>", "NCP Access Key"},
		"ncloud_secret_key": {"<NCLOUD_SECRET_KEY>", "NCP Secret Key"},
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

// The cb-spider calls of this file, kept in variables so tests can stand in
// for them.
var (
	listCloudOS      = spider.ListCloudOS
	driverCapability = spider.GetDriverCapability
	listRegionZone   = spider.ListRegionZone
)

// driverSupports reports whether the driver capability covers a csp
// resource_type that is collected through a driver handler. Discovery and the
// CSP metadata both ask it, so the two cannot disagree. object_storage is not
// a driver handler (see objectStorageSupported).
func driverSupports(capability *spider.DriverCapability, resourceType string) bool {
	if capability == nil {
		return false
	}
	switch resourceType {
	case serverCommon.ResourceTypeVM:
		return capability.VMHandler
	case serverCommon.ResourceTypeK8s:
		return capability.ClusterHandler
	case serverCommon.ResourceTypeNLB:
		return capability.NLBHandler
	case serverCommon.ResourceTypeRDBMS:
		return capability.RDBMSHandler
	}
	return false
}

// objectStorageSupported reports whether buckets of the provider can be
// collected. Buckets are read through the S3 API by lib/spider (s3ConnRule),
// not through a driver handler, so the driver capability does not tell; the
// providers it handles are the ones in providers but onprem.
// TestObjectStorageSupported_MatchesS3Rules holds the two together.
func objectStorageSupported(provider string) bool {
	p := strings.ToLower(strings.TrimSpace(provider))
	return p != providerOnPrem && isSupportedProvider(p)
}

// supportedResourceTypes gives the csp resource_type values a connection of
// the provider can be collected as, in the order of cspResourceTypes. A
// provider honeybee has no driver for supports none.
func supportedResourceTypes(provider string) []string {
	out := []string{}
	capability, err := driverCapability(provider)
	if err != nil {
		return out
	}
	for _, rt := range cspResourceTypes {
		if rt == serverCommon.ResourceTypeObjectStorage {
			if objectStorageSupported(provider) {
				out = append(out, rt)
			}
			continue
		}
		if driverSupports(capability, rt) {
			out = append(out, rt)
		}
	}
	return out
}

// ListCSP godoc
//
//	@ID				list-csp
//	@Summary		List supported CSPs
//	@Description	Return the list of CSPs supported by the linked cb-spider drivers, and for each the resource_type values its connections can be collected as.
//	@Tags			[CSP] Metadata
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	model.ListCSPRes		"List of CSP names"
//	@Failure		500	{object}	common.ErrorResponse	"Failed to read CSP metadata"
//	@Router			/csp [get]
func ListCSP(c echo.Context) error {
	list, err := listCloudOS()
	if err != nil {
		return common.ReturnInternalError(c, err, "failed to list CSPs from the driver metadata")
	}
	// Output provider names in lowercase (e.g. "aws", "azure"); they are matched
	// case-insensitively on input.
	supported := make(map[string][]string, len(list))
	for i := range list {
		supported[strings.ToLower(list[i])] = supportedResourceTypes(list[i])
		list[i] = strings.ToLower(list[i])
	}
	return c.JSONPretty(http.StatusOK, model.ListCSPRes{CSP: list, SupportedResourceTypes: supported}, " ")
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
//	@Failure		500	{object}	common.ErrorResponse	"Failed to read CSP metadata"
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
		return common.ReturnInternalError(c, err, "failed to get CSP metainfo from the driver metadata")
	}

	defaultRegion := ""
	if len(meta.DefaultRegionToQuery) > 0 {
		defaultRegion = meta.DefaultRegionToQuery[0]
	}

	// Advertise the cb-spider "credentialcsp" keys (== cb-tumblebug's
	// template.credentials.yaml). Fall back to the generic keys if a driver has
	// no csp list.
	credKeys := meta.CredentialCSP
	if len(credKeys) == 0 {
		credKeys = meta.Credential
	}

	return c.JSONPretty(http.StatusOK, model.CSPInfo{
		Name:           strings.ToLower(canonical),
		CredentialKeys: credKeys,
		Credentials:    buildCredentialFields(canonical, credKeys),
		RegionKeys:     meta.Region,
		DefaultRegion:  defaultRegion,

		SupportedResourceTypes: supportedResourceTypes(canonical),
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
//	@Failure		500	{object}	common.ErrorResponse	"Failed to load the CSP credential or query the CSP"
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

	provider, credential, err := cspCredential(sourceGroup)
	if err != nil {
		return common.ReturnInternalError(c, err, "failed to load the source group's CSP credential")
	}

	return returnRegions(c, provider, credential)
}

// returnRegions queries the CSP's regions with the credential and writes them
// as the response. A driver error may quote what it was given, so the
// credential values are masked in it before it is logged or returned.
func returnRegions(c echo.Context, provider string, credential []spider.KeyValue) error {
	list, err := listRegionZone(provider, credential)
	if err != nil {
		return common.ReturnInternalError(c, maskCredential(err, credential), "failed to list regions from the CSP")
	}

	regions := make([]model.CSPRegion, 0, len(list))
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

	return c.JSONPretty(http.StatusOK, model.ListRegionRes{
		Provider: strings.ToLower(provider),
		Regions:  regions,
	}, " ")
}

// maskCredential replaces every credential value quoted in err.
func maskCredential(err error, credential []spider.KeyValue) error {
	msg := err.Error()
	for _, kv := range credential {
		if kv.Value != "" {
			msg = strings.ReplaceAll(msg, kv.Value, "****")
		}
	}
	if msg == err.Error() {
		return err
	}
	return errors.New(msg)
}

// ListCSPRegions godoc
//
//	@ID				list-csp-regions
//	@Summary		List CSP regions with a credential
//	@Description	Return the CSP's available regions and zones, queried live with the credential in the body, so the region of a csp source group can be chosen before the group exists. The credential takes the keys GET /csp/{name} lists and is neither stored nor logged. The region queried is the CSP's default one for listing.
//	@Tags			[CSP] Metadata
//	@Accept			json
//	@Produce		json
//	@Param			name path string true "CSP name (case-insensitive, e.g. aws or AWS)"
//	@Param			Credential body model.ListCSPRegionReq true "CSP credential"
//	@Success		200	{object}	model.ListRegionRes		"Regions and zones"
//	@Failure		400	{object}	common.ErrorResponse	"Unsupported CSP name, or a credential key missing or not accepted"
//	@Failure		500	{object}	common.ErrorResponse	"Failed to read CSP metadata or query the CSP"
//	@Router			/csp/{name}/region [post]
func ListCSPRegions(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return common.ReturnErrorMsg(c, "Please provide the CSP name.")
	}

	req := new(model.ListCSPRegionReq)
	if err := c.Bind(req); err != nil {
		return common.ReturnErrorMsg(c, "request body is not valid JSON")
	}
	if len(req.Credential) == 0 {
		return common.ReturnErrorMsg(c, "credential is empty")
	}

	provider, err := spider.NormalizeProvider(name)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	meta, err := spider.GetCloudOSMetaInfo(provider)
	if err != nil {
		return common.ReturnInternalError(c, err, "failed to get CSP metainfo from the driver metadata")
	}
	// The same key check as registering a csp source group, so a credential
	// that lists regions here is one the group registration accepts.
	credential, err := canonicalizeCredentialKV(provider, meta, req.Credential)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return returnRegions(c, provider, toSpiderKV(credential))
}

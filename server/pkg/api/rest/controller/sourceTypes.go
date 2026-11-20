package controller

import (
	"net/http"
	"strings"

	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

// cspResourceTypes are the resource_type values a csp source group accepts,
// in the order they are listed. common.IsValidCSPResourceType must accept
// exactly these; TestSourceGroupTypes_ResourceTypes holds the two together.
var cspResourceTypes = []string{
	serverCommon.ResourceTypeVM,
	serverCommon.ResourceTypeK8s,
	serverCommon.ResourceTypeObjectStorage,
	serverCommon.ResourceTypeNLB,
	serverCommon.ResourceTypeRDBMS,
}

var cspResourceTypeDescriptions = map[string]string{
	serverCommon.ResourceTypeVM:            "Virtual machine. With ip_address and SSH credentials the agent collects it as well.",
	serverCommon.ResourceTypeK8s:           "Managed Kubernetes cluster.",
	serverCommon.ResourceTypeObjectStorage: "Object storage bucket.",
	serverCommon.ResourceTypeNLB:           "Network load balancer.",
	serverCommon.ResourceTypeRDBMS:         "Managed database instance.",
}

// sshFields are the SSH fields of a connection. requiredWhen is empty when
// they are always required, and keyOnly drops password for the connections
// cm-centipede reaches by private key only (see checkSSHKeyOnly).
func sshFields(requiredWhen string, keyOnly bool) []model.SourceFieldSpec {
	required := requiredWhen == ""
	port := model.SourceFieldSpec{Name: "ssh_port", Required: true, Description: "SSH port of the host."}
	if !required {
		// Registration fills 22 in wherever the SSH fields are conditional.
		port = model.SourceFieldSpec{Name: "ssh_port", Default: "22", Description: "SSH port of the host."}
	}

	fields := []model.SourceFieldSpec{
		{Name: "ip_address", Required: required, RequiredWhen: requiredWhen, Description: "IP address of the host."},
		port,
		{Name: "user", Required: required, RequiredWhen: requiredWhen, Description: "SSH user."},
	}
	if keyOnly {
		return append(fields, model.SourceFieldSpec{Name: "private_key", Required: required, RequiredWhen: requiredWhen,
			Secret: true, Description: "SSH private key. A password is refused."})
	}

	return append(fields,
		model.SourceFieldSpec{Name: "password", RequiredWhen: requiredWhen, Secret: true, OneOf: "ssh_auth",
			Description: "SSH password. password or private_key must be given."},
		model.SourceFieldSpec{Name: "private_key", RequiredWhen: requiredWhen, Secret: true, OneOf: "ssh_auth",
			Description: "SSH private key. password or private_key must be given."},
	)
}

func strPtr(s string) *string { return &s }

// providersRequiringRegion are the providers of a minio group that require
// region_name (see regionRequired).
func providersRequiringRegion() []string {
	out := []string{}
	for _, p := range providers {
		if regionRequired(p) {
			out = append(out, p)
		}
	}
	return out
}

func accessTypeField(name string) model.SourceFieldSpec {
	return model.SourceFieldSpec{Name: name, Enum: []string{model.AccessTypeDirect, model.AccessTypeSSHTunnel},
		Default:     model.AccessTypeDirect,
		Description: "direct reaches the source from the honeybee server, ssh-tunnel through the agent on the SSH host."}
}

// sourceGroupTypeSpecs describes what each source group type takes, as
// CreateSourceGroup and checkCreateConnectionInfoReq check it.
func sourceGroupTypeSpecs() []model.SourceGroupTypeSpec {
	cspTypes := make([]model.ConnectionTypeSpec, 0, len(cspResourceTypes))
	for _, rt := range cspResourceTypes {
		auth := []model.SourceFieldSpec{}
		if rt == serverCommon.ResourceTypeVM {
			auth = sshFields("ip_address", false)
			auth[0].RequiredWhen = ""
			auth[0].Description = "IP address the agent is reached on. Without it only the CSP-side information is collected."
		}
		cspTypes = append(cspTypes, model.ConnectionTypeSpec{
			Type:         rt,
			ResourceType: strPtr(rt),
			Description:  cspResourceTypeDescriptions[rt],
			AuthFields:   auth,
			Fields: []model.SourceFieldSpec{
				{Name: "resource_id", Required: true, Description: "ID of the resource at the CSP."},
				{Name: "zone", Description: "Zone override for this connection. Empty takes the zone of the group's region_name, if any."},
			},
		})
	}

	return []model.SourceGroupTypeSpec{
		{
			Type:                serverCommon.SourceGroupTypeOnprem,
			Aliases:             []string{serverCommon.SourceGroupTypeSSH},
			Description:         "On-premise hosts collected through the agent over SSH, and Kubernetes clusters registered by kubeconfig.",
			GroupFields:         []model.SourceFieldSpec{},
			ResourceTypeField:   model.ResourceTypeFieldOptional,
			DefaultResourceType: serverCommon.ResourceTypeVM,
			ConnectionTypes: []model.ConnectionTypeSpec{
				{
					Type:         serverCommon.ResourceTypeVM,
					ResourceType: strPtr(serverCommon.ResourceTypeVM),
					Description:  "Host reached over SSH.",
					AuthFields:   sshFields("", false),
					Fields:       []model.SourceFieldSpec{},
				},
				{
					Type:         serverCommon.ResourceTypeK8s,
					ResourceType: strPtr(serverCommon.ResourceTypeK8s),
					Description:  "Kubernetes cluster reached by its kubeconfig.",
					AuthFields: []model.SourceFieldSpec{
						{Name: "kubeconfig", Required: true, Secret: true, Description: "kubeconfig YAML with at least one cluster."},
					},
					Fields: []model.SourceFieldSpec{},
				},
			},
		},
		{
			Type:        serverCommon.SourceGroupTypeCSP,
			Description: "Cloud resources collected through the CSP driver with the group's credential.",
			GroupFields: []model.SourceFieldSpec{
				{Name: "provider_name", Required: true, ValuesSource: "/csp", Description: "CSP name."},
				{Name: "region_name", Required: true, ValuesSource: "/csp/{name}/region",
					Description: "Region. '<region>/<zone>' sets a zone as well."},
				{Name: "credential", Required: true, Secret: true, CredentialSource: "/csp/{name}",
					Description: "Key-value list of the CSP credential, with the keys GET /csp/{name} lists."},
			},
			ResourceTypeField: model.ResourceTypeFieldRequired,
			ConnectionTypes:   cspTypes,
		},
		{
			Type:              serverCommon.SourceGroupTypeFS,
			Description:       "Hosts whose files under a path are collected through the agent, for data migration.",
			GroupFields:       []model.SourceFieldSpec{},
			ResourceTypeField: model.ResourceTypeFieldNone,
			ConnectionTypes: []model.ConnectionTypeSpec{
				{
					Type:        serverCommon.SourceGroupTypeFS,
					Description: "Host reached over SSH by private key.",
					AuthFields:  sshFields("", true),
					Fields: []model.SourceFieldSpec{
						{Name: "fs_scan_path", Default: "/home", Description: "Absolute path to collect. \"/\" collects the whole host."},
					},
				},
			},
		},
		{
			Type:        serverCommon.SourceGroupTypeDB,
			Description: "DBMS servers reached by host and port, self-managed or CSP managed.",
			GroupFields: []model.SourceFieldSpec{
				{Name: "provider_name", Required: true, Enum: providers, Description: "Where the databases are hosted."},
			},
			ResourceTypeField: model.ResourceTypeFieldNone,
			ConnectionTypes: []model.ConnectionTypeSpec{
				{
					Type:        serverCommon.SourceGroupTypeDB,
					Description: "DBMS server.",
					AuthFields: append([]model.SourceFieldSpec{
						{Name: "db_type", Required: true, Enum: []string{"mysql", "mariadb", "postgresql", "mongodb"}, Description: "DBMS type."},
						{Name: "db_host", Required: true, Description: "DBMS host."},
						{Name: "db_port", Required: true, Description: "DBMS port."},
						{Name: "db_username", Required: true, Description: "DBMS user."},
						{Name: "db_password", Required: true, Secret: true, Description: "DBMS password."},
						accessTypeField("db_access_type"),
						{Name: "db_tls_mode", Enum: []string{"disable", "prefer", "require", "verify-ca", "verify-full"}, Default: "disable",
							Description: "TLS mode."},
						{Name: "db_tls_ca_pem", Description: "CA certificate (PEM) verify-ca and verify-full check the server against."},
						{Name: "db_auth_source", Description: "MongoDB authentication database. Empty is admin."},
						{Name: "db_connect_timeout", Description: "Connect timeout in seconds. Empty is the driver default."},
					}, sshFields("db_access_type="+model.AccessTypeSSHTunnel, true)...),
					Fields: []model.SourceFieldSpec{
						{Name: "db_name", Description: "Database to collect. Empty collects every user database."},
						{Name: "db_pg_schema", Description: "PostgreSQL schemas within db_name. Empty collects every user schema."},
					},
				},
			},
		},
		{
			Type:        serverCommon.SourceGroupTypeMinIO,
			Description: "S3-compatible object storage, of a CSP or self-hosted.",
			GroupFields: []model.SourceFieldSpec{
				{Name: "provider_name", Required: true, Enum: providers, Description: "Picks the S3 endpoint form."},
				{Name: "region_name", RequiredWhen: "provider_name in " + strings.Join(providersRequiringRegion(), ", "),
					Description: "Region of the endpoint."},
			},
			ResourceTypeField: model.ResourceTypeFieldNone,
			ConnectionTypes: []model.ConnectionTypeSpec{
				{
					Type:        serverCommon.SourceGroupTypeMinIO,
					Description: "S3 endpoint and the keys to it.",
					AuthFields: append([]model.SourceFieldSpec{
						{Name: "os_access_key_id", Required: true, Secret: true, Description: "Access key ID (azure: storage account name)."},
						{Name: "os_secret_access_key", Required: true, Secret: true, Description: "Secret access key."},
						{Name: "os_endpoint", RequiredWhen: "provider_name in openstack, " + providerOnPrem,
							Description: "S3 endpoint, host[:port] or http(s) URL. Overrides the derived endpoint."},
						{Name: "os_use_ssl", Description: "HTTPS for openstack and onprem endpoints. A scheme on os_endpoint takes precedence."},
						accessTypeField("os_access_type"),
					}, sshFields("os_access_type="+model.AccessTypeSSHTunnel, true)...),
					Fields: []model.SourceFieldSpec{
						{Name: "os_scan_bucket", Required: true, Description: "Bucket to collect. On tencent it carries the AppId suffix."},
						{Name: "os_scan_prefix", Description: "Collect only the keys under this prefix."},
					},
				},
			},
		},
	}
}

// ListSourceGroupTypes godoc
//
//	@ID				list-source-group-types
//	@Summary		List source group types
//	@Description	Return each source group type with the connection types it holds and the fields each takes, so a client can build the registration forms. The credential keys of a csp group come from GET /csp/{name}.
//	@Tags			[On-premise] SourceGroup
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	model.ListSourceGroupTypesRes	"Source group types"
//	@Router			/source_group/types [get]
func ListSourceGroupTypes(c echo.Context) error {
	return c.JSONPretty(http.StatusOK, model.ListSourceGroupTypesRes{Types: sourceGroupTypeSpecs()}, " ")
}

// connectionTypeKey names the connection type of a stored resource_type in
// the group, as connection_count_by_type counts it and the types API lists it.
func connectionTypeKey(sourceGroup *model.SourceGroup, resourceType string) string {
	switch {
	case serverCommon.IsOnpremType(sourceGroup.Type):
		if resourceType == "" {
			return serverCommon.ResourceTypeVM
		}
		return resourceType
	case serverCommon.IsCSPType(sourceGroup.Type):
		return resourceType
	}

	return sourceGroup.Type
}

// countConnectionsByType counts the connections of one group the way
// dao.ConnectionInfoCountByType does, for the callers that hold the rows
// already.
func countConnectionsByType(sourceGroupID string, conns []model.ConnectionInfo) []dao.ConnectionTypeCount {
	index := map[string]int{}
	counts := []dao.ConnectionTypeCount{}
	for i := range conns {
		ci := &conns[i]
		j, ok := index[ci.ResourceType]
		if !ok {
			j = len(counts)
			index[ci.ResourceType] = j
			counts = append(counts, dao.ConnectionTypeCount{SourceGroupID: sourceGroupID, ResourceType: ci.ResourceType})
		}
		counts[j].Total++
		if ci.ConnectionStatus == model.ConnectionInfoStatusSuccess {
			counts[j].ConnectionSuccess++
		}
		if ci.AgentStatus == model.ConnectionInfoStatusSuccess {
			counts[j].AgentSuccess++
		}
		if strings.TrimSpace(ci.IPAddress) != "" {
			counts[j].WithIPAddress++
		}
	}

	return counts
}

// connectionStatusCount sums the status counts of a group's connections.
func connectionStatusCount(counts []dao.ConnectionTypeCount) model.ConnectionInfoStatusCount {
	var s model.ConnectionInfoStatusCount
	for _, c := range counts {
		s.ConnectionInfoTotal += c.Total
		s.CountConnectionSuccess += c.ConnectionSuccess
		s.CountConnectionFailed += c.Total - c.ConnectionSuccess
		s.CountAgentSuccess += c.AgentSuccess
		s.CountAgentFailed += c.Total - c.AgentSuccess
	}
	return s
}

// connectionCountByType gives connection_count_by_type of a group.
func connectionCountByType(sourceGroup *model.SourceGroup, counts []dao.ConnectionTypeCount) map[string]int {
	out := map[string]int{}
	for _, c := range counts {
		if c.Total > 0 {
			out[connectionTypeKey(sourceGroup, c.ResourceType)] += c.Total
		}
	}
	return out
}

// sourceGroupCapabilities judges what a group can collect from its type and
// the counts of its connections. The judgement is made on a stand-in
// connection per counted kind with the functions the imports choose their
// targets by (infraImportTargets, agentImportTargets, isInstalledRDBMSHost),
// so the capabilities say what an import would act on. An ip_address is the
// only column besides resource_type those functions read.
//
// Infra, software and kubernetes are collected from onprem and csp groups
// only: an fs, db or minio group is a data source, and its own collection is
// the one it reports.
func sourceGroupCapabilities(sourceGroup *model.SourceGroup, counts []dao.ConnectionTypeCount) model.SourceGroupCapabilities {
	var caps model.SourceGroupCapabilities

	switch {
	case serverCommon.IsFSType(sourceGroup.Type):
		caps.IsCollectFs = totalConnections(counts) > 0
		return caps
	case serverCommon.IsDBType(sourceGroup.Type):
		caps.IsCollectDb = totalConnections(counts) > 0
		return caps
	case serverCommon.IsMinIOType(sourceGroup.Type):
		caps.IsCollectObjectStorage = totalConnections(counts) > 0
		return caps
	case !serverCommon.IsOnpremType(sourceGroup.Type) && !serverCommon.IsCSPType(sourceGroup.Type):
		return caps
	}

	for _, c := range counts {
		if c.Total == 0 {
			continue
		}
		probes := []model.ConnectionInfo{}
		if c.WithIPAddress > 0 {
			probes = append(probes, model.ConnectionInfo{ResourceType: c.ResourceType, IPAddress: "probe"})
		}
		if c.WithIPAddress < c.Total {
			probes = append(probes, model.ConnectionInfo{ResourceType: c.ResourceType})
		}

		if len(infraImportTargets(sourceGroup, probes)) > 0 {
			caps.IsCollectInfra = true
		}
		agent := agentImportTargets(sourceGroup, probes)
		if len(agent) > 0 {
			caps.IsCollectSoftware = true
			caps.IsCollectKubernetes = true
			if isInstalledRDBMSHost(&agent[0]) {
				caps.IsCollectDb = true
			}
		}

		if !serverCommon.IsCSPType(sourceGroup.Type) {
			continue
		}
		switch c.ResourceType {
		case serverCommon.ResourceTypeK8s:
			caps.IsCollectKubernetes = true
		case serverCommon.ResourceTypeObjectStorage:
			caps.IsCollectObjectStorage = true
		case serverCommon.ResourceTypeRDBMS:
			caps.IsCollectDb = true
		case serverCommon.ResourceTypeNLB:
			caps.IsCollectNlb = true
		}
	}

	return caps
}

func totalConnections(counts []dao.ConnectionTypeCount) int {
	n := 0
	for _, c := range counts {
		n += c.Total
	}
	return n
}

// sourceGroupRes builds the response of one group from its connection counts.
func sourceGroupRes(sourceGroup *model.SourceGroup, counts []dao.ConnectionTypeCount) model.SourceGroupRes {
	return model.SourceGroupRes{
		ID:                        sourceGroup.ID,
		Name:                      sourceGroup.Name,
		Description:               sourceGroup.Description,
		Type:                      sourceGroup.Type,
		ProviderName:              sourceGroup.ProviderName,
		RegionName:                sourceGroup.RegionName,
		ConnectionInfoStatusCount: connectionStatusCount(counts),
		Capabilities:              sourceGroupCapabilities(sourceGroup, counts),
		ConnectionCountByType:     connectionCountByType(sourceGroup, counts),
	}
}

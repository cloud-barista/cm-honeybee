package model

// CSPInfo describes a CSP supported by cb-spider together with the credential
// keys honeybee will require when registering a SourceGroup of that CSP.
type CSPInfo struct {
	Name           string   `json:"name" example:"AWS"`
	CredentialKeys []string `json:"credential_keys"`
	// Credentials lists each required credential key with an example value and a
	// short description, so a client can render a self-documenting input form.
	Credentials []CSPCredentialField `json:"credentials"`
	// RegionKeys are the keys needed to define a region for this CSP (e.g.
	// ["Region","Zone"]) - NOT the list of available regions. The actual region
	// list requires a credential; get it via GET /source_group/{sgId}/region,
	// or via POST /csp/{name}/region before the group exists.
	RegionKeys    []string `json:"region_keys"`
	DefaultRegion string   `json:"default_region,omitempty"`
	// SupportedResourceTypes are the resource_type values a connection of a
	// csp source group of this CSP can be collected as. Empty when honeybee has
	// no driver for the CSP.
	SupportedResourceTypes []string `json:"supported_resource_types"`
}

// CSPCredentialField describes one credential key of a CSP with an example.
type CSPCredentialField struct {
	Key         string `json:"key" example:"ClientId"`
	Example     string `json:"example,omitempty"`
	Description string `json:"description,omitempty"`
}

// ListCSPRes is the response payload for GET /csp.
type ListCSPRes struct {
	CSP []string `json:"csp"`
	// SupportedResourceTypes maps each name in CSP to the resource_type values
	// its connections can be collected as (see CSPInfo).
	SupportedResourceTypes map[string][]string `json:"supported_resource_types"`
}

// ListCSPRegionReq is the request body for POST /csp/{name}/region. The
// credential is used for this one query and is not stored.
type ListCSPRegionReq struct {
	// Credential takes the keys GET /csp/{name} lists in credential_keys.
	Credential []KeyValue `json:"credential"`
}

// CSPRegion is one region (with its zones) available for a CSP.
type CSPRegion struct {
	Name        string   `json:"name" example:"koreacentral"`
	DisplayName string   `json:"display_name,omitempty"`
	Zones       []string `json:"zones,omitempty"`
}

// ListRegionRes is the response payload for GET /source_group/{sgId}/region
// and POST /csp/{name}/region.
type ListRegionRes struct {
	Provider string      `json:"provider"`
	Regions  []CSPRegion `json:"regions"`
}

// DiscoveredResource is a single CSP resource returned by the discovery API.
type DiscoveredResource struct {
	ResourceType string            `json:"resource_type" example:"vm"`
	ResourceID   string            `json:"resource_id"   example:"i-0abc..."`
	Name         string            `json:"name,omitempty"`
	Region       string            `json:"region,omitempty"`
	Extra        map[string]string `json:"extra,omitempty"`
}

// DiscoverRes is the response payload for the discovery endpoint.
type DiscoverRes struct {
	Items []DiscoveredResource `json:"items"`
	// Unsupported marks a request the CSP's driver cannot serve at all - e.g.
	// Oracle implements no NLBHandler. The call succeeded; the CSP simply cannot
	// answer it. Without this an empty Items list reads as "this account has no
	// such resource", which is a different and wrong conclusion.
	Unsupported bool   `json:"unsupported,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

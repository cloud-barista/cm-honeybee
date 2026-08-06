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
	// ["Region","Zone"]) — NOT the list of available regions. The actual region
	// list requires a credential; get it via GET /source_group/{sgId}/region.
	RegionKeys    []string `json:"region_keys"`
	DefaultRegion string   `json:"default_region,omitempty"`
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
}

// CSPRegion is one region (with its zones) available for a CSP.
type CSPRegion struct {
	Name        string   `json:"name" example:"koreacentral"`
	DisplayName string   `json:"display_name,omitempty"`
	Zones       []string `json:"zones,omitempty"`
}

// ListRegionRes is the response payload for GET /source_group/{sgId}/region.
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
}

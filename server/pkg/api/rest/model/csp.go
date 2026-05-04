package model

// CSPInfo describes a CSP supported by cb-spider together with the credential
// keys honeybee will require when registering a SourceGroup of that CSP.
type CSPInfo struct {
	Name           string   `json:"name" example:"AWS"`
	CredentialKeys []string `json:"credential_keys"`
	Regions        []string `json:"regions"`
	DefaultRegion  string   `json:"default_region,omitempty"`
}

// ListCSPRes is the response payload for GET /csp.
type ListCSPRes struct {
	CSP []string `json:"csp"`
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

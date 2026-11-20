package model

// The kinds of collection a group import by resource kind reports. A CSP
// resource other than a VM is collected from the CSP under the kind of the
// resource; a host is collected as infra, and through the agent as software,
// kubernetes and helm.
const (
	ImportKindInfra      = "infra"
	ImportKindSoftware   = "software"
	ImportKindKubernetes = "kubernetes"
	ImportKindHelm       = "helm"
	ImportKindBucket     = "bucket"
	ImportKindRDBMS      = "rdbms"
	ImportKindNLB        = "nlb"
)

// ImportKindResult is the outcome of one collection of a connection.
type ImportKindResult struct {
	// Kind is infra, software, kubernetes, helm, bucket, rdbms or nlb.
	Kind string `json:"kind" example:"infra"`
	// Status is success or failed.
	Status string `json:"status" example:"success"`
	// ErrorMessage says why the collection failed. It is left out on success.
	ErrorMessage string `json:"error_message,omitempty"`
}

// ImportConnectionResult lists the collections a group import ran for one
// connection, in the order they ran.
type ImportConnectionResult struct {
	ConnectionID string             `json:"connection_id"`
	ResourceType string             `json:"resource_type" example:"vm"`
	Results      []ImportKindResult `json:"results"`
}

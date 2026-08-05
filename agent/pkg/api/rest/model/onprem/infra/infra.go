package infra

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/network"
)

type Infra struct {
	Compute Compute         `json:"compute"`
	Network network.Network `json:"network"`
	GPU     GPU             `json:"gpu"`
	Storage Storage         `json:"storage"`
	HAProxy HAProxy         `json:"haproxy"`
	MinIO   MinIO           `json:"minio"`
	// CSP holds provider-side VM information for CSP-type sources (collected via
	// cb-spider). It is nil for on-premise/SSH sources and omitted from output
	// when absent.
	CSP *CSPInfo `json:"csp,omitempty"`
}

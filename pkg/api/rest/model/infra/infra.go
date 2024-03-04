package infra

import (
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/network"
)

type Infra struct {
	Compute Compute         `json:"compute"`
	Network network.Network `json:"network"`
	GPU     GPU             `json:"gpu"`
	Storage Storage         `json:"storage"`
}

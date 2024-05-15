package infra

import (
	"github.com/cloud-barista/cm-honeybee/agent/gpu/drm"
	"github.com/cloud-barista/cm-honeybee/agent/gpu/nvidia"
)

type GPU struct {
	NVIDIA []nvidia.NVIDIA `json:"nvidia"`
	DRM    []drm.DRM       `json:"drm"`
}

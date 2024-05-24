// Getting DRM information is only available on Linux & Unix like systems

//go:build windows

package drm

import "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"

func GetDRMInfo() ([]infra.DRM, error) {
	return []infra.DRM{}, nil
}

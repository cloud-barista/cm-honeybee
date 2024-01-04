// Getting libvirt network information is not implemented for Windows

//go:build windows

package network

import (
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/network"
	"github.com/jollaman999/utils/logger"
)

func GetLibvirtNetInfo() (network.LibvirtNet, error) {
	var libvirtNetwork network.LibvirtNet

	logger.Println(logger.INFO, false,
		"LIBVIRT_NET: Getting libvirt network information is not implemented for Windows.")

	return libvirtNetwork, nil
}

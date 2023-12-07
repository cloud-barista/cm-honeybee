// Getting ovs bridges is not implemented for Windows

//go:build windows

package network

import (
	"github.com/cloud-barista/cm-honeybee/model/network"
	"github.com/jollaman999/utils/logger"
)

func GetOVSInfo() ([]network.OVSBridge, error) {
	var ovsBridges []network.OVSBridge

	logger.Println(logger.INFO, false,
		"OVS: Getting OVS information is not implemented for Windows.")

	return ovsBridges, nil
}

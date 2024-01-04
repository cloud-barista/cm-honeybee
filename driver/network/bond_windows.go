// Getting bonding interfaces for Windows

//go:build windows

package network

import (
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/network"
	"github.com/jollaman999/utils/logger"
)

func GetBondingInfo() ([]network.Bonding, error) {
	var bonds []network.Bonding

	logger.Println(logger.INFO, false,
		"BONDING: Getting bonding information is not implemented for Windows.")

	return bonds, nil
}

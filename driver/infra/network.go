package infra

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/driver/network"
	modelNet "github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/network"
	"github.com/jollaman999/utils/logger"
)

func GetNetworkInfo() (modelNet.Network, error) {
	var n modelNet.Network
	var err error

	n.Host.NetworkInterface, err = network.GetNICs()
	if err != nil {
		errMsg := "NIC: " + err.Error()
		logger.Println(logger.DEBUG, true, errMsg)

		return n, errors.New(errMsg)
	}

	n.Host.Route, err = network.GetRoutes()
	if err != nil {
		errMsg := "ROUTES: " + err.Error()
		logger.Println(logger.DEBUG, true, errMsg)

		return n, errors.New(errMsg)
	}

	n.Host.FirewallRule, err = network.GetFirewallRules()
	if err != nil {
		errMsg := "FIREWALL RULE: " + err.Error()
		logger.Println(logger.DEBUG, true, errMsg)

		return n, errors.New(errMsg)
	}

	return n, nil
}

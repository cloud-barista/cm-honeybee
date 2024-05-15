// Getting network interfaces for Windows

//go:build windows

package network

import (
	"github.com/cloud-barista/cm-honeybee/agent/lib/routes"
	network2 "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/network"
	"github.com/shirou/gopsutil/v3/net"
	"strings"
)

func GetNICs() ([]network2.NIC, error) {
	var networkInterfaces []network2.NIC

	interfaces, err := net.Interfaces()
	if err != nil {
		return networkInterfaces, err
	}

	var defaultRoutes []routes.RouteStruct

	defaultRoutes, err = routes.GetWindowsRoutes(true)
	if err != nil {
		return networkInterfaces, err
	}

	for _, i := range interfaces {
		var addresses []string
		var addressesWithoutPrefix []string
		var gateways []string

		for _, a := range i.Addrs {
			addresses = append(addresses, a.Addr)
			addrSplit := strings.Split(a.Addr, "/")
			if len(addrSplit) == 2 {
				addressesWithoutPrefix = append(addressesWithoutPrefix, addrSplit[0])
			}
		}

		for _, route := range defaultRoutes {
			for _, a := range addressesWithoutPrefix {
				if route.Interface == a {
					gateways = append(gateways, route.NextHop)
				}
			}
		}

		networkInterfaces = append(networkInterfaces, network2.NIC{
			Interface:  i.Name,
			Address:    addresses,
			Gateway:    gateways,
			MACAddress: i.HardwareAddr,
			MTU:        i.MTU,
		})
	}

	return networkInterfaces, nil
}

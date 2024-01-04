// Getting network interfaces for Windows

//go:build windows

package network

import (
	"github.com/cloud-barista/cm-honeybee/lib/routes"
	network2 "github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/network"
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
	var allRoutes []routes.RouteStruct

	defaultRoutes, err = routes.GetWindowsRoutes(true)
	if err != nil {
		return networkInterfaces, err
	}

	allRoutes, err = routes.GetWindowsRoutes(false)
	if err != nil {
		return networkInterfaces, err
	}

	for _, i := range interfaces {
		var addresses []string
		var addressesWithoutPrefix []string
		var gateways []string
		var ros []network2.Route

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

		for _, route := range allRoutes {
			for _, a := range addressesWithoutPrefix {
				if route.Interface == a {
					ros = append(ros, network2.Route{
						Destination: route.Destination,
						Netmask:     route.Netmask,
						NextHop:     route.NextHop,
					})
				}
			}
		}

		networkInterfaces = append(networkInterfaces, network2.NIC{
			Interface: i.Name,
			Address:   addresses,
			Gateway:   gateways,
			Route:     ros,
			MAC:       i.HardwareAddr,
			MTU:       i.MTU,
		})
	}

	return networkInterfaces, nil
}

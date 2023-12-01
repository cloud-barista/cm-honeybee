// Getting network information for Windows

//go:build windows

package infra

import (
	"fmt"
	"github.com/cloud-barista/cm-honeybee/lib/routes"
	"github.com/cloud-barista/cm-honeybee/model/infra"
	"github.com/shirou/gopsutil/v3/net"
	"strings"
)

func GetNetworkInfo() (infra.Network, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return infra.Network{}, err
	}

	var physicalNetworks []infra.PhysicalNetwork
	var defaultRoutes []routes.RouteStruct
	var allRoutes []routes.RouteStruct

	defaultRoutes, err = routes.GetWindowsRoutes(true)
	if err != nil {
		return infra.Network{}, err
	}

	allRoutes, err = routes.GetWindowsRoutes(false)
	if err != nil {
		return infra.Network{}, err
	}

	for _, i := range interfaces {
		var addresses []string
		var addressesWithoutPrefix []string
		var gateways []string
		var ros []infra.Route

		for _, a := range i.Addrs {
			addresses = append(addresses, a.Addr)
			addrSplit := strings.Split(a.Addr, "/")
			if len(addrSplit) == 2 {
				addressesWithoutPrefix = append(addressesWithoutPrefix, addrSplit[0])
			}
		}

		for _, route := range defaultRoutes {
			for _, a := range addressesWithoutPrefix {
				fmt.Println(route.Interface, a)
				if route.Interface == a {
					gateways = append(gateways, route.NextHop)
				}
			}
		}

		for _, route := range allRoutes {
			for _, a := range addressesWithoutPrefix {
				if route.Interface == a {
					ros = append(ros, infra.Route{
						Destination: route.Destination,
						Netmask:     route.Netmask,
						NextHop:     route.NextHop,
					})
				}
			}
		}

		physicalNetworks = append(physicalNetworks, infra.PhysicalNetwork{
			Interface: i.Name,
			Address:   addresses,
			Gateway:   gateways,
			Route:     ros,
			MAC:       i.HardwareAddr,
			MTU:       i.MTU,
		})
	}

	network := infra.Network{
		PhysicalNetwork: physicalNetworks,
	}

	return network, nil
}

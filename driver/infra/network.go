package infra

import (
	"github.com/cloud-barista/cm-honeybee/lib/routes"
	"github.com/cloud-barista/cm-honeybee/model/infra"
	"github.com/shirou/gopsutil/v3/net"
)

func GetNetworkInfo() (infra.Network, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return infra.Network{}, err
	}

	var physicalNetworks []infra.PhysicalNetwork
	var defaultRoutes []routes.RouteStruct
	var allRoutes []routes.RouteStruct

	defaultRoutes, err = routes.GetLinuxRoutes(true)
	if err != nil {
		return infra.Network{}, err
	}

	allRoutes, err = routes.GetLinuxRoutes(false)
	if err != nil {
		return infra.Network{}, err
	}

	for _, i := range interfaces {
		var addresses []string
		var gateways []string
		var ros []infra.Route

		for _, a := range i.Addrs {
			addresses = append(addresses, a.Addr)
		}

		for _, route := range defaultRoutes {
			if route.Interface == i.Name {
				gateways = append(gateways, route.NextHop)
			}
		}

		for _, route := range allRoutes {
			if route.Interface == i.Name {
				ros = append(ros, infra.Route{
					Destination: route.Destination,
					Netmask:     route.Netmask,
					NextHop:     route.NextHop,
				})
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

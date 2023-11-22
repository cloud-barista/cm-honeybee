package infra

import (
	"github.com/cloud-barista/cm-honeybee/lib/routes"
	"github.com/shirou/gopsutil/v3/net"
)

type Route struct {
	Destination string `json:"destination"`
	Netmask     string `json:"netmask"`
	NextHop     string `json:"next_hop"`
}

type PhysicalNetwork struct {
	Interface string   `json:"interface"`
	Address   []string `json:"address"`
	Gateway   []string `json:"gateway"`
	Route     []Route  `json:"route"`
	MAC       string   `json:"mac"`
	MTU       int      `json:"mtu"`
}

type Network struct {
	PhysicalNetwork []PhysicalNetwork `json:"physical_network"`
}

func GetNetworkInfo() (Network, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return Network{}, err
	}

	var physicalNetworks []PhysicalNetwork
	var defaultRoutes []routes.RouteStruct
	var allRoutes []routes.RouteStruct

	defaultRoutes, err = routes.GetLinuxRoutes(true)
	if err != nil {
		return Network{}, err
	}

	allRoutes, err = routes.GetLinuxRoutes(false)
	if err != nil {
		return Network{}, err
	}

	for _, i := range interfaces {
		var addresses []string
		var gateways []string
		var ros []Route

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
				ros = append(ros, Route{
					Destination: route.Destination,
					Netmask:     route.Netmask,
					NextHop:     route.NextHop,
				})
			}
		}

		physicalNetworks = append(physicalNetworks, PhysicalNetwork{
			Interface: i.Name,
			Address:   addresses,
			Gateway:   gateways,
			Route:     ros,
			MAC:       i.HardwareAddr,
			MTU:       i.MTU,
		})
	}

	network := Network{
		PhysicalNetwork: physicalNetworks,
	}

	return network, nil
}

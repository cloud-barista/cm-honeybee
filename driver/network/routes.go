package network

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/lib/routes"
	"github.com/cloud-barista/cm-honeybee/model/network"
	"runtime"
)

func GetRoutes() ([]network.Route, error) {
	var rs []network.Route
	var osRoutes []routes.RouteStruct
	var err error

	if runtime.GOOS == "linux" {
		osRoutes, err = routes.GetLinuxRoutes(false)
	} else if runtime.GOOS == "windows" {
		osRoutes, err = routes.GetWindowsRoutes(false)
	} else {
		return rs, errors.New("unsupported OS")
	}

	if err != nil {
		return rs, err
	}

	for _, r := range osRoutes {
		if r.NextHop == "on-link" {
			r.NextHop = r.Interface
		}

		rs = append(rs, network.Route{
			Destination: r.Destination,
			Netmask:     r.Netmask,
			NextHop:     r.NextHop,
		})
	}

	return rs, nil
}

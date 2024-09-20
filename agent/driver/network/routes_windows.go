//go:build windows

package network

import (
	"github.com/cloud-barista/cm-honeybee/agent/lib/routes"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/network"
)

func GetRoutes() ([]network.Route, error) {
	var rs []network.Route
	var osRoutes []routes.RouteStruct
	var err error

	osRoutes, err = routes.GetRoutes(false)
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
			Source:      "N/A",
			NextHop:     r.NextHop,
			Metric:      r.Metric,
			Scope:       "N/A",
			Proto:       "N/A",
			Link:        "N/A",
		})
	}

	return rs, nil
}

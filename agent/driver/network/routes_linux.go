//go:build linux

package network

import (
	"github.com/cloud-barista/cm-honeybee/agent/lib/routes"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/network"
	"github.com/vishvananda/netlink"
)

func GetRoutes() ([]network.Route, error) {
	var rs []network.Route
	var osRoutes []routes.RouteStruct
	var err error

	osRoutes, err = routes.GetRoutes(false)
	if err != nil {
		return rs, err
	}

	type routeExt struct {
		Source string
		Scope  string
		Proto  string
		Link   string
	}
	var routeExts = make(map[string]routeExt)

	rts, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return rs, err
	}
	for _, rt := range rts {
		iface := "N/A"
		linkState := ""

		if rt.LinkIndex != 0 {
			link, err := netlink.LinkByIndex(rt.LinkIndex)
			if err == nil {
				iface = link.Attrs().Name
				if iface == "N/A" {
					continue
				}

				if link.Attrs().OperState == netlink.OperDown {
					linkState = "down"
				} else {
					linkState = "up"
				}
			}
		}

		source := ""
		if rt.Src != nil {
			source = rt.Src.String()
		}
		scope := rt.Scope.String()
		protocol := rt.Protocol.String()

		routeExts[iface] = routeExt{
			Source: source,
			Scope:  scope,
			Proto:  protocol,
			Link:   linkState,
		}
	}

	for _, r := range osRoutes {
		if r.NextHop == "on-link" {
			r.NextHop = r.Interface
		}

		rt := network.Route{
			Interface:   r.Interface,
			Destination: r.Destination,
			Netmask:     r.Netmask,
			NextHop:     r.NextHop,
			Metric:      r.Metric,
		}

		ext, ok := routeExts[r.Interface]
		if ok {
			rt.Source = ext.Source
			rt.Scope = ext.Scope
			rt.Proto = ext.Proto
			rt.Link = ext.Link
		}

		rs = append(rs, rt)
	}

	return rs, nil
}

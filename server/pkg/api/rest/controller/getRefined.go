package controller

import (
	"fmt"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/network"
	"github.com/jollaman999/utils/logger"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	inframodel "github.com/cloud-barista/cm-model/infra/on-premise-model"
	"github.com/labstack/echo/v4"
)

func netmaskToCIDR(netmask string) (int, error) {
	ip := net.ParseIP(netmask)
	if ip == nil {
		return 0, fmt.Errorf("invalid netmask: %s", netmask)
	}

	mask := net.IPMask(ip.To4())
	if mask == nil {
		return 0, fmt.Errorf("invalid netmask: %s", netmask)
	}

	cidr, _ := mask.Size()

	return cidr, nil
}

func convertRouteToRefinedRoute(route *network.Route, gateway string) (*inframodel.RouteProperty, error) {
	var destination string

	switch route.Family {
	case "ipv4":
		cidr, err := netmaskToCIDR(route.Netmask)
		if err != nil {
			return nil, err
		}
		destination = route.Destination + "/" + strconv.Itoa(cidr)
	case "ipv6":
		destination = route.Destination + route.Netmask
	default:
		return nil, fmt.Errorf("invalid route family: %s", route.Family)
	}

	routeProperty := inframodel.RouteProperty{
		Destination: destination,
		Gateway:     gateway,
		Interface:   route.Interface,
		Metric:      route.Metric,
		Protocol:    route.Proto,
		Scope:       route.Scope,
		Source:      route.Source,
		LinkState:   route.Link,
	}

	return &routeProperty, nil
}

func doGetRefinedInfraInfo(infraInfo *infra.Infra) (*inframodel.ServerProperty, error) {
	var dataDisks []inframodel.DiskProperty

	for _, dataDisk := range infraInfo.Compute.ComputeResource.DataDisk {
		dataDisks = append(dataDisks, inframodel.DiskProperty{
			Label:     dataDisk.Label,
			Type:      dataDisk.Type,
			TotalSize: uint64(dataDisk.Size),
			Available: uint64(dataDisk.Available),
			Used:      uint64(dataDisk.Used),
		})
	}

	var interfaces []inframodel.NetworkInterfaceProperty

	for _, iface := range infraInfo.Network.Host.NetworkInterface {
		interf := inframodel.NetworkInterfaceProperty{
			Name:           iface.Interface,
			MacAddress:     iface.MACAddress,
			IPv4CidrBlocks: []string{},
			IPv6CidrBlocks: []string{},
			Mtu:            iface.MTU,
		}

		for _, address := range iface.Address {
			split := strings.Split(address, "/")
			if len(split) != 2 {
				continue
			}

			validIPv4 := net.ParseIP(split[0]).To4()
			if validIPv4 != nil {
				interf.IPv4CidrBlocks = append(interf.IPv4CidrBlocks, address)
			} else {
				interf.IPv6CidrBlocks = append(interf.IPv6CidrBlocks, address)
			}
		}

		for _, route := range infraInfo.Network.Host.Route {
			if iface.Interface == route.Interface {
				interf.State = route.Link
				break
			}
		}

		interfaces = append(interfaces, interf)
	}

	var routingTable []inframodel.RouteProperty

	for _, route := range infraInfo.Network.Host.Route {
		var gateway string

		for _, iface := range infraInfo.Network.Host.NetworkInterface {
			if iface.Interface == route.Interface {
				gateway = iface.Gateway
				break
			}
		}
		refinedRoute, err := convertRouteToRefinedRoute(&route, gateway)
		if err != nil {
			logger.Println(logger.WARN, true, err.Error())
			continue
		}

		routingTable = append(routingTable, *refinedRoute)
	}

	var firewallTable []inframodel.FirewallRuleProperty

	for _, firewall := range infraInfo.Network.Host.FirewallRule {
		firewallTable = append(firewallTable, inframodel.FirewallRuleProperty{
			SrcCIDR:   firewall.Src,
			DstCIDR:   firewall.Dst,
			SrcPorts:  firewall.SrcPorts,
			DstPorts:  firewall.DstPorts,
			Protocol:  firewall.Protocol,
			Direction: firewall.Direction,
			Action:    firewall.Action,
		})
	}

	refinedInfraInfo := inframodel.ServerProperty{
		Hostname: infraInfo.Compute.OS.Node.Hostname,
		CPU: inframodel.CpuProperty{
			Architecture: infraInfo.Compute.OS.Kernel.Architecture,
			Cpus:         uint32(infraInfo.Compute.ComputeResource.CPU.Cpus),
			Cores:        uint32(infraInfo.Compute.ComputeResource.CPU.Cores),
			Threads:      uint32(infraInfo.Compute.ComputeResource.CPU.Threads),
			MaxSpeed:     float32(infraInfo.Compute.ComputeResource.CPU.MaxSpeed) / 1000, // GHz
			Vendor:       infraInfo.Compute.ComputeResource.CPU.Vendor,
			Model:        infraInfo.Compute.ComputeResource.CPU.Model,
		},
		Memory: inframodel.MemoryProperty{
			Type:      infraInfo.Compute.ComputeResource.Memory.Type,
			TotalSize: uint64(infraInfo.Compute.ComputeResource.Memory.Size / 1024),      // GiB
			Available: uint64(infraInfo.Compute.ComputeResource.Memory.Available / 1024), // GiB
			Used:      uint64(infraInfo.Compute.ComputeResource.Memory.Used / 1024),      // GiB
		},
		RootDisk: inframodel.DiskProperty{
			Label:     infraInfo.Compute.ComputeResource.RootDisk.Label,
			Type:      infraInfo.Compute.ComputeResource.RootDisk.Type,
			TotalSize: uint64(infraInfo.Compute.ComputeResource.RootDisk.Size),      // GiB
			Available: uint64(infraInfo.Compute.ComputeResource.RootDisk.Available), // GiB
			Used:      uint64(infraInfo.Compute.ComputeResource.RootDisk.Used),      // GiB
		},
		DataDisks:     dataDisks,
		Interfaces:    interfaces,
		RoutingTable:  routingTable,
		FirewallTable: firewallTable,
		OS: inframodel.OsProperty{
			PrettyName:      infraInfo.Compute.OS.OS.PrettyName,
			Version:         infraInfo.Compute.OS.OS.Version,
			Name:            infraInfo.Compute.OS.OS.Name,
			VersionID:       infraInfo.Compute.OS.OS.VersionID,
			VersionCodename: infraInfo.Compute.OS.OS.VersionCodename,
			ID:              infraInfo.Compute.OS.OS.ID,
			IDLike:          infraInfo.Compute.OS.OS.IDLike,
		},
	}

	return &refinedInfraInfo, nil
}

func doGetRefinedNetworkInfo(networkProperty *inframodel.NetworkProperty, routes *[]network.Route, machineID *string) {
	for _, route := range *routes {
		if strings.ToLower(route.Interface) == "lo" {
			continue
		}

		if route.Family == "ipv4" {
			if route.Destination == "0.0.0.0" && route.Netmask == "0.0.0.0" {
				var gatewayProperty inframodel.GatewayProperty

				gatewayProperty.IP = route.NextHop
				gatewayProperty.InterfaceName = route.Interface
				gatewayProperty.MachineId = *machineID

				networkProperty.IPv4Networks.DefaultGateways = append(networkProperty.IPv4Networks.DefaultGateways, gatewayProperty)
			} else {
				var dup bool

				cidr, err := netmaskToCIDR(route.Destination)
				if err != nil {
					logger.Println(logger.ERROR, true, err.Error())
				}
				cidrBlock := route.Destination + "/" + strconv.Itoa(cidr)

				for _, ipv4CidrBlock := range networkProperty.IPv4Networks.CidrBlocks {
					if ipv4CidrBlock == cidrBlock {
						dup = true
						break
					}
				}
				if dup {
					continue
				}

				networkProperty.IPv4Networks.CidrBlocks = append(networkProperty.IPv4Networks.CidrBlocks, cidrBlock)
			}
		} else if route.Family == "ipv6" {
			if route.Destination == "::" && route.Netmask == "/0" {
				var gatewayProperty inframodel.GatewayProperty

				gatewayProperty.IP = route.NextHop
				gatewayProperty.InterfaceName = route.Interface
				gatewayProperty.MachineId = *machineID

				networkProperty.IPv6Networks.DefaultGateways = append(networkProperty.IPv6Networks.DefaultGateways, gatewayProperty)
			} else {
				if strings.HasPrefix(route.Destination, "fe80:") { // Skip lick local addresses
					continue
				} else if strings.HasPrefix(route.Destination, "ff") { // Skip multicast addresses
					continue
				}

				var dup bool

				cidrBlock := route.Destination + route.Netmask

				for _, ipv6CidrBlock := range networkProperty.IPv6Networks.CidrBlocks {
					if ipv6CidrBlock == cidrBlock {
						dup = true
						break
					}
				}
				if dup {
					continue
				}

				networkProperty.IPv6Networks.CidrBlocks = append(networkProperty.IPv6Networks.CidrBlocks, cidrBlock)
			}
		}
	}
}

// GetInfraInfoRefined godoc
//
//	@ID				get-infra-info-refined
//	@Summary		Get Refined Infra Information
//	@Description	Get the refined infra information of the connection information.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	inframodel.OnpremiseInfraModel	"Successfully get refined information of the infra."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the infra."
//	@Router			/source_group/{sgId}/connection_info/{connId}/infra/refined [get]
func GetInfraInfoRefined(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	infraInfo, err := doGetInfraInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	refinedInfraInfo, err := doGetRefinedInfraInfo(infraInfo)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var onpremiseInfraModel inframodel.OnpremiseInfraModel
	var onpremiseInfra inframodel.OnpremInfra

	onpremiseInfra.Servers = append(onpremiseInfra.Servers, *refinedInfraInfo)
	doGetRefinedNetworkInfo(&onpremiseInfra.Network, &infraInfo.Network.Host.Route, &infraInfo.Compute.OS.Node.Machineid)

	onpremiseInfraModel.OnpremiseInfraModel = onpremiseInfra

	return c.JSONPretty(http.StatusOK, onpremiseInfraModel, " ")
}

// GetInfraInfoSourceGroupRefined godoc
//
//	@ID				get-infra-info-source-group-refined
//	@Summary		Get Refined Infra Information Source Group
//	@Description	Get the refined infra information for all connections in the source group.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	inframodel.OnpremiseInfraModel		"Successfully get refined information of the infra."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the infra."
//	@Router		/source_group/{sgId}/infra/refined [get]
func GetInfraInfoSourceGroupRefined(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var onpremiseInfraModel inframodel.OnpremiseInfraModel
	var onpremiseInfra inframodel.OnpremInfra

	for _, conn := range *list {
		infraInfo, err := doGetInfraInfo(conn.ID)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		refinedInfraInfo, err := doGetRefinedInfraInfo(infraInfo)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		onpremiseInfra.Servers = append(onpremiseInfra.Servers, *refinedInfraInfo)
		doGetRefinedNetworkInfo(&onpremiseInfra.Network, &infraInfo.Network.Host.Route, &infraInfo.Compute.OS.Node.Machineid)
	}

	onpremiseInfraModel.OnpremiseInfraModel = onpremiseInfra

	return c.JSONPretty(http.StatusOK, onpremiseInfraModel, " ")
}

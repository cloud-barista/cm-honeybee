package controller

import (
	"net"
	"net/http"
	"strings"

	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra" // Need for swag
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	inframodel "github.com/cloud-barista/cm-model/infra/onprem"
	"github.com/labstack/echo/v4"
)

func doGetRefinedInfraInfo(connID string) (*inframodel.ServerProperty, error) {
	infraInfo, err := doGetInfraInfo(connID)
	if err != nil {
		return nil, err
	}

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

		routingTable = append(routingTable, inframodel.RouteProperty{
			Interface:   route.Interface,
			Destination: route.Destination,
			Gateway:     gateway,
			Metric:      route.Metric,
			Protocol:    route.Proto,
			Scope:       route.Scope,
			Source:      route.Source,
			LinkState:   route.Link,
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
		DataDisks:    dataDisks,
		Interfaces:   interfaces,
		RoutingTable: routingTable,
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

	refinedInfraInfo, err := doGetRefinedInfraInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var onpremiseInfraModel inframodel.OnpremiseInfraModel
	var onpremiseInfra inframodel.OnpremInfra

	onpremiseInfra.Servers = append(onpremiseInfra.Servers, *refinedInfraInfo)

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
		refinedInfraInfo, err := doGetRefinedInfraInfo(conn.ID)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		onpremiseInfra.Servers = append(onpremiseInfra.Servers, *refinedInfraInfo)
	}

	onpremiseInfraModel.OnpremiseInfraModel = onpremiseInfra

	return c.JSONPretty(http.StatusOK, onpremiseInfraModel, " ")
}

package controller

import (
	"net/http"

	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra" // Need for swag
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/cloud-barista/cm-model/infra/onprem"
	"github.com/labstack/echo/v4"
)

func doGetRefinedInfraInfo(connID string) (*onprem.ServerProperty, error) {
	infraInfo, err := doGetInfraInfo(connID)
	if err != nil {
		return nil, err
	}

	var dataDisks []onprem.DiskProperty

	for _, dataDisk := range infraInfo.Compute.ComputeResource.DataDisk {
		dataDisks = append(dataDisks, onprem.DiskProperty{
			Label:     dataDisk.Label,
			Type:      dataDisk.Type,
			TotalSize: uint64(dataDisk.Size),
			Available: uint64(dataDisk.Available),
			Used:      uint64(dataDisk.Used),
		})
	}

	var interfaces []onprem.NetworkInterfaceProperty

	for _, iface := range infraInfo.Network.Host.NetworkInterface {
		var ipAddress string
		if len(iface.Address) > 0 {
			ipAddress = iface.Address[0]
		}
		interfaces = append(interfaces, onprem.NetworkInterfaceProperty{
			MacAddress: iface.MACAddress,
			IpAddress:  ipAddress,
			Mtu:        iface.MTU,
		})
	}

	refinedInfraInfo := onprem.ServerProperty{
		Hostname: infraInfo.Compute.OS.Node.Hostname,
		CPU: onprem.CpuProperty{
			Architecture: infraInfo.Compute.OS.Kernel.Architecture,
			Cpus:         uint32(infraInfo.Compute.ComputeResource.CPU.Cpus),
			Cores:        uint32(infraInfo.Compute.ComputeResource.CPU.Cores),
			Threads:      uint32(infraInfo.Compute.ComputeResource.CPU.Threads),
			MaxSpeed:     float32(infraInfo.Compute.ComputeResource.CPU.MaxSpeed) / 1024, // GHz
			Vendor:       infraInfo.Compute.ComputeResource.CPU.Vendor,
			Model:        infraInfo.Compute.ComputeResource.CPU.Model,
		},
		Memory: onprem.MemoryProperty{
			Type:      infraInfo.Compute.ComputeResource.Memory.Type,
			TotalSize: uint64(infraInfo.Compute.ComputeResource.Memory.Size / 1024),      // GiB
			Available: uint64(infraInfo.Compute.ComputeResource.Memory.Available / 1024), // GiB
			Used:      uint64(infraInfo.Compute.ComputeResource.Memory.Used / 1024),      // GiB
		},
		RootDisk: onprem.DiskProperty{
			Label:     infraInfo.Compute.ComputeResource.RootDisk.Label,
			Type:      infraInfo.Compute.ComputeResource.RootDisk.Type,
			TotalSize: uint64(infraInfo.Compute.ComputeResource.RootDisk.Size),      // GiB
			Available: uint64(infraInfo.Compute.ComputeResource.RootDisk.Available), // GiB
			Used:      uint64(infraInfo.Compute.ComputeResource.RootDisk.Used),      // GiB
		},
		DataDisks:  dataDisks,
		Interfaces: interfaces,
		OS: onprem.OsProperty{
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
//	@Success		200	{object}	infra.Infra				"Successfully get refined information of the infra."
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

	return c.JSONPretty(http.StatusOK, refinedInfraInfo, " ")
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
//	@Success		200	{object}	model.InfraInfoList		"Successfully get refined information of the infra."
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

	var onPremInfra onprem.OnPremInfra
	gateways := make(map[string]int)
	var gateway string

	for _, conn := range *list {
		refinedInfraInfo, _ := doGetRefinedInfraInfo(conn.ID)
		onPremInfra.Servers = append(onPremInfra.Servers, *refinedInfraInfo)

		infraInfo, _ := doGetInfraInfo(conn.ID)
		for _, iface := range infraInfo.Network.Host.NetworkInterface {
			if iface.Gateway != nil {
				for _, gw := range iface.Gateway {
					val, ok := gateways[gw]
					if ok {
						gateways[gw] = val + 1
					} else {
						gateways[gw] = 1
					}
				}
			}
		}
	}

	for key := range gateways {
		if gateways[gateway] < gateways[key] {
			gateway = key
		}
	}

	onPremInfra.Network.Gateway = gateway

	return c.JSONPretty(http.StatusOK, onPremInfra, " ")
}

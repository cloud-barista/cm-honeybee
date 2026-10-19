package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	rdbmsmodel "github.com/cloud-barista/cm-beetle/imdl/rdbms-model"
	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
)

// rdbmsRoleReplica is the only role honeybee reports. A primary and a
// standalone instance look the same to cm-beetle, which fills an empty role
// with "standalone", so claiming either without evidence adds nothing.
const rdbmsRoleReplica = "replica"

// errRDBMSNotApplicable marks a connection whose resource_type cannot hold a
// database at all (k8s, object_storage, nlb). Source group queries skip these
// connections instead of failing the whole group.
type errRDBMSNotApplicable struct{ resourceType string }

func (e errRDBMSNotApplicable) Error() string {
	return "rdbms information is not available for resource_type " + e.resourceType
}

// errRDBMSNotCollected marks a managed database connection that has not been
// imported yet. Source group queries skip it, as they skip a VM whose software
// has not been imported.
var errRDBMSNotCollected = errors.New("rdbms information has not been collected")

// isManagedRDBMS reports whether the connection is a CSP managed database,
// collected through the CSP driver into SavedRDBMSInfo.
func isManagedRDBMS(connectionInfo *model.ConnectionInfo) bool {
	return connectionInfo.ResourceType == serverCommon.ResourceTypeRDBMS
}

// isInstalledRDBMSHost reports whether the connection is a machine a database
// may be installed on. An on-prem connection registered before resource_type
// existed carries an empty value, which means a VM.
func isInstalledRDBMSHost(connectionInfo *model.ConnectionInfo) bool {
	return connectionInfo.ResourceType == "" || connectionInfo.ResourceType == serverCommon.ResourceTypeVM
}

func doGetManagedRDBMSInfo(connectionInfo *model.ConnectionInfo) (*model.RDBMSInfo, error) {
	savedRDBMSInfo, err := dao.SavedRDBMSInfoGet(connectionInfo.ID)
	if err != nil {
		errMsg := "Failed to get information of the rdbms." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, fmt.Errorf("%s: %w", errMsg, errRDBMSNotCollected)
	}

	var rdbmsInfo model.RDBMSInfo
	err = json.Unmarshal([]byte(savedRDBMSInfo.RDBMSData), &rdbmsInfo)
	if err != nil || rdbmsInfo.RDBMS == nil {
		errMsg := "Error occurred while parsing rdbms information." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	return &rdbmsInfo, nil
}

// doGetInstalledRDBMSInfo returns the raw database findings for a VM: the
// software entries judged to be a database server, with the evidence for each.
// A VM with no database gives an empty list.
func doGetInstalledRDBMSInfo(connID string) ([]model.RDBMSInfo, error) {
	softwareInfo, err := doGetInstalledSoftwareForRDBMS(connID)
	if err != nil {
		return nil, err
	}

	infos := []model.RDBMSInfo{}
	for _, db := range findInstalledRDBMS(softwareInfo) {
		infos = append(infos, model.RDBMSInfo{Installed: &db})
	}

	return infos, nil
}

// doGetRDBMSInfo returns the raw database information of one connection,
// choosing the source by the connection's resource_type.
func doGetRDBMSInfo(connID string) ([]model.RDBMSInfo, error) {
	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, err
	}

	switch {
	case isManagedRDBMS(connectionInfo):
		rdbmsInfo, err := doGetManagedRDBMSInfo(connectionInfo)
		if err != nil {
			return nil, err
		}
		return []model.RDBMSInfo{*rdbmsInfo}, nil
	case isInstalledRDBMSHost(connectionInfo):
		return doGetInstalledRDBMSInfo(connectionInfo.ID)
	default:
		return nil, errRDBMSNotApplicable{resourceType: connectionInfo.ResourceType}
	}
}

// doGetRefinedInstalledRDBMS assembles the databases installed on a VM from the
// saved software information (which package or process is a database server,
// and on which port it listens) and the saved infra information (the host's
// CPU, memory and disks). A VM with no database gives an empty list.
func doGetRefinedInstalledRDBMS(connID string) ([]rdbmsmodel.SourceRDBMSProperty, error) {
	softwareInfo, err := doGetInstalledSoftwareForRDBMS(connID)
	if err != nil {
		return nil, err
	}

	found := findInstalledRDBMS(softwareInfo)
	instances := []rdbmsmodel.SourceRDBMSProperty{}
	if len(found) == 0 {
		return instances, nil
	}

	node := installedRDBMSNode(connID)
	for _, db := range found {
		instances = append(instances, refineInstalledRDBMS(db, node))
	}

	return instances, nil
}

// doGetRefinedRDBMSInfo returns the refined databases of one connection,
// choosing the source by the connection's resource_type.
func doGetRefinedRDBMSInfo(connID string) ([]rdbmsmodel.SourceRDBMSProperty, error) {
	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, err
	}

	switch {
	case isManagedRDBMS(connectionInfo):
		rdbmsInfo, err := doGetManagedRDBMSInfo(connectionInfo)
		if err != nil {
			return nil, err
		}
		return []rdbmsmodel.SourceRDBMSProperty{refineManagedRDBMS(rdbmsInfo)}, nil
	case isInstalledRDBMSHost(connectionInfo):
		return doGetRefinedInstalledRDBMS(connectionInfo.ID)
	default:
		return nil, errRDBMSNotApplicable{resourceType: connectionInfo.ResourceType}
	}
}

// refineManagedRDBMS maps a collected managed database onto the source model.
//
// A value the CSP did not report stays zero rather than being guessed: cm-beetle
// fills zero CPU, memory, storage, port and role with its own defaults and says
// so in its warnings, which a guess here would hide.
func refineManagedRDBMS(info *model.RDBMSInfo) rdbmsmodel.SourceRDBMSProperty {
	db := info.RDBMS

	node := rdbmsmodel.DBNodeProperty{}
	if info.DBSpec != nil {
		node.CPU = rdbmsCPU(info.DBSpec.VCpu)
		node.Memory.TotalSize = rdbmsMemoryGiB(info.DBSpec.MemSizeMiB)
	}
	if size := parseUintOrZero(db.StorageSize); size > 0 {
		// RootDisk, not DataDisks: cm-beetle sums both, and RootDisk is where its
		// own storage default goes, so a filled and a defaulted instance look alike.
		node.RootDisk = rdbmsmodel.DiskProperty{
			Type:      db.StorageType,
			TotalSize: size, // GB
		}
	}

	return rdbmsmodel.SourceRDBMSProperty{
		DisplayName: db.IId.NameId,
		DBNode:      node,
		DBEngine: rdbmsmodel.DBEngineProperty{
			Engine:        normalizeRDBMSEngine(db.DBEngine),
			EngineVersion: db.DBEngineVersion,
			Port:          endpointPort(db.Endpoint),
			Role:          rdbmsRole(db.KeyValueList),
		},
	}
}

// normalizeRDBMSEngine lowercases the engine name and maps AWS's "postgres" to
// the "postgresql" every other driver and cm-beetle use. AWS passes its API's
// engine name through unchanged, while the GCP driver already does this mapping.
func normalizeRDBMSEngine(engine string) string {
	e := strings.ToLower(strings.TrimSpace(engine))
	if e == "postgres" {
		return "postgresql"
	}
	return e
}

// rdbmsCPU turns a spec's vCPU count into the CPU shape cm-beetle reads: it
// counts Threads first, then Cpus x Cores, and fills a missing CPU as one
// socket with that many cores and threads. A count of "-1" (not exposed by the
// CSP) leaves the CPU empty.
func rdbmsCPU(vcpu spider.VCpuInfo) rdbmsmodel.CpuProperty {
	count := parseUintOrZero(vcpu.Count)
	if count == 0 || count > uint64(^uint32(0)) {
		return rdbmsmodel.CpuProperty{}
	}

	cpu := rdbmsmodel.CpuProperty{
		Cpus:    1,
		Cores:   uint32(count),
		Threads: uint32(count),
	}
	if ghz, err := strconv.ParseFloat(strings.TrimSpace(vcpu.ClockGHz), 32); err == nil && ghz > 0 {
		cpu.MaxSpeed = float32(ghz)
	}

	return cpu
}

// rdbmsMemoryGiB converts a spec's memory to GiB; "-1" gives 0.
func rdbmsMemoryGiB(memSizeMiB string) uint64 {
	return mibToGiBCapacity(parseUintOrZero(memSizeMiB))
}

// parseUintOrZero reads a driver's numeric string. cb-spider reports an unknown
// number as "-1" or "NA", both of which give 0.
func parseUintOrZero(s string) uint64 {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// endpointPort reads the port out of a driver's Endpoint. Most drivers report
// "host:port", but some report the host alone, and IBM reports "NA"; those give
// 0, the same as an absent port.
func endpointPort(endpoint string) int {
	_, portStr, err := net.SplitHostPort(strings.TrimSpace(endpoint))
	if err != nil {
		return 0
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return 0
	}
	return port
}

// rdbmsRole reports "replica" only when the CSP's own record says the instance
// replicates another one, and "" otherwise. cb-spider has no role field, and
// its DBInstanceType means something different on every CSP, so the evidence is
// read from the flattened CSP record in KeyValueList:
//   - AWS: ReadReplicaSourceDBInstanceIdentifier names the source instance.
//   - GCP: InstanceType is READ_REPLICA_INSTANCE, or MasterInstanceName names
//     the primary.
func rdbmsRole(kvs []spider.KeyValue) string {
	kv := keyValueListToMap(kvs)

	if strings.TrimSpace(kv["ReadReplicaSourceDBInstanceIdentifier"]) != "" {
		return rdbmsRoleReplica
	}
	if kv["InstanceType"] == "READ_REPLICA_INSTANCE" || strings.TrimSpace(kv["MasterInstanceName"]) != "" {
		return rdbmsRoleReplica
	}

	return ""
}

// rdbmsSourceCloud describes where a CSP source group's databases run. An
// on-prem group has no cloud, and gives nil.
func rdbmsSourceCloud(sourceGroup *model.SourceGroup) *rdbmsmodel.CloudProperty {
	if !serverCommon.IsCSPType(sourceGroup.Type) {
		return nil
	}

	region, zone, err := splitRegionZone(sourceGroup.RegionName, "")
	if err != nil {
		return nil
	}

	return &rdbmsmodel.CloudProperty{
		Csp:    strings.ToLower(sourceGroup.ProviderName),
		Region: region,
		Zone:   zone,
	}
}

// skippableInSourceGroup reports whether a connection's error only means it has
// no database information to contribute to a source group query.
func skippableInSourceGroup(err error) bool {
	var notApplicable errRDBMSNotApplicable
	return errors.As(err, &notApplicable) || errors.Is(err, errRDBMSNotCollected)
}

// GetRDBMSInfo godoc
//
//	@ID				get-rdbms-info
//	@Summary		Get RDBMS Information
//	@Description	Get the database information of the connection information. A CSP managed database (resource_type rdbms) gives what the CSP driver reported; a VM gives the databases installed on it.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	model.RDBMSInfoList		"Successfully get information of the rdbms."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the rdbms."
//	@Router			/source_group/{sgId}/connection_info/{connId}/rdbms [get]
func GetRDBMSInfo(c echo.Context) error {
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

	rdbmsInfo, err := doGetRDBMSInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, model.RDBMSInfoList{Servers: rdbmsInfo}, " ")
}

// GetRDBMSInfoSourceGroup godoc
//
//	@ID				get-rdbms-info-source-group
//	@Summary		Get RDBMS Information Source Group
//	@Description	Get the database information for all connections in the source group. Connections that cannot hold a database or have not been collected are skipped.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	model.RDBMSInfoList		"Successfully get information of the rdbms."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the rdbms."
//	@Router			/source_group/{sgId}/rdbms [get]
func GetRDBMSInfoSourceGroup(c echo.Context) error {
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

	rdbmsInfoList := model.RDBMSInfoList{Servers: []model.RDBMSInfo{}}

	for _, conn := range *list {
		rdbmsInfo, err := doGetRDBMSInfo(conn.ID)
		if err != nil {
			continue // not applicable / not collected yet / unreadable - skip (logged in doGetManagedRDBMSInfo)
		}
		rdbmsInfoList.Servers = append(rdbmsInfoList.Servers, rdbmsInfo...)
	}

	return c.JSONPretty(http.StatusOK, rdbmsInfoList, " ")
}

// GetRDBMSInfoRefined godoc
//
//	@ID				get-rdbms-info-refined
//	@Summary		Get Refined RDBMS Information
//	@Description	Get the refined database information of the connection information, in the source model cm-beetle recommends from. A value the source did not report is left empty for cm-beetle to fill in.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	rdbmsmodel.SourceRDBMSModel	"Successfully get refined information of the rdbms."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the rdbms."
//	@Router			/source_group/{sgId}/connection_info/{connId}/rdbms/refined [get]
func GetRDBMSInfoRefined(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	instances, err := doGetRefinedRDBMSInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var sourceRDBMSModel rdbmsmodel.SourceRDBMSModel
	sourceRDBMSModel.SourceRDBMSModel.SourceCloud = rdbmsSourceCloud(sourceGroup)
	sourceRDBMSModel.SourceRDBMSModel.SourceRDBMSInstances = instances

	return c.JSONPretty(http.StatusOK, sourceRDBMSModel, " ")
}

// GetRDBMSInfoSourceGroupRefined godoc
//
//	@ID				get-rdbms-info-source-group-refined
//	@Summary		Get Refined RDBMS Information Source Group
//	@Description	Get the refined database information for all connections in the source group, in the source model cm-beetle recommends from. Connections that cannot hold a database or have not been collected are skipped.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	rdbmsmodel.SourceRDBMSModel	"Successfully get refined information of the rdbms."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the rdbms."
//	@Router			/source_group/{sgId}/rdbms/refined [get]
func GetRDBMSInfoSourceGroupRefined(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	instances := []rdbmsmodel.SourceRDBMSProperty{}

	for _, conn := range *list {
		refined, err := doGetRefinedRDBMSInfo(conn.ID)
		if err != nil {
			if skippableInSourceGroup(err) {
				continue
			}
			return common.ReturnErrorMsg(c, err.Error())
		}
		instances = append(instances, refined...)
	}

	var sourceRDBMSModel rdbmsmodel.SourceRDBMSModel
	sourceRDBMSModel.SourceRDBMSModel.SourceCloud = rdbmsSourceCloud(sourceGroup)
	sourceRDBMSModel.SourceRDBMSModel.SourceRDBMSInstances = instances

	return c.JSONPretty(http.StatusOK, sourceRDBMSModel, " ")
}

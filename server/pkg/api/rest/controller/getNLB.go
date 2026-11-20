package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	inframodel "github.com/cloud-barista/cm-beetle/imdl/on-premise-model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
)

// isCSPNLB reports whether the connection is a CSP nlb connection, whose
// collected NLB is kept in SavedNLBInfo.
func isCSPNLB(sourceGroup *model.SourceGroup, connectionInfo *model.ConnectionInfo) bool {
	return serverCommon.IsCSPType(sourceGroup.Type) &&
		connectionInfo.ResourceType == serverCommon.ResourceTypeNLB
}

// readSavedNLB reads the stored NLB of a connection together with the status
// of its last import. A record that holds no NLB, which a failed first import
// leaves, is an error.
func readSavedNLB(connID string) (*model.CSPNLBRecord, string, error) {
	saved, err := dao.SavedNLBInfoGet(connID)
	if err != nil {
		return nil, "", errors.New("nlb information has not been collected; run import/infra." +
			" (ConnectionID = " + connID + ")")
	}
	if strings.TrimSpace(saved.NLBData) == "" {
		return nil, saved.Status, errors.New("nlb information has not been collected successfully; run import/infra." +
			" (ConnectionID = " + connID + ", Status = " + saved.Status + ")")
	}

	var rec model.CSPNLBRecord
	if err := json.Unmarshal([]byte(saved.NLBData), &rec); err != nil {
		errMsg := "Error occurred while parsing nlb information." +
			" (ConnectionID = " + connID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, saved.Status, errors.New(errMsg)
	}
	// A record stored before vm_group_unknown existed lacks the field; the
	// empty VM group it stored tells the same thing.
	if rec.VMGroup.Protocol == "" && rec.VMGroup.Port == "" && len(rec.VMGroup.VMs) == 0 {
		rec.VMGroupUnknown = true
	}

	return &rec, saved.Status, nil
}

// doGetNLBInfo returns the stored NLB of a csp nlb connection. The last NLB
// collected is returned even when a later import failed, as GET does for the
// other CSP resources; refined leaves such a record out.
func doGetNLBInfo(connID string) (*model.CSPNLBRecord, error) {
	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, err
	}
	if connectionInfo.ResourceType != serverCommon.ResourceTypeNLB {
		return nil, errors.New("nlb information is not available for resource_type '" + connectionInfo.ResourceType + "'")
	}

	rec, _, err := readSavedNLB(connectionInfo.ID)
	return rec, err
}

// tryGetCSPNLB returns the stored NLB of the connection when its last import
// succeeded, or nil. A record whose last import failed is no longer current,
// and refined output is what a migration is planned from.
func tryGetCSPNLB(connID string) *model.CSPNLBRecord {
	rec, status, err := readSavedNLB(connID)
	if err != nil {
		return nil
	}
	if status != model.ConnectionInfoStatusSuccess {
		logger.Println(logger.WARN, false, "Skipping nlb information whose last import did not succeed."+
			" (ConnectionID = "+connID+", Status = "+status+")")
		return nil
	}

	return rec
}

// cspVMPrivateIPs maps the group's CSP VMs to their private IPs, keyed by
// every identifier an NLB may name the VM by: the CSP ID and name the CSP
// collection stored, and the connection's resource_id. The VMs read are those
// that become refined nodes: the last infra import succeeded, or only its
// agent call failed after the CSP collection succeeded.
func cspVMPrivateIPs(sourceGroup *model.SourceGroup, conns []model.ConnectionInfo) map[string]string {
	ips := map[string]string{}
	if !serverCommon.IsCSPType(sourceGroup.Type) {
		return ips
	}

	for i := range conns {
		conn := &conns[i]
		if conn.ResourceType != serverCommon.ResourceTypeVM {
			continue
		}
		rec, err := dao.SavedInfraInfoGet(conn.ID)
		if err != nil || rec == nil || strings.TrimSpace(rec.CSPData) == "" {
			continue
		}
		if rec.Status != model.ConnectionInfoStatusSuccess && !cspNodeAfterAgentFailure(sourceGroup, conn, rec) {
			continue
		}
		var csp infra.CSPInfo
		if err := json.Unmarshal([]byte(rec.CSPData), &csp); err != nil || csp.PrivateIP == "" {
			continue
		}
		for _, key := range []string{csp.ID, csp.Name, conn.ResourceID, cspVMIdentifier(conn.ResourceID)} {
			if key != "" {
				ips[key] = csp.PrivateIP
			}
		}
	}

	return ips
}

// refinedCSPNLBs gives the refined NLB entries of the CSP nlb connections
// among conns whose last import succeeded. The backend servers are given the
// private IPs of the CSP VMs in allConns, the whole source group.
func refinedCSPNLBs(sourceGroup *model.SourceGroup, conns []model.ConnectionInfo, allConns []model.ConnectionInfo) []inframodel.NlbProperty {
	var nlbs []inframodel.NlbProperty
	var vmIPs map[string]string

	for i := range conns {
		if !isCSPNLB(sourceGroup, &conns[i]) {
			continue
		}
		rec := tryGetCSPNLB(conns[i].ID)
		if rec == nil {
			continue
		}
		if vmIPs == nil {
			vmIPs = cspVMPrivateIPs(sourceGroup, allConns)
		}
		nlbs = append(nlbs, cspNLBToNlbProperty(rec, sourceGroup.ProviderName, vmIPs))
	}

	return nlbs
}

// GetNLBInfo godoc
//
//	@ID				get-nlb-info
//	@Summary		Get NLB Information
//	@Description	Get the network load balancer collected from the CSP for a csp nlb connection. The last NLB collected is returned even when a later import failed.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	model.CSPNLBRecord		"Successfully get information of the nlb."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the nlb."
//	@Router			/source_group/{sgId}/connection_info/{connId}/nlb [get]
func GetNLBInfo(c echo.Context) error {
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

	nlbInfo, err := doGetNLBInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, nlbInfo, " ")
}

// GetNLBInfoSourceGroup godoc
//
//	@ID				get-nlb-info-source-group
//	@Summary		Get NLB Information Source Group
//	@Description	Get the network load balancers collected from the CSP for all csp nlb connections in the source group. Connections that are not nlb connections, have not been collected or whose last import failed are skipped.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	model.NLBInfoList		"Successfully get information of the nlb."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the nlb."
//	@Router			/source_group/{sgId}/nlb [get]
func GetNLBInfoSourceGroup(c echo.Context) error {
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

	nlbInfoList := model.NLBInfoList{Servers: []model.CSPNLBRecord{}}

	for i := range *list {
		if !isCSPNLB(sourceGroup, &(*list)[i]) {
			continue
		}
		rec := tryGetCSPNLB((*list)[i].ID)
		if rec == nil {
			continue // not collected yet / last import failed / unreadable - skip
		}
		nlbInfoList.Servers = append(nlbInfoList.Servers, *rec)
	}

	return c.JSONPretty(http.StatusOK, nlbInfoList, " ")
}

package controller

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/google/uuid"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
)

func deleteSavedInfraInfo(connectionInfo *model.ConnectionInfo) {
	savedInfraInfo, _ := dao.SavedInfraInfoGet(connectionInfo.ID)
	if savedInfraInfo == nil {
		return
	}
	err := dao.SavedInfraInfoDelete(savedInfraInfo)
	if err != nil {
		logger.Println(logger.ERROR, true, err)
	}
}

func deleteSavedSoftwareInfo(connectionInfo *model.ConnectionInfo) {
	savedSoftwareInfo, _ := dao.SavedSoftwareInfoGet(connectionInfo.ID)
	if savedSoftwareInfo == nil {
		return
	}
	err := dao.SavedSoftwareInfoDelete(savedSoftwareInfo)
	if err != nil {
		logger.Println(logger.ERROR, true, err)
	}
}

func deleteSavedKubernetesInfo(connectionInfo *model.ConnectionInfo) {
	savedKubernetesInfo, _ := dao.SavedKubernetesInfoGet(connectionInfo.ID)
	if savedKubernetesInfo == nil {
		return
	}
	err := dao.SavedKubernetesInfoDelete(savedKubernetesInfo)
	if err != nil {
		logger.Println(logger.ERROR, true, err)
	}
}

func doDeleteSourceGroup(sourceGroupID string) error {
	sourceGroup, err := dao.SourceGroupGet(sourceGroupID)
	if err != nil {
		return err
	}

	connectionInfoList, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{
		SourceGroupID: sourceGroupID,
	}, 0, 0)
	if err != nil {
		return errors.New("failed to get connection info list to delete")
	}
	for _, connectionInfo := range *connectionInfoList {
		deleteSavedInfraInfo(&connectionInfo)
		deleteSavedSoftwareInfo(&connectionInfo)
		deleteSavedKubernetesInfo(&connectionInfo)
		err = dao.ConnectionInfoDelete(&connectionInfo)
		if err != nil {
			logger.Println(logger.ERROR, true, err)
		}
	}

	unregisterCSPFromSpider(sourceGroup)

	err = dao.SourceGroupDelete(sourceGroup)
	if err != nil {
		return err
	}

	return nil
}

// CreateSourceGroup godoc
//
//	@ID				register-source-group
//	@Summary		Register SourceGroup
//	@Description	Register the source group.
//	@Tags			[On-premise] SourceGroup
//	@Accept			json
//	@Produce		json
//	@Param			SourceGroup body model.CreateSourceGroupReq true "source group of the node."
//	@Success		200	{object}	model.SourceGroupRes		"Successfully register the source group"
//	@Failure		400	{object}	common.ErrorResponse		"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse		"Failed to register the source group"
//	@Router			/source_group [post]
func CreateSourceGroup(c echo.Context) error {
	createSourceGroupReq := new(model.CreateSourceGroupReq)
	err := c.Bind(createSourceGroupReq)
	if err != nil {
		return err
	}

	if createSourceGroupReq.Name == "" {
		return common.ReturnErrorMsg(c, "Please provide the source group's name.")
	}

	if len(createSourceGroupReq.ConnectionInfo) > model.ConnectionInfoMaxLength {
		return common.ReturnErrorMsg(c, "Maximum number of connection info is exceeded."+
			" (Max: "+strconv.Itoa(model.ConnectionInfoMaxLength)+")")
	}

	sgType := strings.ToLower(strings.TrimSpace(createSourceGroupReq.Type))
	if sgType == "" {
		sgType = serverCommon.SourceGroupTypeSSH
	}
	if sgType != serverCommon.SourceGroupTypeSSH && sgType != serverCommon.SourceGroupTypeCSP {
		return common.ReturnErrorMsg(c, "type must be 'ssh' or 'csp'.")
	}

	sourceGroup := &model.SourceGroup{
		ID:          uuid.New().String(),
		Name:        createSourceGroupReq.Name,
		Description: createSourceGroupReq.Description,
		Type:        sgType,
	}

	if sgType == serverCommon.SourceGroupTypeCSP {
		sourceGroup.ProviderName = createSourceGroupReq.ProviderName
		sourceGroup.RegionName = createSourceGroupReq.RegionName
		if err := registerCSPOnSpider(sourceGroup, createSourceGroupReq.Credential); err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		// Encrypt credential values before persisting.
		encrypted, encErr := encryptCredentialValues(sourceGroup.Credential)
		if encErr != nil {
			unregisterCSPFromSpider(sourceGroup)
			return common.ReturnErrorMsg(c, encErr.Error())
		}
		sourceGroup.Credential = encrypted
	}

	var connectionInfoList []*model.ConnectionInfo
	for i, connectionInfoCreateReq := range createSourceGroupReq.ConnectionInfo {
		connectionInfo, err := checkCreateConnectionInfoReq(sourceGroup, &connectionInfoCreateReq)
		if err != nil {
			errMsg := "Error in provided connection info (Connection Info order: " + strconv.Itoa(i+1) +
				", Error:" + err.Error() + ")"
			logger.Println(logger.ERROR, true, errMsg)

			unregisterCSPFromSpider(sourceGroup)
			return common.ReturnErrorMsg(c, errMsg)
		}

		connectionInfoList = append(connectionInfoList, connectionInfo)
	}

	sourceGroup, err = dao.SourceGroupRegister(sourceGroup)
	if err != nil {
		unregisterCSPFromSpider(sourceGroup)
		return common.ReturnErrorMsg(c, err.Error())
	}

	sourceGroupRes := model.SourceGroupRes{
		ID:                        sourceGroup.ID,
		Name:                      sourceGroup.Name,
		Description:               sourceGroup.Description,
		Type:                      sourceGroup.Type,
		ProviderName:              sourceGroup.ProviderName,
		RegionName:                sourceGroup.RegionName,
		SpiderConnectionName:      sourceGroup.SpiderConnectionName,
		ConnectionInfoStatusCount: model.ConnectionInfoStatusCount{},
	}

	var encryptedConnectionInfosLock sync.Mutex
	var errMsgLock sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(connectionInfoList))
	var errMsg string

	for _, ci := range connectionInfoList {
		go func(connectionInfo *model.ConnectionInfo) {
			defer func() {
				wg.Done()
			}()

			encryptedConnectionInfo, err := doCreateConnectionInfo(connectionInfo)
			if err != nil {
				errMsgLock.Lock()
				if errMsg != "" {
					errMsg += ", "
				}
				errMsg += "Error occurred while creating the connection info (Connection Info name: " + connectionInfo.Name +
					", Error:" + err.Error() + ")"
				errMsgLock.Unlock()
				return
			}

			encryptedConnectionInfosLock.Lock()
			sourceGroupRes.ConnectionInfoStatusCount.ConnectionInfoTotal++
			if encryptedConnectionInfo.ConnectionStatus == model.ConnectionInfoStatusSuccess {
				sourceGroupRes.ConnectionInfoStatusCount.CountConnectionSuccess++
			} else {
				sourceGroupRes.ConnectionInfoStatusCount.CountConnectionFailed++
			}
			if encryptedConnectionInfo.AgentStatus == model.ConnectionInfoStatusSuccess {
				sourceGroupRes.ConnectionInfoStatusCount.CountAgentSuccess++
			} else {
				sourceGroupRes.ConnectionInfoStatusCount.CountAgentFailed++
			}
			encryptedConnectionInfosLock.Unlock()
		}(ci)
	}

	wg.Wait()

	if errMsg != "" {
		logger.Println(logger.ERROR, true, errMsg)
		_ = doDeleteSourceGroup(sourceGroup.ID)
		return common.ReturnErrorMsg(c, errMsg)
	}

	return c.JSONPretty(http.StatusOK, sourceGroupRes, " ")
}

// GetSourceGroup godoc
//
//	@ID				get-source-group
//	@Summary		Get SourceGroup
//	@Description	Get the source group.
//	@Tags			[On-premise] SourceGroup
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Success		200	{object}	model.SourceGroupRes	"Successfully get the source group"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get the source group"
//	@Router			/source_group/{sgId} [get]
func GetSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var sourceGroupRes model.SourceGroupRes
	sourceGroupRes.ID = sourceGroup.ID
	sourceGroupRes.Name = sourceGroup.Name
	sourceGroupRes.Description = sourceGroup.Description
	sourceGroupRes.Type = sourceGroup.Type
	sourceGroupRes.ProviderName = sourceGroup.ProviderName
	sourceGroupRes.RegionName = sourceGroup.RegionName
	sourceGroupRes.SpiderConnectionName = sourceGroup.SpiderConnectionName

	connectionInfo := &model.ConnectionInfo{
		SourceGroupID: sourceGroup.ID,
	}
	connectionInfos, err := dao.ConnectionInfoGetList(connectionInfo, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	for _, ci := range *connectionInfos {
		sourceGroupRes.ConnectionInfoStatusCount.ConnectionInfoTotal++
		if ci.ConnectionStatus == model.ConnectionInfoStatusSuccess {
			sourceGroupRes.ConnectionInfoStatusCount.CountConnectionSuccess++
		} else {
			sourceGroupRes.ConnectionInfoStatusCount.CountConnectionFailed++
		}
		if ci.AgentStatus == model.ConnectionInfoStatusSuccess {
			sourceGroupRes.ConnectionInfoStatusCount.CountAgentSuccess++
		} else {
			sourceGroupRes.ConnectionInfoStatusCount.CountAgentFailed++
		}
	}

	return c.JSONPretty(http.StatusOK, sourceGroupRes, " ")
}

// ListSourceGroup godoc
//
//	@ID				list-source-group
//	@Summary		List SourceGroup
//	@Description	Get a list of source group.
//	@Tags			[On-premise] SourceGroup
//	@Accept			json
//	@Produce		json
//	@Param			page query string false "Page of the source group list."
//	@Param			row query string false "Row of the source group list."
//	@Param			name query string false "Name of the source group."
//	@Param			description query string false "Description of the source group."
//	@Success		200	{object}	[]model.ListSourceGroupRes		"Successfully get a list of source group."
//	@Failure		400	{object}	common.ErrorResponse			"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse			"Failed to get a list of source group."
//	@Router			/source_group [get]
func ListSourceGroup(c echo.Context) error {
	page, row, err := common.CheckPageRow(c)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	sourceGroup := &model.SourceGroup{
		Name:        c.QueryParam("name"),
		Description: c.QueryParam("description"),
	}

	sourceGroups, err := dao.SourceGroupGetList(sourceGroup, page, row)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var listSourceGroupRes model.ListSourceGroupRes

	for _, sg := range *sourceGroups {
		connectionInfo := &model.ConnectionInfo{
			SourceGroupID: sg.ID,
		}
		connectionInfos, err := dao.ConnectionInfoGetList(connectionInfo, 0, 0)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}

		var sourceGroupRes = model.SourceGroupRes{
			ID:                   sg.ID,
			Name:                 sg.Name,
			Description:          sg.Description,
			Type:                 sg.Type,
			ProviderName:         sg.ProviderName,
			RegionName:           sg.RegionName,
			SpiderConnectionName: sg.SpiderConnectionName,
		}

		for _, ci := range *connectionInfos {
			sourceGroupRes.ConnectionInfoStatusCount.ConnectionInfoTotal++
			listSourceGroupRes.ConnectionInfoStatusCount.ConnectionInfoTotal++

			if ci.ConnectionStatus == model.ConnectionInfoStatusSuccess {
				sourceGroupRes.ConnectionInfoStatusCount.CountConnectionSuccess++
				listSourceGroupRes.ConnectionInfoStatusCount.CountConnectionSuccess++
			} else {
				sourceGroupRes.ConnectionInfoStatusCount.CountConnectionFailed++
				listSourceGroupRes.ConnectionInfoStatusCount.CountConnectionFailed++
			}

			if ci.AgentStatus == model.ConnectionInfoStatusSuccess {
				sourceGroupRes.ConnectionInfoStatusCount.CountAgentSuccess++
				listSourceGroupRes.ConnectionInfoStatusCount.CountAgentSuccess++
			} else {
				sourceGroupRes.ConnectionInfoStatusCount.CountAgentFailed++
				listSourceGroupRes.ConnectionInfoStatusCount.CountAgentFailed++
			}
		}

		listSourceGroupRes.SourceGroup = append(listSourceGroupRes.SourceGroup, sourceGroupRes)
	}

	sort.Slice(listSourceGroupRes.SourceGroup, func(i, j int) bool {
		return strings.Compare(listSourceGroupRes.SourceGroup[i].Name, listSourceGroupRes.SourceGroup[j].Name) < 0
	})

	return c.JSONPretty(http.StatusOK, &listSourceGroupRes, " ")
}

// UpdateSourceGroup godoc
//
//	@ID				update-source-group
//	@Summary		Update SourceGroup
//	@Description	Update the source group.
//	@Tags			[On-premise] SourceGroup
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			SourceGroup body model.UpdateSourceGroupReq true	"source group to modify."
//	@Success		200	{object}	model.SourceGroup					"Successfully update the source group"
//	@Failure		400	{object}	common.ErrorResponse				"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse				"Failed to update the source group"
//	@Router			/source_group/{sgId} [put]
func UpdateSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	updateSourceGroupReq := new(model.UpdateSourceGroupReq)
	err := c.Bind(updateSourceGroupReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	oldSourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	if updateSourceGroupReq.Name != "" {
		oldSourceGroup.Name = updateSourceGroupReq.Name
	}

	if updateSourceGroupReq.Description != "" {
		oldSourceGroup.Description = updateSourceGroupReq.Description
	}

	// CSP-only: re-register region/credential on cb-spider when changed.
	if oldSourceGroup.Type == serverCommon.SourceGroupTypeCSP &&
		(updateSourceGroupReq.RegionName != "" || len(updateSourceGroupReq.Credential) > 0) {

		newRegion := updateSourceGroupReq.RegionName
		if newRegion == "" {
			newRegion = oldSourceGroup.RegionName
		}
		newKV := updateSourceGroupReq.Credential
		if len(newKV) == 0 {
			// No KV change — keep existing canonical KV (still encrypted in DB; we'd
			// need to decrypt to re-register, which we explicitly avoid). Reject the
			// region-only update path for now.
			return common.ReturnErrorMsg(c,
				"region change requires re-supplying credential (RSA-encrypted values cannot be reused)")
		}

		// Best-effort: tear down spider state, then re-register.
		unregisterCSPFromSpider(oldSourceGroup)
		oldSourceGroup.RegionName = newRegion
		if err := registerCSPOnSpider(oldSourceGroup, newKV); err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		encrypted, encErr := encryptCredentialValues(oldSourceGroup.Credential)
		if encErr != nil {
			return common.ReturnErrorMsg(c, encErr.Error())
		}
		oldSourceGroup.Credential = encrypted
	}

	err = dao.SourceGroupUpdate(oldSourceGroup)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, oldSourceGroup, " ")
}

// RegisterTargetInfoToSourceGroup godoc
//
//	@ID				register-target-to-source-group
//	@Summary		Register TargetInfo to SourceGroup
//	@Description	Register target information to the source group.
//	@Tags			[On-premise] SourceGroup
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			TargetInfo body model.RegisterTargetInfoReq true "Target info data received from infra migration via beetle."
//	@Success		200	{object}	model.SourceGroup	"Successfully update the source group"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to update the source group"
//	@Router			/source_group/{sgId}/target [post]
func RegisterTargetInfoToSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	registerTargetInfoReq := new(model.RegisterTargetInfoReq)
	err := c.Bind(registerTargetInfoReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	oldSourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	if registerTargetInfoReq.ResourceType == "mci" {
		oldSourceGroup.TargetInfo.NSID = registerTargetInfoReq.Label.SysNamespace
		oldSourceGroup.TargetInfo.MCIID = registerTargetInfoReq.ID
	}

	err = dao.SourceGroupUpdate(oldSourceGroup)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, oldSourceGroup, " ")
}

// DeleteSourceGroup godoc
//
//	@ID				delete-source-group
//	@Summary		Delete SourceGroup
//	@Description	Delete the source group.
//	@Tags			[On-premise] SourceGroup
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Success		200	{object}	model.SimpleMsg			"Successfully delete the source group"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to delete the source group"
//	@Router			/source_group/{sgId} [delete]
func DeleteSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	err := doDeleteSourceGroup(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "success"}, " ")
}

// RefreshSourceGroupConnectionInfoStatus godoc
//
//	@ID				refresh-source-group-connection-info-status
//	@Summary		Refresh SourceGroup Connection Info Status
//	@Description	Refresh connection info status of the source group.
//	@Tags			[On-premise] SourceGroup
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Success		200	{object}	model.SimpleMsg			"Successfully refresh the source group"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to refresh the source group"
//	@Router			/source_group/{sgId}/refresh [put]
func RefreshSourceGroupConnectionInfoStatus(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfo := &model.ConnectionInfo{
		SourceGroupID: sourceGroup.ID,
	}
	connectionInfos, err := dao.ConnectionInfoGetList(connectionInfo, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var errMsgLock sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(*connectionInfos))
	var errMsg string

	for _, ci := range *connectionInfos {
		go func(connectionInfo model.ConnectionInfo) {
			defer func() {
				wg.Done()
			}()

			_, err := doGetConnectionInfo(connectionInfo.ID, true)
			if err != nil {
				errMsgLock.Lock()
				if errMsg != "" {
					errMsg += ", "
				}
				errMsg += "Error occurred while refreshing the connection info (Connection Info name: " + connectionInfo.Name +
					", Error:" + err.Error() + ")"
				errMsgLock.Unlock()
			}
		}(ci)
	}

	wg.Wait()

	if errMsg != "" {
		logger.Println(logger.ERROR, true, errMsg)
		return common.ReturnErrorMsg(c, errMsg)
	}

	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "success"}, " ")
}

package controller

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/google/uuid"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
	"sync"
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

	sourceGroup := &model.SourceGroup{
		ID:          uuid.New().String(),
		Name:        createSourceGroupReq.Name,
		Description: createSourceGroupReq.Description,
	}

	var connectionInfoList []*model.ConnectionInfo
	for i, connectionInfoCreateReq := range createSourceGroupReq.ConnectionInfo {
		connectionInfo, err := checkCreateConnectionInfoReq(sourceGroup.ID, &connectionInfoCreateReq)
		if err != nil {
			errMsg := "Error in provided connection info (Connection Info order: " + strconv.Itoa(i+1) +
				", Error:" + err.Error() + ")"
			logger.Println(logger.ERROR, true, errMsg)

			return common.ReturnErrorMsg(c, errMsg)
		}

		connectionInfoList = append(connectionInfoList, connectionInfo)
	}

	sourceGroup, err = dao.SourceGroupRegister(sourceGroup)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	sourceGroupRes := model.SourceGroupRes{
		ID:                        sourceGroup.ID,
		Name:                      sourceGroup.Name,
		Description:               sourceGroup.Description,
		ConnectionInfo:            []model.ConnectionInfo{},
		ConnectionInfoStatusCount: model.ConnectionInfoStatusCount{},
	}

	for _, connectionInfo := range connectionInfoList {
		encryptedConnectionInfo, err := doCreateConnectionInfo(connectionInfo)
		if err != nil {
			errMsg := "Error occurred while creating the connection info (Connection Info name: " + connectionInfo.Name +
				", Error:" + err.Error() + ")"
			logger.Println(logger.ERROR, true, errMsg)
			_ = doDeleteSourceGroup(connectionInfo.SourceGroupID)

			return common.ReturnErrorMsg(c, errMsg)
		}

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
		sourceGroupRes.ConnectionInfo = append(sourceGroupRes.ConnectionInfo, *encryptedConnectionInfo)
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

	connectionInfo := &model.ConnectionInfo{
		SourceGroupID: sourceGroup.ID,
	}
	connectionInfos, err := dao.ConnectionInfoGetList(connectionInfo, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var encryptedConnectionInfosLock sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(*connectionInfos))

	for _, ci := range *connectionInfos {
		go func(connectionInfo model.ConnectionInfo) {
			defer func() {
				wg.Done()
			}()

			encryptedConnectionInfo, err := doGetConnectionInfo(connectionInfo.ID)
			if err != nil {
				encryptedConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusFailed
				encryptedConnectionInfo.ConnectionFailedMessage = err.Error()
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
			sourceGroupRes.ConnectionInfo = append(sourceGroupRes.ConnectionInfo, *encryptedConnectionInfo)
			encryptedConnectionInfosLock.Unlock()
		}(ci)
	}

	wg.Wait()

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
//	@Success		200	{object}	[]model.SourceGroup		"Successfully get a list of source group."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get a list of source group."
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

	SourceGroups, err := dao.SourceGroupGetList(sourceGroup, page, row)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, SourceGroups, " ")
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

	updateSourceGroupReq := new(model.CreateSourceGroupReq)
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
		if registerTargetInfoReq.ID == "" {
			oldSourceGroup.TargetInfo.NSID = ""
		} else {
			// TODO: Must be able to set NSID next time.
			oldSourceGroup.TargetInfo.NSID = "mig01"
		}
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

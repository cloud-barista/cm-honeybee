package controller

import (
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/ssh"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/google/uuid"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
	"net/http"
)

// CreateSourceGroup godoc
//
//	@ID				register-source-group
//	@Summary		Register SourceGroup
//	@Description	Register the source group.
//	@Tags			[On-premise] SourceGroup
//	@Accept			json
//	@Produce		json
//	@Param			SourceGroup body model.CreateSourceGroupReq true "source group of the node."
//	@Success		200	{object}	model.CreateSourceGroupReq	"Successfully register the source group"
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
		return common.ReturnErrorMsg(c, "Please provide the name.")
	}

	sourceGroup := &model.SourceGroup{
		ID:          uuid.New().String(),
		Name:        createSourceGroupReq.Name,
		Description: createSourceGroupReq.Description,
	}

	sourceGroup, err = dao.SourceGroupRegister(sourceGroup)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, sourceGroup, " ")
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
//	@Success		200	{object}	model.SourceGroup	"Successfully get the source group"
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

	return c.JSONPretty(http.StatusOK, sourceGroup, " ")
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
//	@Success		200	{object}	[]model.SourceGroup	"Successfully get a list of source group."
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
//	@Param			SourceGroup body model.CreateSourceGroupReq true "source group to modify."
//	@Success		200	{object}	model.SourceGroup	"Successfully update the source group"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to update the source group"
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

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfoList, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{
		SourceGroupID: sgID,
	}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, "Failed to get connection info list to delete.")
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
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "success"}, " ")
}

// CheckConnectionSourceGroup godoc
//
//	@ID				check-connection-source-group
//	@Summary		Check Connection SourceGroup
//	@Description	Check if SSH connection is available for each connection info in source group. Show each status by returning connection info list.
//	@Tags			[On-premise] SourceGroup
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Success		200	{object}	[]model.ConnectionInfo		"Successfully checked SSH connection for the source group"
//	@Failure		400	{object}	common.ErrorResponse		"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse		"Failed to check SSH connection for the source group"
//	@Router			/source_group/{sgId}/connection_check [get]
func CheckConnectionSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfoList, err := dao.SourceGroupCheckConnection(sourceGroup)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var encryptedConnectionInfos []model.ConnectionInfo
	for _, ci := range *connectionInfoList {
		encryptedConnectionInfo, err := encryptPasswordAndPrivateKey(&ci)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}

		encryptedConnectionInfos = append(encryptedConnectionInfos, *encryptedConnectionInfo)
	}

	return c.JSONPretty(http.StatusOK, encryptedConnectionInfos, " ")
}

// CheckAgentSourceGroup godoc
//
//	@ID				check-agent-source-group
//	@Summary		Check Agent SourceGroup
//	@Description	Check Agent in source group. Show each status by returning agent info list. If no Agent is present on the connected server, the Agent will be automatically installed.
//	@Tags			[On-premise] SourceGroup
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Success		200	{object}	[]model.AgentInfo		"Successfully checked Agent for the source group"
//	@Failure		400	{object}	common.ErrorResponse		"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse		"Failed to check Agent for the source group"
//	@Router			/source_group/{sgId}/agent_check [get]
func CheckAgentSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfoList, err := dao.SourceGroupCheckConnection(sourceGroup)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var agentInfos []model.AgentInfo
	for _, ci := range *connectionInfoList {

		s := &ssh.SSH{
			Options: ssh.DefaultSSHOptions(),
		}

		data, err := s.RunAgent(ci)
		agentInfo := model.AgentInfo{
			Connection: ci.Name,
			Result:     data,
			ErrorMsg:   err,
		}

		agentInfos = append(agentInfos, agentInfo)
	}

	return c.JSONPretty(http.StatusOK, agentInfos, " ")
}

package controller

import (
	agent "github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/config"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
	"time"
)

// ImportInfra godoc
//
// @Summary		Import Infra
// @Description	Import the infra information.
// @Tags		[Import] Import source info
// @Accept		json
// @Produce		json
// @Param		sgId path string true "ID of the source group."
// @Param		connId path string true "ID of the connection info."
// @Success		200	{object}	model.SavedInfraInfo	"Successfully saved the infra information"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to save the infra information"
// @Router		/honeybee/source_group/{sgId}/connection_info/{connId}/import/infra [get]
func ImportInfra(c echo.Context) error {
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

	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	oldSavedInfraInfo, _ := dao.SavedInfraInfoGet(connectionInfo.ID)

	if oldSavedInfraInfo == nil {
		savedInfraInfo := new(model.SavedInfraInfo)
		savedInfraInfo.ConnectionID = connectionInfo.ID
		savedInfraInfo.InfraData = ""
		savedInfraInfo.Status = "importing"
		savedInfraInfo.SavedTime = time.Now()
		savedInfraInfo, err = dao.SavedInfraInfoRegister(savedInfraInfo)
		if err != nil {
			return common.ReturnInternalError(c, err, "Error occurred while getting infra information.")
		}
		oldSavedInfraInfo = savedInfraInfo
	}

	data, err := common.GetHTTPRequest("http://" + connectionInfo.IPAddress + ":" + config.CMHoneybeeConfig.CMHoneybee.Agent.Port + "/" + strings.ToLower(agent.ShortModuleName) + "/infra")
	if err != nil {
		oldSavedInfraInfo.Status = "failed"
		_ = dao.SavedInfraInfoUpdate(oldSavedInfraInfo)
		return common.ReturnInternalError(c, err, "Error occurred while getting infra information.")
	}

	oldSavedInfraInfo.InfraData = string(data)
	oldSavedInfraInfo.Status = "success"
	oldSavedInfraInfo.SavedTime = time.Now()
	err = dao.SavedInfraInfoUpdate(oldSavedInfraInfo)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while saving the infra information.")
	}

	return c.JSONPretty(http.StatusOK, oldSavedInfraInfo, " ")
}

// ImportSoftware godoc
//
// @Summary		Import software
// @Description	Import the software information.
// @Tags		[Import] Import source info
// @Accept		json
// @Produce		json
// @Param		sgId path string true "ID of the source group."
// @Param		connId path string true "ID of the connection info."
// @Success		200	{object}	model.SavedSoftwareInfo	"Successfully saved the software information"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to save the software information"
// @Router		/honeybee/source_group/{sgId}/connection_info/{connId}/import/software [get]
func ImportSoftware(c echo.Context) error {
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

	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	oldSavedSoftwareInfo, _ := dao.SavedSoftwareInfoGet(connectionInfo.ID)

	if oldSavedSoftwareInfo == nil {
		savedSoftwareInfo := new(model.SavedSoftwareInfo)
		savedSoftwareInfo.ConnectionID = connectionInfo.ID
		savedSoftwareInfo.SoftwareData = ""
		savedSoftwareInfo.Status = "importing"
		savedSoftwareInfo.SavedTime = time.Now()
		savedSoftwareInfo, err = dao.SavedSoftwareInfoRegister(savedSoftwareInfo)
		if err != nil {
			return common.ReturnErrorMsg(c, "Error occurred while getting infra information.")
		}
		oldSavedSoftwareInfo = savedSoftwareInfo
	}

	data, err := common.GetHTTPRequest("http://" + connectionInfo.IPAddress + ":" + config.CMHoneybeeConfig.CMHoneybee.Agent.Port + "/" + strings.ToLower(agent.ShortModuleName) + "/software")
	if err != nil {
		oldSavedSoftwareInfo.Status = "failed"
		_ = dao.SavedSoftwareInfoUpdate(oldSavedSoftwareInfo)
		return common.ReturnErrorMsg(c, "Error occurred while getting software information.")
	}

	oldSavedSoftwareInfo.SoftwareData = string(data)
	oldSavedSoftwareInfo.Status = "success"
	oldSavedSoftwareInfo.SavedTime = time.Now()
	err = dao.SavedSoftwareInfoUpdate(oldSavedSoftwareInfo)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while saving the software information.")
	}

	return c.JSONPretty(http.StatusOK, oldSavedSoftwareInfo, " ")
}

package controller

import (
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/config"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

// ImportInfra godoc
//
// @Summary		Import Infra
// @Description	Import the infra information.
// @Tags		[Import] ImportInfra
// @Accept		json
// @Produce		json
// @Param		uuid path string true "UUID of the connectionInfo"
// @Success		200	{object}	model.SavedInfraInfo	"Successfully saved the infra information"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to save the infra information"
// @Router			/import/infra/{uuid} [get]
func ImportInfra(c echo.Context) error {
	uuid := c.Param("uuid")
	if uuid == "" {
		return common.ReturnErrorMsg(c, "uuid is empty")
	}

	connectionInfo, err := dao.ConnectionInfoGet(uuid)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	oldSavedInfraInfo, _ := dao.SavedInfraInfoGet(connectionInfo.UUID)

	if oldSavedInfraInfo == nil {
		savedInfraInfo := new(model.SavedInfraInfo)
		savedInfraInfo.ConnectionUUID = connectionInfo.UUID
		savedInfraInfo.InfraData = ""
		savedInfraInfo.Status = "importing"
		savedInfraInfo.SavedTime = time.Now()
		savedInfraInfo, err = dao.SavedInfraInfoRegister(savedInfraInfo)
		if err != nil {
			return common.ReturnInternalError(c, err, "Error occurred while getting infra information.")
		}
		oldSavedInfraInfo = savedInfraInfo
	}

	data, err := common.GetHTTPRequest("http://" + connectionInfo.IPAddress + ":" + config.CMHoneybeeConfig.CMHoneybee.Agent.Port + "/infra")
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
// @Tags		[Import] ImportSoftware
// @Accept		json
// @Produce		json
// @Param		uuid path string true "UUID of the connectionInfo"
// @Success		200	{object}	model.SavedSoftwareInfo	"Successfully saved the software information"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to save the software information"
// @Router			/import/software/{uuid} [get]
func ImportSoftware(c echo.Context) error {
	uuid := c.Param("uuid")
	if uuid == "" {
		return common.ReturnErrorMsg(c, "uuid is empty")
	}

	connectionInfo, err := dao.ConnectionInfoGet(uuid)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	oldSavedSoftwareInfo, _ := dao.SavedSoftwareInfoGet(connectionInfo.UUID)

	if oldSavedSoftwareInfo == nil {
		savedSoftwareInfo := new(model.SavedSoftwareInfo)
		savedSoftwareInfo.ConnectionUUID = connectionInfo.UUID
		savedSoftwareInfo.SoftwareData = ""
		savedSoftwareInfo.Status = "importing"
		savedSoftwareInfo.SavedTime = time.Now()
		savedSoftwareInfo, err = dao.SavedSoftwareInfoRegister(savedSoftwareInfo)
		oldSavedSoftwareInfo = savedSoftwareInfo
		if err != nil {
			return common.ReturnInternalError(c, err, "Error occurred while getting infra information.")
		}
	}

	data, err := common.GetHTTPRequest("http://" + connectionInfo.IPAddress + ":" + config.CMHoneybeeConfig.CMHoneybee.Agent.Port + "/software")
	if err != nil {
		oldSavedSoftwareInfo.Status = "failed"
		_ = dao.SavedSoftwareInfoUpdate(oldSavedSoftwareInfo)
		return common.ReturnInternalError(c, err, "Error occurred while getting software information.")
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

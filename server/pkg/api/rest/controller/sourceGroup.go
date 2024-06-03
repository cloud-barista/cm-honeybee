package controller

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
	"net/http"
)

type CreateSourceGroupReq struct {
	ID          string `gorm:"primaryKey" json:"id" validate:"required"`
	Description string `gorm:"column:description" json:"description"`
}

type UpdateSourceGroupReq struct {
	Description string `gorm:"column:description" json:"description"`
}

// CreateSourceGroup godoc
//
// @Summary		Register SourceGroup
// @Description	Register the source group.
// @Tags		[On-premise] SourceGroup
// @Accept		json
// @Produce		json
// @Param		SourceGroup body CreateSourceGroupReq true "source group of the node."
// @Success		200	{object}	model.SourceGroup	"Successfully register the source group"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to register the source group"
// @Router		/honeybee/source_group [post]
func CreateSourceGroup(c echo.Context) error {
	createSourceGroupReq := new(CreateSourceGroupReq)
	err := c.Bind(createSourceGroupReq)
	if err != nil {
		return err
	}

	if createSourceGroupReq.ID == "" {
		return errors.New("id is empty")
	}

	sourceGroup := &model.SourceGroup{
		ID:          createSourceGroupReq.ID,
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
// @Summary		Get SourceGroup
// @Description	Get the source group.
// @Tags		[On-premise] SourceGroup
// @Accept		json
// @Produce		json
// @Param		sgId path string true "ID of the SourceGroup"
// @Success		200	{object}	model.SourceGroup	"Successfully get the source group"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to get the source group"
// @Router		/honeybee/source_group/{sgId} [get]
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
// @Summary		List SourceGroup
// @Description	Get a list of source group.
// @Tags		[On-premise] SourceGroup
// @Accept		json
// @Produce		json
// @Param		page query string false "Page of the source group list."
// @Param		row query string false "Row of the source group list."
// @Param		id query string false "ID of the source group."
// @Param		description query string false "Description of the source group."
// @Success		200	{object}	[]model.SourceGroup	"Successfully get a list of source group."
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to get a list of source group."
// @Router		/honeybee/source_group [get]
func ListSourceGroup(c echo.Context) error {
	page, row, err := common.CheckPageRow(c)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	sourceGroup := &model.SourceGroup{
		ID:          c.QueryParam("id"),
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
// @Summary		Update SourceGroup
// @Description	Update the source group.
// @Tags		[On-premise] SourceGroup
// @Accept		json
// @Produce		json
// @Param		sgId path string true "ID of the SourceGroup"
// @Param		SourceGroup body UpdateSourceGroupReq true "source group to modify."
// @Success		200	{object}	model.SourceGroup	"Successfully update the source group"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to update the source group"
// @Router		/honeybee/source_group/{sgId} [put]
func UpdateSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup := new(model.SourceGroup)
	err := c.Bind(sourceGroup)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	oldSourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	if sourceGroup.Description != "" {
		oldSourceGroup.Description = sourceGroup.Description
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

// DeleteSourceGroup godoc
//
// @Summary		Delete SourceGroup
// @Description	Delete the source group.
// @Tags		[On-premise] SourceGroup
// @Accept		json
// @Produce		json
// @Param		sgId path string true "ID of the SourceGroup"
// @Success		200	{object}	model.SourceGroup	"Successfully delete the source group"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to delete the source group"
// @Router		/honeybee/source_group/{sgId} [delete]
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
		err = dao.ConnectionInfoDelete(&connectionInfo)
		if err != nil {
			logger.Println(logger.ERROR, true, err)
		}
	}

	err = dao.SourceGroupDelete(sourceGroup)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, sourceGroup, " ")
}

// CheckConnectionSourceGroup godoc
//
// @Summary		Check Connection SourceGroup
// @Description	Check if SSH connection is available for each connection info in source group. Show each status by returning connection info list.
// @Tags		[On-premise] SourceGroup
// @Accept		json
// @Produce		json
// @Param		sgId path string true "ID of the SourceGroup"
// @Success		200	{object}	[]model.ConnectionInfo		"Successfully checked SSH connection for the source group"
// @Failure		400	{object}	common.ErrorResponse		"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse		"Failed to check SSH connection for the source group"
// @Router		/honeybee/source_group/{sgId}/connection_check [get]
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

	return c.JSONPretty(http.StatusOK, connectionInfoList, " ")
}

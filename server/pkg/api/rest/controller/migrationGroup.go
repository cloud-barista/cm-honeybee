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

// CreateMigrationGroup godoc
//
// @Summary		Register MigrationGroup
// @Description	Register the migration group.
// @Tags		[On-premise] MigrationGroup
// @Accept		json
// @Produce		json
// @Param		MigrationGroup body model.MigrationGroup true "migration group of the node."
// @Success		200	{object}	model.MigrationGroup	"Successfully register the migration group"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to register the migration group"
// @Router			/migration_group [post]
func CreateMigrationGroup(c echo.Context) error {
	migrationGroup := new(model.MigrationGroup)
	err := c.Bind(migrationGroup)
	if err != nil {
		return err
	}

	if migrationGroup.Name == "" {
		return errors.New("name is empty")
	}

	migrationGroup, err = dao.MigrationGroupRegister(migrationGroup)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, migrationGroup, " ")
}

// GetMigrationGroup godoc
//
// @Summary		Get MigrationGroup
// @Description	Get the migration group.
// @Tags		[On-premise] MigrationGroup
// @Accept		json
// @Produce		json
// @Param		uuid path string true "UUID of the MigrationGroup"
// @Success		200	{object}	model.MigrationGroup	"Successfully get the migration group"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to get the migration group"
// @Router		/migration_group/{uuid} [get]
func GetMigrationGroup(c echo.Context) error {
	uuid := c.Param("uuid")
	if uuid == "" {
		return common.ReturnErrorMsg(c, "uuid is empty")
	}

	migrationGroup, err := dao.MigrationGroupGet(uuid)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, migrationGroup, " ")
}

// ListMigrationGroup godoc
//
// @Summary		List MigrationGroup
// @Description	Get a list of migration group.
// @Tags		[On-premise] MigrationGroup
// @Accept		json
// @Produce		json
// @Param		page query string false "Page of the migration group list."
// @Param		row query string false "Row of the migration group list."
// @Param		uuid query string false "UUID of the migration group."
// @Param		name query string false "Migration group name."
// @Success		200	{object}	[]model.MigrationGroup	"Successfully get a list of migration group."
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to get a list of migration group."
// @Router			/migration_group [get]
func ListMigrationGroup(c echo.Context) error {
	page, row, err := common.CheckPageRow(c)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	migrationGroup := &model.MigrationGroup{
		UUID: c.QueryParam("uuid"),
		Name: c.QueryParam("name"),
	}

	MigrationGroups, err := dao.MigrationGroupGetList(migrationGroup, page, row)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, MigrationGroups, " ")
}

// UpdateMigrationGroup godoc
//
// @Summary		Update MigrationGroup
// @Description	Update the migration group.
// @Tags		[On-premise] MigrationGroup
// @Accept		json
// @Produce		json
// @Param		MigrationGroup body model.MigrationGroup true "migration group to modify."
// @Success		200	{object}	model.MigrationGroup	"Successfully update the migration group"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to update the migration group"
// @Router		/migration_group/{uuid} [put]
func UpdateMigrationGroup(c echo.Context) error {
	migrationGroup := new(model.MigrationGroup)
	err := c.Bind(migrationGroup)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	migrationGroup.UUID = c.Param("uuid")
	oldMigrationGroup, err := dao.MigrationGroupGet(migrationGroup.UUID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	if migrationGroup.Name != "" {
		oldMigrationGroup.Name = migrationGroup.Name
	}

	err = dao.MigrationGroupUpdate(oldMigrationGroup)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, oldMigrationGroup, " ")
}

func deleteSavedInfraInfo(connectionInfo *model.ConnectionInfo) {
	savedInfraInfo, _ := dao.SavedInfraInfoGet(connectionInfo.UUID)
	if savedInfraInfo == nil {
		return
	}
	err := dao.SavedInfraInfoDelete(savedInfraInfo)
	if err != nil {
		logger.Println(logger.ERROR, true, err)
	}
}

func deleteSavedSoftwareInfo(connectionInfo *model.ConnectionInfo) {
	savedSoftwareInfo, _ := dao.SavedSoftwareInfoGet(connectionInfo.UUID)
	if savedSoftwareInfo == nil {
		return
	}
	err := dao.SavedSoftwareInfoDelete(savedSoftwareInfo)
	if err != nil {
		logger.Println(logger.ERROR, true, err)
	}
}

// DeleteMigrationGroup godoc
//
// @Summary		Delete MigrationGroup
// @Description	Delete the migration group.
// @Tags		[On-premise] MigrationGroup
// @Accept		json
// @Produce		json
// @Success		200	{object}	model.MigrationGroup	"Successfully delete the migration group"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to delete the migration group"
// @Router		/migration_group/{uuid} [delete]
func DeleteMigrationGroup(c echo.Context) error {
	uuid := c.Param("uuid")
	if uuid == "" {
		return common.ReturnErrorMsg(c, "uuid is empty")
	}

	migrationGroup, err := dao.MigrationGroupGet(uuid)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfoList, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{
		GroupUUID: uuid,
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

	err = dao.MigrationGroupDelete(migrationGroup)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, migrationGroup, " ")
}

// CheckConnectionMigrationGroup godoc
//
// @Summary		Check Connection MigrationGroup
// @Description	Check if SSH connection is available for each connection info in migration group. Show each status by returning connection info list.
// @Tags		[On-premise] MigrationGroup
// @Accept		json
// @Produce		json
// @Param		MigrationGroup body model.MigrationGroup true "migration group to check SSH connection for each connection info in migration group"
// @Success		200	{object}	[]model.ConnectionInfo		"Successfully checked SSH connection for the migration group"
// @Failure		400	{object}	common.ErrorResponse		"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse		"Failed to check SSH connection for the migration group"
// @Router		/migration_group/check/{uuid} [get]
func CheckConnectionMigrationGroup(c echo.Context) error {
	uuid := c.Param("uuid")
	if uuid == "" {
		return common.ReturnErrorMsg(c, "uuid is empty")
	}

	migrationGroup, err := dao.MigrationGroupGet(uuid)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfoList, err := dao.MigrationGroupCheckConnection(migrationGroup)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, connectionInfoList, " ")
}

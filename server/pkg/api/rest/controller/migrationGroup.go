package controller

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/dao"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/onprem"
	"github.com/labstack/echo/v4"
	"net/http"
)

// MigrationGroupRegister godoc
//
// @Summary		Register MigrationGroup
// @Description	Register the migration group.
// @Tags		[On-premise] MigrationGroup
// @Accept		json
// @Produce		json
// @Param		MigrationGroup body onprem.MigrationGroup true "migration group of the node."
// @Success		200	{object}	onprem.MigrationGroup	"Successfully register the migration group"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to register the migration group"
// @Router			/migration_group [post]
func MigrationGroupRegister(c echo.Context) error {
	migrationGroup := new(onprem.MigrationGroup)
	err := c.Bind(migrationGroup)
	if err != nil {
		return err
	}

	if migrationGroup.Name == "" {
		return errors.New("name is empty")
	}

	migrationGroup, err = dao.MigrationGroupRegister(migrationGroup)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while registering the migration group.")
	}

	return c.JSONPretty(http.StatusOK, migrationGroup, " ")
}

// MigrationGroupGet godoc
//
// @Summary		Get MigrationGroup
// @Description	Get the migration group.
// @Tags		[On-premise] MigrationGroup
// @Accept		json
// @Produce		json
// @Param		uuid path string true "UUID of the MigrationGroup"
// @Success		200	{object}	onprem.MigrationGroup	"Successfully get the migration group"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to get the migration group"
// @Router		/migration_group/{uuid} [get]
func MigrationGroupGet(c echo.Context) error {
	uuid := c.Param("uuid")
	if uuid == "" {
		return common.ReturnErrorMsg(c, "uuid is empty")
	}

	migrationGroup, err := dao.MigrationGroupGet(uuid)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while getting the migration group.")
	}

	return c.JSONPretty(http.StatusOK, migrationGroup, " ")
}

// MigrationGroupGetList godoc
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
// @Success		200	{object}	[]onprem.MigrationGroup	"Successfully get a list of migration group."
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to get a list of migration group."
// @Router			/migration_group [get]
func MigrationGroupGetList(c echo.Context) error {
	page, row, err := common.CheckPageRow(c)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	migrationGroup := &onprem.MigrationGroup{
		UUID: c.QueryParam("uuid"),
		Name: c.QueryParam("name"),
	}

	MigrationGroups, err := dao.MigrationGroupGetList(migrationGroup, page, row)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while getting the migration group list.")
	}

	return c.JSONPretty(http.StatusOK, MigrationGroups, " ")
}

// MigrationGroupUpdate godoc
//
// @Summary		Update MigrationGroup
// @Description	Update the migration group.
// @Tags		[On-premise] MigrationGroup
// @Accept		json
// @Produce		json
// @Param		MigrationGroup body onprem.MigrationGroup true "migration group to modify."
// @Success		200	{object}	onprem.MigrationGroup	"Successfully update the migration group"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to update the migration group"
// @Router		/migration_group/{uuid} [put]
func MigrationGroupUpdate(c echo.Context) error {
	migrationGroup := new(onprem.MigrationGroup)
	err := c.Bind(migrationGroup)
	if err != nil {
		return err
	}

	migrationGroup.UUID = c.Param("uuid")
	oldMigrationGroup, err := dao.MigrationGroupGet(migrationGroup.UUID)
	if err != nil {
		return err
	}

	if migrationGroup.Name != "" {
		oldMigrationGroup.Name = migrationGroup.Name
	}

	err = dao.MigrationGroupUpdate(oldMigrationGroup)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while updating the migration group.")
	}

	return c.JSONPretty(http.StatusOK, oldMigrationGroup, " ")
}

// MigrationGroupDelete godoc
//
// @Summary		Delete MigrationGroup
// @Description	Delete the migration group.
// @Tags		[On-premise] MigrationGroup
// @Accept		json
// @Produce		json
// @Success		200	{object}	onprem.MigrationGroup	"Successfully delete the migration group"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to delete the migration group"
// @Router		/migration_group/{uuid} [delete]
func MigrationGroupDelete(c echo.Context) error {
	uuid := c.Param("uuid")
	if uuid == "" {
		return common.ReturnErrorMsg(c, "uuid is empty")
	}

	migrationGroup, err := dao.MigrationGroupGet(uuid)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while getting the migration group.")
	}

	err = dao.MigrationGroupDelete(migrationGroup)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while deleting the migration group.")
	}

	return c.JSONPretty(http.StatusOK, migrationGroup, " ")
}

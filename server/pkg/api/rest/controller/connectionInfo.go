package controller

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/dao"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/onprem"
	"github.com/jollaman999/utils/iputil"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

func checkIPAddress(ipAddress string) error {
	if ipAddress == "" {
		return errors.New("ip_address is empty")
	}

	if iputil.CheckValidIP(ipAddress) == nil {
		return errors.New("ip_address is invalid")
	}

	return nil
}

func checkPort(port int) error {
	if port < 1 || port > 65535 {
		return errors.New("port value is invalid")
	}

	return nil
}

// ConnectionInfoRegister godoc
//
// @Summary		Register ConnectionInfo
// @Description	Register the connection information.
// @Tags		[On-premise] ConnectionInfo
// @Accept		json
// @Produce		json
// @Param		ConnectionInfo body onprem.ConnectionInfo true "Connection information of the node."
// @Success		200	{object}	onprem.ConnectionInfo	"Successfully register the connection information"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to register the connection information"
// @Router			/connection_info [post]
func ConnectionInfoRegister(c echo.Context) error {
	connectionInfo := new(onprem.ConnectionInfo)
	err := c.Bind(connectionInfo)
	if err != nil {
		return err
	}

	if connectionInfo.GroupUUID == "" {
		return errors.New("group_uuid is empty")
	}
	err = checkIPAddress(connectionInfo.IPAddress)
	if err != nil {
		return err
	}
	err = checkPort(connectionInfo.SSHPort)
	if err != nil {
		return err
	}
	if connectionInfo.User == "" {
		return errors.New("user is empty")
	}
	if connectionInfo.Password == "" || connectionInfo.PrivateKey == "" {
		return errors.New("password or private_key must be provided")
	}
	if connectionInfo.Type == "" {
		return errors.New("type is empty")
	}

	_, err = dao.MigrationGroupGet(connectionInfo.GroupUUID)
	if err != nil {
		return err
	}

	connectionInfo, err = dao.ConnectionInfoRegister(connectionInfo)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while registering the connection information.")
	}

	return c.JSONPretty(http.StatusOK, connectionInfo, " ")
}

// ConnectionInfoGet godoc
//
// @Summary		Get ConnectionInfo
// @Description	Get the connection information.
// @Tags		[On-premise] ConnectionInfo
// @Accept		json
// @Produce		json
// @Param		uuid path string true "UUID of the connectionInfo"
// @Success		200	{object}	onprem.ConnectionInfo	"Successfully get the connection information"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to get the connection information"
// @Router		/connection_info/{uuid} [get]
func ConnectionInfoGet(c echo.Context) error {
	uuid := c.Param("uuid")
	if uuid == "" {
		return common.ReturnErrorMsg(c, "uuid is empty")
	}

	connectionInfo, err := dao.ConnectionInfoGet(uuid)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while getting the connection information.")
	}

	return c.JSONPretty(http.StatusOK, connectionInfo, " ")
}

// ConnectionInfoGetList godoc
//
// @Summary		List ConnectionInfo
// @Description	Get a list of connection information.
// @Tags		[On-premise] ConnectionInfo
// @Accept		json
// @Produce		json
// @Param		page query string false "Page of the connection information list."
// @Param		row query string false "Row of the connection information list."
// @Param		uuid query string false "UUID of the connection information."
// @Param		group_uuid query string false "Migration group UUID."
// @Param		ip_address query string false "IP address of the connection information."
// @Param		ssh_port query string false "SSH port of the connection information."
// @Param		user query string false "User of the connection information."
// @Param		type query string false "Type of the connection information."
// @Success		200	{object}	[]onprem.ConnectionInfo	"Successfully get a list of connection information."
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to get a list of connection information."
// @Router			/connection_info [get]
func ConnectionInfoGetList(c echo.Context) error {
	page, row, err := common.CheckPageRow(c)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	sshPort, _ := strconv.Atoi(c.QueryParam("ssh_port"))

	connectionInfo := &onprem.ConnectionInfo{
		UUID:      c.QueryParam("uuid"),
		GroupUUID: c.QueryParam("group_uuid"),
		IPAddress: c.QueryParam("ip_address"),
		SSHPort:   sshPort,
		User:      c.QueryParam("user"),
		Type:      c.QueryParam("type"),
	}

	connectionInfos, err := dao.ConnectionInfoGetList(connectionInfo, page, row)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while getting the connectionInfo list.")
	}

	return c.JSONPretty(http.StatusOK, connectionInfos, " ")
}

// ConnectionInfoUpdate godoc
//
// @Summary		Update ConnectionInfo
// @Description	Update the connection information.
// @Tags		[On-premise] ConnectionInfo
// @Accept		json
// @Produce		json
// @Param		ConnectionInfo body onprem.ConnectionInfo true "Connection information to modify."
// @Success		200	{object}	onprem.ConnectionInfo	"Successfully update the connection information"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to update the connection information"
// @Router		/connection_info/{uuid} [put]
func ConnectionInfoUpdate(c echo.Context) error {
	connectionInfo := new(onprem.ConnectionInfo)
	err := c.Bind(connectionInfo)
	if err != nil {
		return err
	}

	connectionInfo.UUID = c.Param("uuid")
	oldConnectionInfo, err := dao.ConnectionInfoGet(connectionInfo.UUID)
	if err != nil {
		return err
	}

	_, err = dao.MigrationGroupGet(connectionInfo.GroupUUID)
	if err != nil {
		return err
	}

	if connectionInfo.GroupUUID != "" {
		oldConnectionInfo.GroupUUID = connectionInfo.GroupUUID
	}
	err = checkIPAddress(connectionInfo.IPAddress)
	if err == nil {
		oldConnectionInfo.IPAddress = connectionInfo.IPAddress
	}
	err = checkPort(connectionInfo.SSHPort)
	if err == nil {
		oldConnectionInfo.SSHPort = connectionInfo.SSHPort
	}
	if connectionInfo.User != "" {
		oldConnectionInfo.User = connectionInfo.User
	}
	if connectionInfo.Password == "" {
		oldConnectionInfo.Password = connectionInfo.Password
	}
	if connectionInfo.PrivateKey == "" {
		oldConnectionInfo.PrivateKey = connectionInfo.PrivateKey
	}
	if connectionInfo.Type == "" {
		oldConnectionInfo.Type = connectionInfo.Type
	}

	err = dao.ConnectionInfoUpdate(oldConnectionInfo)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while updating the connection information.")
	}

	return c.JSONPretty(http.StatusOK, oldConnectionInfo, " ")
}

// ConnectionInfoDelete godoc
//
// @Summary		Delete ConnectionInfo
// @Description	Delete the connection information.
// @Tags		[On-premise] ConnectionInfo
// @Accept		json
// @Produce		json
// @Success		200	{object}	onprem.ConnectionInfo	"Successfully delete the connection information"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to delete the connection information"
// @Router		/connection_info/{uuid} [delete]
func ConnectionInfoDelete(c echo.Context) error {
	uuid := c.Param("uuid")
	if uuid == "" {
		return common.ReturnErrorMsg(c, "uuid is empty")
	}

	connectionInfo, err := dao.ConnectionInfoGet(uuid)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while getting the connection information.")
	}

	err = dao.ConnectionInfoDelete(connectionInfo)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while deleting the connection information.")
	}

	return c.JSONPretty(http.StatusOK, connectionInfo, " ")
}

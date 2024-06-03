package controller

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/iputil"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

type CreateConnectionInfoReq struct {
	ID          string `gorm:"primaryKey" json:"id" validate:"required"`
	Description string `gorm:"column:description" json:"description"`
	IPAddress   string `gorm:"column:ip_address" json:"ip_address" validate:"required"`
	SSHPort     int    `gorm:"column:ssh_port" json:"ssh_port" validate:"required"`
	User        string `gorm:"column:user" json:"user" validate:"required"`
	Password    string `gorm:"column:password" json:"password"`
	PrivateKey  string `gorm:"column:private_key" json:"private_key"`
}

type UpdateConnectionInfoReq struct {
	Description string `gorm:"column:description" json:"description"`
	IPAddress   string `gorm:"column:ip_address" json:"ip_address" validate:"required"`
	SSHPort     int    `gorm:"column:ssh_port" json:"ssh_port" validate:"required"`
	User        string `gorm:"column:user" json:"user" validate:"required"`
	Password    string `gorm:"column:password" json:"password"`
	PrivateKey  string `gorm:"column:private_key" json:"private_key"`
}

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

// CreateConnectionInfo godoc
//
// @Summary		Create ConnectionInfo
// @Description	Create the connection information.
// @Tags		[On-premise] ConnectionInfo
// @Accept		json
// @Produce		json
// @Param		sgId path string true "ID of the SourceGroup"
// @Param		ConnectionInfo body model.ConnectionInfo true "Connection information of the node."
// @Success		200	{object}	model.ConnectionInfo	"Successfully register the connection information"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to register the connection information"
// @Router		/honeybee/source_group/{sgId}/connection_info [post]
func CreateConnectionInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	createConnectionInfoReq := new(CreateConnectionInfoReq)
	err = c.Bind(createConnectionInfoReq)
	if err != nil {
		return err
	}

	connectionInfo := &model.ConnectionInfo{
		ID:            createConnectionInfoReq.ID,
		SourceGroupID: sourceGroup.ID,
		IPAddress:     createConnectionInfoReq.IPAddress,
		SSHPort:       createConnectionInfoReq.SSHPort,
		User:          createConnectionInfoReq.User,
		Password:      createConnectionInfoReq.Password,
		PrivateKey:    createConnectionInfoReq.PrivateKey,
	}

	if connectionInfo.ID == "" {
		return common.ReturnErrorMsg(c, "id is empty")
	}
	err = checkIPAddress(connectionInfo.IPAddress)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	err = checkPort(connectionInfo.SSHPort)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if connectionInfo.User == "" {
		return common.ReturnErrorMsg(c, "user is empty")
	}
	if connectionInfo.Password == "" && connectionInfo.PrivateKey == "" {
		return common.ReturnErrorMsg(c, "password or private_key must be provided")
	}

	_, err = dao.SourceGroupGet(connectionInfo.SourceGroupID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfo, err = dao.ConnectionInfoRegister(connectionInfo)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, connectionInfo, " ")
}

// GetConnectionInfo godoc
//
// @Summary		Get ConnectionInfo
// @Description	Get the connection information.
// @Tags		[On-premise] ConnectionInfo
// @Accept		json
// @Produce		json
// @Param		sgId path string true "ID of the SourceGroup"
// @Param		connId path string true "ID of the connectionInfo"
// @Success		200	{object}	model.ConnectionInfo	"Successfully get the connection information"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to get the connection information"
// @Router		/honeybee/source_group/{sgId}/connection_info/{connId} [get]
func GetConnectionInfo(c echo.Context) error {
	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, connectionInfo, " ")
}

// ListConnectionInfo godoc
//
// @Summary		List ConnectionInfo
// @Description	Get a list of connection information.
// @Tags		[On-premise] ConnectionInfo
// @Accept		json
// @Produce		json
// @Param		sgId path string true "ID of the SourceGroup"
// @Param		page query string false "Page of the connection information list."
// @Param		row query string false "Row of the connection information list."
// @Param		id query string false "ID of the connection information."
// @Param		description query string false "Description of the connection information."
// @Param		source_group_id query string false "Source group ID."
// @Param		ip_address query string false "IP address of the connection information."
// @Param		ssh_port query string false "SSH port of the connection information."
// @Param		user query string false "User of the connection information."
// @Success		200	{object}	[]model.ConnectionInfo	"Successfully get a list of connection information."
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to get a list of connection information."
// @Router		/honeybee/source_group/{sgId}/connection_info [get]
func ListConnectionInfo(c echo.Context) error {
	page, row, err := common.CheckPageRow(c)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	sshPort, _ := strconv.Atoi(c.QueryParam("ssh_port"))

	connectionInfo := &model.ConnectionInfo{
		ID:            c.QueryParam("id"),
		Description:   c.QueryParam("description"),
		SourceGroupID: c.QueryParam("source_group_id"),
		IPAddress:     c.QueryParam("ip_address"),
		SSHPort:       sshPort,
		User:          c.QueryParam("user"),
	}

	connectionInfos, err := dao.ConnectionInfoGetList(connectionInfo, page, row)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, connectionInfos, " ")
}

// UpdateConnectionInfo godoc
//
// @Summary		Update ConnectionInfo
// @Description	Update the connection information.
// @Tags		[On-premise] ConnectionInfo
// @Accept		json
// @Produce		json
// @Param		sgId path string true "ID of the SourceGroup"
// @Param		connId path string true "ID of the connectionInfo"
// @Param		ConnectionInfo body model.ConnectionInfo true "Connection information to modify."
// @Success		200	{object}	model.ConnectionInfo	"Successfully update the connection information"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to update the connection information"
// @Router		/honeybee/source_group/{sgId}/connection_info/{connId} [put]
func UpdateConnectionInfo(c echo.Context) error {
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

	oldConnectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	updateConnectionInfoReq := new(UpdateConnectionInfoReq)
	err = c.Bind(updateConnectionInfoReq)
	if err != nil {
		return err
	}

	err = checkIPAddress(updateConnectionInfoReq.IPAddress)
	if err == nil {
		oldConnectionInfo.IPAddress = updateConnectionInfoReq.IPAddress
	}
	err = checkPort(updateConnectionInfoReq.SSHPort)
	if err == nil {
		oldConnectionInfo.SSHPort = updateConnectionInfoReq.SSHPort
	}
	if updateConnectionInfoReq.User != "" {
		oldConnectionInfo.User = updateConnectionInfoReq.User
	}
	if updateConnectionInfoReq.Password != "" {
		oldConnectionInfo.Password = updateConnectionInfoReq.Password
	}
	if updateConnectionInfoReq.PrivateKey != "" {
		oldConnectionInfo.PrivateKey = updateConnectionInfoReq.PrivateKey
	}

	err = dao.ConnectionInfoUpdate(oldConnectionInfo)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, oldConnectionInfo, " ")
}

// DeleteConnectionInfo godoc
//
// @Summary		Delete ConnectionInfo
// @Description	Delete the connection information.
// @Tags		[On-premise] ConnectionInfo
// @Accept		json
// @Produce		json
// @Param		sgId path string true "ID of the SourceGroup"
// @Param		connId path string true "ID of the connectionInfo"
// @Success		200	{object}	model.ConnectionInfo	"Successfully delete the connection information"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to delete the connection information"
// @Router		/honeybee/source_group/{sgId}/connection_info/{connId} [delete]
func DeleteConnectionInfo(c echo.Context) error {
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

	err = dao.ConnectionInfoDelete(connectionInfo)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, connectionInfo, " ")
}

package controller

import (
	"encoding/base64"
	"errors"
	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/rsautil"
	"github.com/cloud-barista/cm-honeybee/server/lib/ssh"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/google/uuid"
	"github.com/jollaman999/utils/iputil"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
	"net/http"
	"sort"
	"strconv"
	"strings"
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

func checkPort(port string) error {
	portInt, err := strconv.Atoi(port)
	if err != nil || portInt < 1 || portInt > 65535 {
		return errors.New("port value is invalid")
	}

	return nil
}

func encryptSecrets(connectionInfo *model.ConnectionInfo) (*model.ConnectionInfo, error) {
	rsaEncryptedSSHPort, err := rsautil.EncryptWithPublicKey([]byte(connectionInfo.SSHPort), serverCommon.PubKey)
	if err != nil {
		errMsg := "error occurred while encrypting the ssh port (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}
	base64EncodedEncryptedSSHPort := base64.StdEncoding.EncodeToString(rsaEncryptedSSHPort)
	connectionInfo.SSHPort = base64EncodedEncryptedSSHPort

	rsaEncryptedUser, err := rsautil.EncryptWithPublicKey([]byte(connectionInfo.User), serverCommon.PubKey)
	if err != nil {
		errMsg := "error occurred while encrypting the user (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}
	base64EncodedEncryptedUser := base64.StdEncoding.EncodeToString(rsaEncryptedUser)
	connectionInfo.User = base64EncodedEncryptedUser

	rsaEncryptedPasswordBytes, err := rsautil.EncryptWithPublicKey([]byte(connectionInfo.Password), serverCommon.PubKey)
	if err != nil {
		errMsg := "error occurred while encrypting the password (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}
	base64EncodedEncryptedPassword := base64.StdEncoding.EncodeToString(rsaEncryptedPasswordBytes)
	connectionInfo.Password = base64EncodedEncryptedPassword

	rsaEncryptedPrivateKeyBytes, err := rsautil.EncryptWithPublicKey([]byte(connectionInfo.PrivateKey), serverCommon.PubKey)
	if err != nil {
		errMsg := "error occurred while encrypting the private key (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}
	base64EncodedEncryptedPrivateKey := base64.StdEncoding.EncodeToString(rsaEncryptedPrivateKeyBytes)
	connectionInfo.PrivateKey = base64EncodedEncryptedPrivateKey

	return connectionInfo, nil
}

func checkCreateConnectionInfoReq(sourceGroupID string, createConnectionInfoReq *model.CreateConnectionInfoReq) (*model.ConnectionInfo, error) {
	if sourceGroupID == "" {
		return nil, errors.New("source group ID is empty")
	}

	connectionInfo := &model.ConnectionInfo{
		ID:            uuid.New().String(),
		Name:          createConnectionInfoReq.Name,
		Description:   createConnectionInfoReq.Description,
		SourceGroupID: sourceGroupID,
		IPAddress:     createConnectionInfoReq.IPAddress,
		SSHPort:       createConnectionInfoReq.SSHPort,
		User:          createConnectionInfoReq.User,
		Password:      createConnectionInfoReq.Password,
		PrivateKey:    createConnectionInfoReq.PrivateKey,
	}

	if connectionInfo.ID == "" {
		return nil, errors.New("id is empty")
	}

	if connectionInfo.Name == "" {
		return nil, errors.New("name is empty")
	}

	err := checkIPAddress(connectionInfo.IPAddress)
	if err != nil {
		return nil, err
	}

	err = checkPort(connectionInfo.SSHPort)
	if err != nil {
		return nil, err
	}

	if connectionInfo.User == "" {
		return nil, errors.New("user is empty")
	}

	if connectionInfo.Password == "" && connectionInfo.PrivateKey == "" {
		return nil, errors.New("password or private_key must be provided")
	}

	if connectionInfo.PrivateKey == "" {
		connectionInfo.PrivateKey = "-"
	}

	return connectionInfo, nil
}

func doGetConnectionInfo(connID string) (*model.ConnectionInfo, error) {
	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, err
	}

	oldConnectionInfo, err := dao.ConnectionInfoGet(connectionInfo.ID)
	if err != nil {
		return nil, err
	}

	c := &ssh.SSH{
		Options: ssh.DefaultSSHOptions(),
	}

	err = c.NewClientConn(*connectionInfo)
	if err != nil {
		oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusFailed
		oldConnectionInfo.ConnectionFailedMessage = err.Error()
	} else {
		c.Close()
		oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusSuccess
		oldConnectionInfo.ConnectionFailedMessage = ""
	}

	err = c.RunAgent(*connectionInfo)
	if err != nil {
		oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusFailed
		oldConnectionInfo.AgentFailedMessage = err.Error()
	} else {
		c.Close()
		oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusSuccess
		oldConnectionInfo.AgentFailedMessage = ""
	}

	err = dao.ConnectionInfoUpdate(oldConnectionInfo)
	if err != nil {
		return nil, errors.New("Error occurred while updating the connection information. " +
			"(ID: " + oldConnectionInfo.ID + ", Error: " + err.Error() + ")")
	}

	connectionInfo, err = encryptSecrets(oldConnectionInfo)
	if err != nil {
		return nil, err
	}

	return connectionInfo, nil
}

func doCreateConnectionInfo(connectionInfo *model.ConnectionInfo) (*model.ConnectionInfo, error) {
	_, err := dao.SourceGroupGet(connectionInfo.SourceGroupID)
	if err != nil {
		return nil, err
	}

	connectionInfo, err = dao.ConnectionInfoRegister(connectionInfo)
	if err != nil {
		return nil, err
	}

	connectionInfo, err = doGetConnectionInfo(connectionInfo.ID)
	if err != nil {
		return nil, err
	}

	return connectionInfo, nil
}

// CreateConnectionInfo godoc
//
//	@ID				create-connection-info
//	@Summary		Create ConnectionInfo
//	@Description	Create the connection information.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			ConnectionInfo body model.CreateConnectionInfoReq true "Connection information of the node."
//	@Success		200	{object}	model.ConnectionInfo	"Successfully register the connection information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to register the connection information"
//	@Router			/source_group/{sgId}/connection_info [post]
func CreateConnectionInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	createConnectionInfoReq := new(model.CreateConnectionInfoReq)
	err = c.Bind(createConnectionInfoReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfo, err := checkCreateConnectionInfoReq(sourceGroup.ID, createConnectionInfoReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	listOption := &model.ConnectionInfo{
		SourceGroupID: sourceGroup.ID,
	}
	connectionInfos, err := dao.ConnectionInfoGetList(listOption, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if len(*connectionInfos) >= model.ConnectionInfoMaxLength {
		return common.ReturnErrorMsg(c, "Maximum number of connection info is exceeded."+
			" (Max: "+strconv.Itoa(model.ConnectionInfoMaxLength)+")")
	}

	connectionInfo, err = doCreateConnectionInfo(connectionInfo)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, connectionInfo, " ")
}

// GetConnectionInfo godoc
//
//	@ID				get-connection-info
//	@Summary		Get ConnectionInfo
//	@Description	Get the connection information.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			connId path string true "ID of the connectionInfo"
//	@Success		200	{object}	model.ConnectionInfo	"Successfully get the connection information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get the connection information"
//	@Router			/source_group/{sgId}/connection_info/{connId} [get]
func GetConnectionInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	connectionInfo, err := doGetConnectionInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, connectionInfo, " ")
}

// GetConnectionInfoDirectly godoc
//
//	@ID				get-connection-info-directly
//	@Summary		Get ConnectionInfo Directly
//	@Description	Get the connection information directly.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			connId path string true "ID of the connectionInfo"
//	@Success		200	{object}	model.ConnectionInfo	"Successfully get the connection information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get the connection information"
//	@Router			/connection_info/{connId} [get]
func GetConnectionInfoDirectly(c echo.Context) error {
	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	connectionInfo, err := doGetConnectionInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, connectionInfo, " ")
}

// ListConnectionInfo godoc
//
//	@ID				list-connection-info
//	@Summary		List ConnectionInfo
//	@Description	Get a list of connection information.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			page query string false "Page of the connection information list."
//	@Param			row query string false "Row of the connection information list."
//	@Param			name query string false "Name of the connection information."
//	@Param			description query string false "Description of the connection information."
//	@Param			ip_address query string false "IP address of the connection information."
//	@Param			ssh_port query string false "SSH port of the connection information."
//	@Param			user query string false "User of the connection information."
//	@Success		200	{object}	[]model.ConnectionInfo	"Successfully get a list of connection information."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get a list of connection information."
//	@Router			/source_group/{sgId}/connection_info [get]
func ListConnectionInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	page, row, err := common.CheckPageRow(c)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfo := &model.ConnectionInfo{
		Name:          c.QueryParam("name"),
		Description:   c.QueryParam("description"),
		SourceGroupID: sourceGroup.ID,
		IPAddress:     c.QueryParam("ip_address"),
		SSHPort:       c.QueryParam("ssh_port"),
		User:          c.QueryParam("user"),
	}

	connectionInfos, err := dao.ConnectionInfoGetList(connectionInfo, page, row)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var encryptedConnectionInfos []model.ConnectionInfo

	for _, ci := range *connectionInfos {
		encryptedConnectionInfo, err := encryptSecrets(&ci)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}

		encryptedConnectionInfos = append(encryptedConnectionInfos, *encryptedConnectionInfo)
	}

	sort.Slice(encryptedConnectionInfos, func(i, j int) bool {
		return strings.Compare(encryptedConnectionInfos[i].Name, encryptedConnectionInfos[j].Name) < 0
	})

	return c.JSONPretty(http.StatusOK, &encryptedConnectionInfos, " ")
}

// UpdateConnectionInfo godoc
//
//	@ID				update-connection-info
//	@Summary		Update ConnectionInfo
//	@Description	Update the connection information.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			connId path string true "ID of the connectionInfo"
//	@Param			ConnectionInfo body model.CreateConnectionInfoReq true "Connection information to modify."
//	@Success		200	{object}	model.ConnectionInfo	"Successfully update the connection information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to update the connection information"
//	@Router			/source_group/{sgId}/connection_info/{connId} [put]
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

	updateConnectionInfoReq := new(model.CreateConnectionInfoReq)
	err = c.Bind(updateConnectionInfoReq)
	if err != nil {
		return err
	}

	if updateConnectionInfoReq.Description != "" {
		oldConnectionInfo.Description = updateConnectionInfoReq.Description
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

	connectionInfo, err := doGetConnectionInfo(oldConnectionInfo.ID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, connectionInfo, " ")
}

// DeleteConnectionInfo godoc
//
//	@ID				delete-connection-info
//	@Summary		Delete ConnectionInfo
//	@Description	Delete the connection information.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			connId path string true "ID of the connectionInfo"
//	@Success		200	{object}	model.SimpleMsg			"Successfully delete the connection information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to delete the connection information"
//	@Router			/source_group/{sgId}/connection_info/{connId} [delete]
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

	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "success"}, " ")
}

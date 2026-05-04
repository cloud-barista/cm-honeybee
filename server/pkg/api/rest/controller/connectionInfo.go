package controller

import (
	"encoding/base64"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

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

func encryptField(plaintext string, label string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	enc, err := rsautil.EncryptWithPublicKey([]byte(plaintext), serverCommon.PubKey)
	if err != nil {
		errMsg := "error occurred while encrypting the " + label + " (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)
		return "", errors.New(errMsg)
	}
	return base64.StdEncoding.EncodeToString(enc), nil
}

func encryptSecrets(connectionInfo *model.ConnectionInfo) (*model.ConnectionInfo, error) {
	user, err := encryptField(connectionInfo.User, "user")
	if err != nil {
		return nil, err
	}
	connectionInfo.User = user

	password, err := encryptField(connectionInfo.Password, "password")
	if err != nil {
		return nil, err
	}
	connectionInfo.Password = password

	privateKey, err := encryptField(connectionInfo.PrivateKey, "private key")
	if err != nil {
		return nil, err
	}
	connectionInfo.PrivateKey = privateKey

	return connectionInfo, nil
}

func checkCreateConnectionInfoReq(sourceGroup *model.SourceGroup, createConnectionInfoReq *model.CreateConnectionInfoReq) (*model.ConnectionInfo, error) {
	if sourceGroup == nil || sourceGroup.ID == "" {
		return nil, errors.New("source group is missing")
	}

	connectionInfo := &model.ConnectionInfo{
		ID:            uuid.New().String(),
		Name:          createConnectionInfoReq.Name,
		Description:   createConnectionInfoReq.Description,
		SourceGroupID: sourceGroup.ID,
	}

	if connectionInfo.Name == "" {
		return nil, errors.New("name is empty")
	}

	switch sourceGroup.Type {
	case "", serverCommon.SourceGroupTypeSSH:
		connectionInfo.IPAddress = createConnectionInfoReq.IPAddress
		connectionInfo.SSHPort = createConnectionInfoReq.SSHPort
		connectionInfo.User = createConnectionInfoReq.User
		connectionInfo.Password = createConnectionInfoReq.Password
		connectionInfo.PrivateKey = createConnectionInfoReq.PrivateKey

		if err := checkIPAddress(connectionInfo.IPAddress); err != nil {
			return nil, err
		}
		if err := checkPort(connectionInfo.SSHPort); err != nil {
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
	case serverCommon.SourceGroupTypeCSP:
		connectionInfo.ResourceType = strings.ToLower(strings.TrimSpace(createConnectionInfoReq.ResourceType))
		connectionInfo.ResourceID = strings.TrimSpace(createConnectionInfoReq.ResourceID)
		switch connectionInfo.ResourceType {
		case serverCommon.ResourceTypeVM,
			serverCommon.ResourceTypeK8s,
			serverCommon.ResourceTypeObjectStorage:
		default:
			return nil, errors.New("resource_type must be one of vm | k8s | object_storage")
		}
		if connectionInfo.ResourceID == "" {
			return nil, errors.New("resource_id is empty")
		}
	default:
		return nil, errors.New("unsupported source group type: " + sourceGroup.Type)
	}

	return connectionInfo, nil
}

func doGetConnectionInfo(connID string, refresh bool) (*model.ConnectionInfo, error) {
	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, err
	}

	oldConnectionInfo, err := dao.ConnectionInfoGet(connectionInfo.ID)
	if err != nil {
		return nil, err
	}

	if refresh {
		sourceGroup, err := dao.SourceGroupGet(connectionInfo.SourceGroupID)
		if err != nil {
			return nil, err
		}

		switch sourceGroup.Type {
		case serverCommon.SourceGroupTypeCSP:
			if err := refreshCSPConnection(sourceGroup, connectionInfo); err != nil {
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.ConnectionFailedMessage = err.Error()
				oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.AgentFailedMessage = err.Error()
			} else {
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusSuccess
				oldConnectionInfo.ConnectionFailedMessage = ""
				oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusSuccess
				oldConnectionInfo.AgentFailedMessage = ""
			}
		default:
			c := &ssh.SSH{}

			if err := c.NewClientConn(*connectionInfo); err != nil {
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.ConnectionFailedMessage = err.Error()
			} else {
				c.Close()
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusSuccess
				oldConnectionInfo.ConnectionFailedMessage = ""
			}

			if err := c.RunAgent(*connectionInfo); err != nil {
				oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.AgentFailedMessage = err.Error()
			} else {
				c.Close()
				oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusSuccess
				oldConnectionInfo.AgentFailedMessage = ""
			}
		}

		err = dao.ConnectionInfoUpdateWithSelect(oldConnectionInfo, []string{
			"connection_status",
			"connection_failed_message",
			"agent_status",
			"agent_failed_message",
		})
		if err != nil {
			return nil, errors.New("Error occurred while updating the connection information. " +
				"(ID: " + oldConnectionInfo.ID + ", Error: " + err.Error() + ")")
		}
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

	connectionInfo, err = doGetConnectionInfo(connectionInfo.ID, true)
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

	connectionInfo, err := checkCreateConnectionInfoReq(sourceGroup, createConnectionInfoReq)
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

	connectionInfo, err := doGetConnectionInfo(connID, false)
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

	connectionInfo, err := doGetConnectionInfo(connID, false)
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
//	@Success		200	{object}	[]model.ListConnectionInfoRes	"Successfully get a list of connection information."
//	@Failure		400	{object}	common.ErrorResponse			"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse			"Failed to get a list of connection information."
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

	var listConnectionInfoRes model.ListConnectionInfoRes
	var encryptedConnectionInfos []model.ConnectionInfo

	for _, ci := range *connectionInfos {
		listConnectionInfoRes.ConnectionInfoStatusCount.ConnectionInfoTotal++
		if ci.ConnectionStatus == model.ConnectionInfoStatusSuccess {
			listConnectionInfoRes.ConnectionInfoStatusCount.CountConnectionSuccess++
		} else {
			listConnectionInfoRes.ConnectionInfoStatusCount.CountConnectionFailed++
		}
		if ci.AgentStatus == model.ConnectionInfoStatusSuccess {
			listConnectionInfoRes.ConnectionInfoStatusCount.CountAgentSuccess++
		} else {
			listConnectionInfoRes.ConnectionInfoStatusCount.CountAgentFailed++
		}

		encryptedConnectionInfo, err := encryptSecrets(&ci)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}

		encryptedConnectionInfos = append(encryptedConnectionInfos, *encryptedConnectionInfo)
	}

	sort.Slice(encryptedConnectionInfos, func(i, j int) bool {
		return strings.Compare(encryptedConnectionInfos[i].Name, encryptedConnectionInfos[j].Name) < 0
	})

	listConnectionInfoRes.ConnectionInfo = encryptedConnectionInfos

	return c.JSONPretty(http.StatusOK, &listConnectionInfoRes, " ")
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

	sourceGroup, err := dao.SourceGroupGet(sgID)
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

	if updateConnectionInfoReq.Name != "" {
		oldConnectionInfo.Name = updateConnectionInfoReq.Name
	}
	if updateConnectionInfoReq.Description != "" {
		oldConnectionInfo.Description = updateConnectionInfoReq.Description
	}

	switch sourceGroup.Type {
	case serverCommon.SourceGroupTypeCSP:
		if updateConnectionInfoReq.ResourceType != "" {
			rt := strings.ToLower(strings.TrimSpace(updateConnectionInfoReq.ResourceType))
			switch rt {
			case serverCommon.ResourceTypeVM,
				serverCommon.ResourceTypeK8s,
				serverCommon.ResourceTypeObjectStorage:
				oldConnectionInfo.ResourceType = rt
			default:
				return common.ReturnErrorMsg(c, "resource_type must be one of vm | k8s | object_storage")
			}
		}
		if updateConnectionInfoReq.ResourceID != "" {
			oldConnectionInfo.ResourceID = strings.TrimSpace(updateConnectionInfoReq.ResourceID)
		}
	default:
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
	}

	err = dao.ConnectionInfoUpdate(oldConnectionInfo)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfo, err := doGetConnectionInfo(oldConnectionInfo.ID, true)
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

// RefreshConnectionInfoStatus godoc
//
//	@ID				refresh-connection-info-status
//	@Summary		Refresh Connection Info Status
//	@Description	Refresh the connection info status.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			connId path string true "ID of the connectionInfo"
//	@Success		200	{object}	model.SimpleMsg			"Successfully refresh the source group"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to refresh the source group"
//	@Router			/source_group/{sgId}/connection_info/{connId}/refresh [put]
func RefreshConnectionInfoStatus(c echo.Context) error {
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

	_, err = doGetConnectionInfo(connID, true)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "success"}, " ")
}

// RefreshConnectionInfoStatusDirectly godoc
//
//	@ID				refresh-connection-info-status-directly
//	@Summary		Refresh Connection Info Status Directly
//	@Description	Refresh the connection info status directly.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			connId path string true "ID of the connectionInfo"
//	@Success		200	{object}	model.SimpleMsg			"Successfully refresh the source group"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to refresh the source group"
//	@Router			/connection_info/{connId}/refresh [put]
func RefreshConnectionInfoStatusDirectly(c echo.Context) error {
	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, err := doGetConnectionInfo(connID, true)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "success"}, " ")
}

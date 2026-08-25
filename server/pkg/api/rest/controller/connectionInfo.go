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
	"github.com/cloud-barista/cm-honeybee/server/lib/openbao"
	"github.com/cloud-barista/cm-honeybee/server/lib/rsautil"
	"github.com/cloud-barista/cm-honeybee/server/lib/ssh"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/google/uuid"
	"github.com/jollaman999/utils/iputil"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
)

// validateKubeconfig checks that s is a non-empty, structurally valid kubeconfig.
func validateKubeconfig(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("kubeconfig is empty")
	}
	var kc struct {
		Kind     string        `yaml:"kind"`
		Clusters []interface{} `yaml:"clusters"`
	}
	if err := yaml.Unmarshal([]byte(s), &kc); err != nil {
		return errors.New("kubeconfig is not valid YAML (" + err.Error() + ")")
	}
	if kc.Kind != "" && kc.Kind != "Config" {
		return errors.New("kubeconfig kind must be 'Config'")
	}
	if len(kc.Clusters) == 0 {
		return errors.New("kubeconfig has no clusters")
	}
	return nil
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

	kubeconfig, err := encryptField(connectionInfo.Kubeconfig, "kubeconfig")
	if err != nil {
		return nil, err
	}
	connectionInfo.Kubeconfig = kubeconfig

	return connectionInfo, nil
}

// sshSecretPath is the OpenBao KV path for a connection's SSH secrets.
func sshSecretPath(connID string) string { return "honeybee/ssh/" + connID }

// storeConnectionSecrets moves the connection secrets (SSH password/private key,
// or kubeconfig) into OpenBao when enabled, clearing them from ci so the DB row
// holds no secrets. Errors when there are secrets but OpenBao is off.
func storeConnectionSecrets(ci *model.ConnectionInfo) error {
	if ci.Password == "" && (ci.PrivateKey == "" || ci.PrivateKey == "-") && ci.Kubeconfig == "" {
		return nil // nothing sensitive to store (e.g. CSP connection without SSH)
	}
	if !openbao.Enabled() {
		return errors.New("OpenBao is required to store connection secrets (set cm-honeybee.openbao.address)")
	}
	data := map[string]string{"password": ci.Password, "private_key": ci.PrivateKey, "kubeconfig": ci.Kubeconfig}
	if err := openbao.Put(sshSecretPath(ci.ID), data); err != nil {
		return err
	}
	ci.Password = ""
	ci.PrivateKey = ""
	ci.Kubeconfig = ""
	return nil
}

// hydrateConnectionSecrets loads the SSH secrets from OpenBao into ci before an
// SSH operation. Falls back to whatever is already on ci (DB values) when
// OpenBao is off or the connection predates OpenBao.
func hydrateConnectionSecrets(ci *model.ConnectionInfo) error {
	if !openbao.Enabled() {
		return nil
	}
	data, err := openbao.Get(sshSecretPath(ci.ID))
	if err != nil {
		if errors.Is(err, openbao.ErrNotFound) {
			return nil
		}
		return err
	}
	ci.Password = data["password"]
	ci.PrivateKey = data["private_key"]
	if ci.PrivateKey == "" {
		ci.PrivateKey = "-"
	}
	ci.Kubeconfig = data["kubeconfig"]
	return nil
}

// deleteConnectionSecrets removes a connection's SSH secrets from OpenBao.
func deleteConnectionSecrets(connID string) {
	if !openbao.Enabled() {
		return
	}
	if err := openbao.Delete(sshSecretPath(connID)); err != nil {
		logger.Println(logger.WARN, true, "OpenBao: failed to delete SSH secrets ("+connID+"): "+err.Error())
	}
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
	case "", serverCommon.SourceGroupTypeSSH, serverCommon.SourceGroupTypeOnprem:
		connectionInfo.ResourceType = strings.ToLower(strings.TrimSpace(createConnectionInfoReq.ResourceType))
		switch connectionInfo.ResourceType {
		case "", serverCommon.ResourceTypeVM:
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
		case serverCommon.ResourceTypeK8s:
			connectionInfo.Kubeconfig = createConnectionInfoReq.Kubeconfig
			if err := validateKubeconfig(connectionInfo.Kubeconfig); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("resource_type for on-prem must be one of vm | k8s")
		}
	case serverCommon.SourceGroupTypeCSP:
		connectionInfo.ResourceType = strings.ToLower(strings.TrimSpace(createConnectionInfoReq.ResourceType))
		connectionInfo.ResourceID = strings.TrimSpace(createConnectionInfoReq.ResourceID)
		connectionInfo.Zone = strings.TrimSpace(createConnectionInfoReq.Zone)
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

		// Optional SSH access. cb-spider gives us the CSP-level VM metadata, but
		// OS-level details (CPU/memory/kernel/software) need the in-guest agent.
		// When SSH credentials are provided, honeybee installs and queries that
		// agent just like an SSH source; without them, only CSP metadata is
		// available.
		connectionInfo.IPAddress = strings.TrimSpace(createConnectionInfoReq.IPAddress)
		connectionInfo.SSHPort = strings.TrimSpace(createConnectionInfoReq.SSHPort)
		connectionInfo.User = createConnectionInfoReq.User
		connectionInfo.Password = createConnectionInfoReq.Password
		connectionInfo.PrivateKey = createConnectionInfoReq.PrivateKey
		if connectionInfo.IPAddress != "" {
			if connectionInfo.SSHPort == "" {
				connectionInfo.SSHPort = "22"
			}
			if err := checkPort(connectionInfo.SSHPort); err != nil {
				return nil, err
			}
			if connectionInfo.User == "" {
				return nil, errors.New("user is empty (required when ip_address is provided)")
			}
			if connectionInfo.Password == "" && connectionInfo.PrivateKey == "" {
				return nil, errors.New("password or private_key must be provided when ip_address is provided")
			}
		}
		if connectionInfo.PrivateKey == "" {
			connectionInfo.PrivateKey = "-"
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
		// Load SSH secrets from OpenBao (when enabled) before any SSH operation.
		if err := hydrateConnectionSecrets(connectionInfo); err != nil {
			return nil, err
		}

		sourceGroup, err := dao.SourceGroupGet(connectionInfo.SourceGroupID)
		if err != nil {
			return nil, err
		}

		switch sourceGroup.Type {
		case serverCommon.SourceGroupTypeCSP:
			// connection_status reflects CSP reachability: whether cb-spider can
			// identify this resource. This is a status-only check — CSP data is
			// collected/persisted by import/infra, not here (refresh/registration).
			if err := checkCSPConnection(sourceGroup, connectionInfo); err != nil {
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.ConnectionFailedMessage = err.Error()
			} else {
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusSuccess
				oldConnectionInfo.ConnectionFailedMessage = ""
			}

			// agent_status reflects the REAL in-guest agent: it requires SSH
			// access, so it is only attempted when credentials were provided.
			// Previously this was faked to "success" from the cb-spider result
			// even though no agent was installed.
			if connectionInfo.IPAddress == "" {
				oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.AgentFailedMessage = "no SSH access configured for this CSP connection; " +
					"provide ip_address/user (and password or private_key) to install the agent"
			} else {
				c := &ssh.SSH{}
				if err := c.RunAgent(*connectionInfo); err != nil {
					oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusFailed
					oldConnectionInfo.AgentFailedMessage = err.Error()
				} else {
					c.Close()
					oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusSuccess
					oldConnectionInfo.AgentFailedMessage = ""
				}
			}
		default:
			if connectionInfo.ResourceType == serverCommon.ResourceTypeK8s {
				// on-prem k8s has no SSH host. The kubeconfig (validated at
				// register) is consumed by downstream migration; agent-based
				// collection over SSH does not apply.
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusSuccess
				oldConnectionInfo.ConnectionFailedMessage = ""
				oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.AgentFailedMessage = "agent-based collection is not applicable to on-prem k8s connections"
				break
			}

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

	// Restore SSH secrets (password, private key) from OpenBao before encrypting
	// the response. storeConnectionSecrets moves them out of the DB row into
	// OpenBao, so the row we just read has them empty. Consumers such as
	// cm-grasshopper read these fields (encrypted here, decrypted on their side)
	// to SSH into the source host; without this they receive empty credentials
	// and fail with "failed to determine auth method". The DB is untouched — the
	// refresh status update above uses a field allowlist.
	if err := hydrateConnectionSecrets(oldConnectionInfo); err != nil {
		return nil, err
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

	// Move SSH secrets into OpenBao (when enabled) before persisting the row.
	if err := storeConnectionSecrets(connectionInfo); err != nil {
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

		// Restore SSH secrets from OpenBao so the list carries them encrypted,
		// same as the single-connection GET (see doGetConnectionInfo).
		if err := hydrateConnectionSecrets(&ci); err != nil {
			return common.ReturnErrorMsg(c, err.Error())
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

	// Load existing SSH secrets (OpenBao) so fields not present in the update
	// request keep their current values when re-stored below.
	if err := hydrateConnectionSecrets(oldConnectionInfo); err != nil {
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
		if oldConnectionInfo.ResourceType == serverCommon.ResourceTypeK8s {
			if updateConnectionInfoReq.Kubeconfig != "" {
				if err := validateKubeconfig(updateConnectionInfoReq.Kubeconfig); err != nil {
					return common.ReturnErrorMsg(c, err.Error())
				}
				oldConnectionInfo.Kubeconfig = updateConnectionInfoReq.Kubeconfig
			}
			break
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
	}

	// Persist SSH secrets to OpenBao (when enabled), clearing them from the row.
	if err := storeConnectionSecrets(oldConnectionInfo); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
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

	deleteConnectionSecrets(connectionInfo.ID)

	err = dao.ConnectionInfoDelete(connectionInfo)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "success"}, " ")
}

// countConnectionsOnHost returns how many connections other than excludeID
// target the same host (ip_address). The agent is a host-wide systemd service,
// so this is used to avoid removing an agent still shared by another connection.
func countConnectionsOnHost(ipAddress, excludeID string) (int, error) {
	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{}, 0, 0)
	if err != nil {
		return 0, err
	}
	ip := strings.TrimSpace(ipAddress)
	count := 0
	for _, conn := range *list {
		if conn.ID != excludeID && strings.TrimSpace(conn.IPAddress) == ip {
			count++
		}
	}
	return count, nil
}

// UninstallAgent godoc
//
//	@ID				uninstall-agent
//	@Summary		Uninstall Agent
//	@Description	Uninstall cm-honeybee-agent from the connection's host over SSH. It is kept (not removed) when another connection on the same host (same ip_address) still uses it. This does not delete the connection record.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			connId path string true "ID of the connectionInfo"
//	@Success		200	{object}	model.SimpleMsg			"Agent uninstalled, or kept because it is still shared."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to uninstall the agent."
//	@Router			/source_group/{sgId}/connection_info/{connId}/agent [delete]
func UninstallAgent(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}
	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	if _, err := dao.SourceGroupGet(sgID); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if strings.TrimSpace(connectionInfo.IPAddress) == "" {
		return common.ReturnErrorMsg(c, "connection has no ip_address; the agent is installed only on SSH-reachable hosts")
	}

	// Keep the agent if another connection (in any source group) targets the same
	// host — it is a host-wide service shared across connections.
	shared, err := countConnectionsOnHost(connectionInfo.IPAddress, connectionInfo.ID)
	if err != nil {
		return common.ReturnInternalError(c, err, "failed to check other connections on the host")
	}
	if shared > 0 {
		return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "agent kept: host " +
			connectionInfo.IPAddress + " is still used by " + strconv.Itoa(shared) + " other connection(s)"}, " ")
	}

	// Load SSH secrets (OpenBao) before connecting.
	if err := hydrateConnectionSecrets(connectionInfo); err != nil {
		return common.ReturnInternalError(c, err, "failed to load SSH secrets")
	}

	s := &ssh.SSH{}
	if err := s.UninstallAgent(*connectionInfo); err != nil {
		return common.ReturnInternalError(c, err, "failed to uninstall the agent")
	}

	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "agent uninstalled from " + connectionInfo.IPAddress}, " ")
}

// countConnectionsOnHostOutsideGroup counts connections targeting the same host
// (ip_address) that belong to a source group other than sgID.
func countConnectionsOnHostOutsideGroup(ipAddress, sgID string) (int, error) {
	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{}, 0, 0)
	if err != nil {
		return 0, err
	}
	ip := strings.TrimSpace(ipAddress)
	count := 0
	for _, conn := range *list {
		if conn.SourceGroupID != sgID && strings.TrimSpace(conn.IPAddress) == ip {
			count++
		}
	}
	return count, nil
}

// AgentUninstallResult is one host's outcome in a source-group agent uninstall.
type AgentUninstallResult struct {
	Host    string `json:"host"`
	Message string `json:"message"`
}

// UninstallAgentSourceGroup godoc
//
//	@ID				uninstall-agent-source-group
//	@Summary		Uninstall Agent (Source Group)
//	@Description	Uninstall cm-honeybee-agent from every host in the source group over SSH. Each host is processed once; a host is kept (not removed) when a connection in another source group still uses it. This does not delete the connection records.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Success		200	{object}	model.SimpleMsg			"Per-host uninstall results."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to uninstall the agents."
//	@Router			/source_group/{sgId}/agent [delete]
func UninstallAgentSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}
	if _, err := dao.SourceGroupGet(sgID); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnInternalError(c, err, "failed to list connections")
	}

	results := make([]AgentUninstallResult, 0)
	done := make(map[string]bool) // process each host once

	for i := range *list {
		conn := (*list)[i]
		ip := strings.TrimSpace(conn.IPAddress)
		if ip == "" || done[ip] {
			continue
		}
		done[ip] = true

		shared, err := countConnectionsOnHostOutsideGroup(ip, sgID)
		if err != nil {
			results = append(results, AgentUninstallResult{ip, "error checking other connections: " + err.Error()})
			continue
		}
		if shared > 0 {
			results = append(results, AgentUninstallResult{ip, "kept: still used by " +
				strconv.Itoa(shared) + " connection(s) in other source group(s)"})
			continue
		}

		if err := hydrateConnectionSecrets(&conn); err != nil {
			results = append(results, AgentUninstallResult{ip, "failed to load SSH secrets: " + err.Error()})
			continue
		}
		s := &ssh.SSH{}
		if err := s.UninstallAgent(conn); err != nil {
			results = append(results, AgentUninstallResult{ip, "failed to uninstall: " + err.Error()})
			continue
		}
		results = append(results, AgentUninstallResult{ip, "uninstalled"})
	}

	return c.JSONPretty(http.StatusOK, map[string]interface{}{"results": results}, " ")
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

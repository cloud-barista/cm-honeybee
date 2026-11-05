package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	commonmodel "github.com/cloud-barista/cm-centipede/dmdl/common-model"
	sourcemodel "github.com/cloud-barista/cm-centipede/dmdl/source-model"
	transxex "github.com/cloud-barista/cm-centipede/transx-ex"
	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/ssh"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
)

// ---------------------------------------------------------------------------
// Shared by the filesystem, object storage and database imports
// (importFS.go, importObjectStorage.go, importDB.go).
// ---------------------------------------------------------------------------

// failedUnlessStale keeps a stale mark through a failed re-import: the stored
// data still describes the old scope, and only "stale" makes the get refuse it.
// The next successful import is what clears it.
func failedUnlessStale(current, stale, failed string) string {
	if current == stale {
		return stale
	}
	return failed
}

// sourceGroupConnection loads the source group and the connection, refusing a
// connection that belongs to a different group than the URL names.
func sourceGroupConnection(sgID, connID string) (*model.SourceGroup, *model.ConnectionInfo, error) {
	sg, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return nil, nil, err
	}
	ci, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, nil, err
	}
	if ci.SourceGroupID != sg.ID {
		return nil, nil, errors.New("connection " + connID +
			" does not belong to source group " + sgID)
	}
	return sg, ci, nil
}

// agentErrorResponse mirrors the agent's common.ErrorResponse.
type agentErrorResponse struct {
	Error string `json:"error"`
}

// requestAgentInspect sends req to the agent and decodes a successful reply into
// result, returning the raw reply so the caller can store it verbatim. what
// names the collection in the message ("filesystem", "object storage",
// "database").
func requestAgentInspect(ci model.ConnectionInfo, requestPath, what string, req, result any) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", errors.New("failed to encode the " + what + " request")
	}

	s := &ssh.SSH{}
	status, reply, err := s.SendPostRequestToAgent(ci, requestPath, body)
	if err != nil {
		return "", errors.New("failed to reach the agent for the " + what +
			" information: " + err.Error())
	}

	// The status is what separates a result from an error, so the error body is
	// only ever read for its message — never decoded into result. That is the
	// trap this avoids: an ErrorResponse decodes cleanly into every inspect
	// result type, missing fields and all, so a failed inspect would otherwise
	// be stored as a successful, empty one.
	if status != http.StatusOK {
		var agentErr agentErrorResponse
		if json.Unmarshal([]byte(reply), &agentErr) == nil && agentErr.Error != "" {
			return "", errors.New("the agent failed to get the " + what +
				" information (HTTP " + strconv.Itoa(status) + "): " + agentErr.Error)
		}
		return "", errors.New("the agent failed to get the " + what +
			" information (HTTP " + strconv.Itoa(status) + "): " + agentResponseExcerpt(reply))
	}

	if strings.TrimSpace(reply) == "" {
		return "", errors.New("the agent returned an empty " + what + " response")
	}
	if err := json.Unmarshal([]byte(reply), result); err != nil {
		return "", errors.New("the agent returned an unreadable " + what +
			" response: " + agentResponseExcerpt(reply))
	}

	return reply, nil
}

// agentResponseExcerpt keeps an unreadable body short enough for a log line and
// an API message — a real listing can run to megabytes. Cut on runes so a
// multi-byte character never breaks in half.
func agentResponseExcerpt(body string) string {
	const max = 200

	body = strings.ReplaceAll(body, "\n", " ")
	if r := []rune(body); len(r) > max {
		return string(r[:max]) + "..."
	}

	return body
}

// ---------------------------------------------------------------------------
// Filesystem import
// ---------------------------------------------------------------------------

// agentFSInspectReq is the body of the agent's POST /fs. Its JSON tags have to
// match the agent's FSInspectReq exactly; the two are separate types because the
// server and the agent are separate modules, so a mistyped tag shows up as a
// field silently lost at runtime rather than as a build failure.
type agentFSInspectReq struct {
	Path     string                     `json:"path,omitempty"`
	MaxDepth int                        `json:"max_depth,omitempty"`
	Metric   *transxex.FilesystemMetric `json:"metric,omitempty"`
}

func doImportFS(connID string, req model.ImportFSReq) (*model.SavedFSInfo, error) {
	total := time.Now()
	defer func() {
		logImportStep("fs", connID, "total", total, "")
	}()

	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, err
	}

	sourceGroup, err := dao.SourceGroupGet(connectionInfo.SourceGroupID)
	if err != nil {
		return nil, err
	}

	// Target guards run before any record is registered, so a rejected request
	// leaves no "importing" row behind. The inspect executes inside the in-guest
	// agent, which is reached over SSH.
	//
	// checkAgentImportable is deliberately not called: it filters out on-prem k8s
	// connections, and an fs source group takes no resource_type, so it can hold
	// no k8s connection to filter.
	if !serverCommon.IsFSType(sourceGroup.Type) {
		return nil, errors.New("filesystem inspection is supported only for fs source group")
	}
	if connectionInfo.IPAddress == "" {
		return nil, errors.New("ip_address is empty (required to reach the agent)")
	}

	// The SSH credentials come from OpenBao and SendPostRequestToAgent dials the
	// host with them, so the hydrate has to stay ahead of it.
	if err := hydrateConnectionSecrets(connectionInfo); err != nil {
		return nil, err
	}

	oldSavedFSInfo, _ := dao.SavedFSInfoGet(connectionInfo.ID)

	if oldSavedFSInfo == nil {
		savedFSInfo := new(model.SavedFSInfo)
		savedFSInfo.ConnectionID = connectionInfo.ID
		savedFSInfo.FSData = ""
		savedFSInfo.Status = model.SavedFSInfoStatusImporting
		savedFSInfo.SavedTime = time.Now()
		registered, err := dao.SavedFSInfoRegister(savedFSInfo)
		if err != nil {
			// A concurrent import of the same connection may have registered the row
			// between the get above and this create; carry on with that row.
			registered, _ = dao.SavedFSInfoGet(connectionInfo.ID)
			if registered == nil {
				errMsg := "Error occurred while getting filesystem information." +
					" (ConnectionID = " + connectionInfo.ID + ")"
				logger.Println(logger.ERROR, false, errMsg)
				return nil, errors.New(errMsg)
			}
		}
		oldSavedFSInfo = registered
	}

	// The request travels as a JSON body on the session's stdin, so nothing here
	// has to survive a remote shell's quoting. An omitted field is what tells the
	// agent to keep its own default, which is why MaxDepth is sent as given:
	// omitempty drops a zero, and the agent reads a missing depth as unlimited.
	//
	// The path comes from the connection, normalized at registration, so there is
	// no default to apply here and a group-level import reads each host at its
	// own root.
	agentReq := agentFSInspectReq{
		Path:     connectionInfo.FSScanPath,
		MaxDepth: req.MaxDepth,
		Metric:   fsMetric(req.Metric),
	}

	// The reply is checked before it is stored: the agent reports a failed
	// inspect with a 4xx/5xx and an error body, which without the check would
	// reach here as a successful, empty collection. fsResult only proves the
	// reply has the right shape — it is never read.
	var fsResult sourcemodel.FSInfo
	fetchStart := time.Now()
	data, err := requestAgentInspect(*connectionInfo, "/fs", "filesystem", agentReq, &fsResult)
	logImportStep("fs", connectionInfo.ID, "agent fetch", fetchStart, strconv.Itoa(len(data))+" bytes")
	if err != nil {
		oldSavedFSInfo.Status = failedUnlessStale(oldSavedFSInfo.Status,
			model.SavedFSInfoStatusStale, model.SavedFSInfoStatusFailed)
		_ = dao.SavedFSInfoUpdate(oldSavedFSInfo)
		errMsg := "Error occurred while getting filesystem information." +
			" (ConnectionID = " + connectionInfo.ID + ", Error = " + err.Error() + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	// The agent's result is stored verbatim, so get and refined return exactly
	// what was collected.
	oldSavedFSInfo.FSData = data
	oldSavedFSInfo.Status = model.SavedFSInfoStatusSuccess
	oldSavedFSInfo.SavedTime = time.Now()
	err = dao.SavedFSInfoUpdate(oldSavedFSInfo)
	if err != nil {
		errMsg := "Error occurred while saving the filesystem information." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	return oldSavedFSInfo, nil
}

func doGetFSInfo(connID string) (*sourcemodel.FSInfo, error) {
	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, err
	}

	savedFSInfo, err := dao.SavedFSInfoGet(connectionInfo.ID)
	if err != nil {
		errMsg := "Failed to get information of the filesystem." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	// A stale result was collected under a scan path this connection no longer
	// has, so answering with it would describe the wrong tree. FSInfo carries no
	// status field, which leaves the consumer no way to notice — hence a refusal
	// rather than a flag. The row itself is kept; only reading it as current is
	// refused, and the next import clears the state.
	if savedFSInfo.Status == model.SavedFSInfoStatusStale {
		errMsg := "The filesystem information was collected for a different scan path." +
			" Run the import again. (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	var fsInfo sourcemodel.FSInfo
	err = json.Unmarshal([]byte(savedFSInfo.FSData), &fsInfo)
	if err != nil {
		errMsg := "Error occurred while parsing filesystem information." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	return &fsInfo, nil
}

func doGetFSInfoRefined(connID string) (*sourcemodel.SourceFileSystemModel, error) {
	fsInfo, err := doGetFSInfo(connID)
	if err != nil {
		return nil, err
	}

	// Refining a filesystem adds the connection reference and nothing else: the
	// inspect result is already the shape a consumer wants. ConnectionRef covers
	// the connection sources of every consumer of the model, so only the honeybee
	// one is filled here; the rest are omitempty and never reach the response.
	return &sourcemodel.SourceFileSystemModel{
		Connection: commonmodel.ConnectionRef{
			Source:   commonmodel.ConnectionSourceHoneybee,
			Honeybee: &commonmodel.HoneybeeRef{ConnectionID: connID},
		},
		FSInfo: *fsInfo,
	}, nil
}

// ImportFS godoc
//
//	@ID				import-fs
//	@Summary		Import filesystem
//	@Description	Import the filesystem information of the connection information. The path inspected is the connection's fs_scan_path.
//	@Tags			[Import] Import source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Param			ImportFSReq body model.ImportFSReq false "Scan depth and metric selection."
//	@Success		200	{object}	model.SavedFSInfo		"Successfully saved the filesystem information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to save the filesystem information"
//	@Router			/source_group/{sgId}/connection_info/{connId}/import/fs [post]
func ImportFS(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, _, err := sourceGroupConnection(sgID, connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	importFSReq := new(model.ImportFSReq)
	err = c.Bind(importFSReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	savedFSInfo, err := doImportFS(connID, *importFSReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, savedFSInfo, " ")
}

// ImportFSSourceGroup godoc
//
//	@ID				import-fs-source-group
//	@Summary		Import filesystem Source Group
//	@Description	Import filesystem information for all connections in the source group. Each connection is inspected at its own fs_scan_path.
//	@Tags			[Import] Import source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			ImportFSReq body model.ImportFSReq false "Scan depth and metric selection applied to every connection."
//	@Success		200	{object}	[]model.SavedFSInfo		"Successfully saved the filesystem information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to save the filesystem information"
//	@Router			/source_group/{sgId}/import/fs [post]
func ImportFSSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	importFSReq := new(model.ImportFSReq)
	err = c.Bind(importFSReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var savedFSInfoList []model.SavedFSInfo

	for _, conn := range *list {
		// A failure fails the whole group, as the other group imports do. The
		// ErrOnpremK8sAgent skip they carry has no counterpart here: an fs group
		// holds no k8s connection to skip.
		savedFSInfo, err := doImportFS(conn.ID, *importFSReq)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		savedFSInfoList = append(savedFSInfoList, *savedFSInfo)
	}

	return c.JSONPretty(http.StatusOK, savedFSInfoList, " ")
}

// GetFSInfo godoc
//
//	@ID				get-fs-info
//	@Summary		Get Filesystem Information
//	@Description	Get the filesystem information of the connection information.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	sourcemodel.FSInfo		"Successfully get information of the filesystem."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the filesystem."
//	@Router			/source_group/{sgId}/connection_info/{connId}/fs [get]
func GetFSInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, _, err := sourceGroupConnection(sgID, connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	fsInfo, err := doGetFSInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, fsInfo, " ")
}

// GetFSInfoSourceGroup godoc
//
//	@ID				get-fs-info-source-group
//	@Summary		Get Filesystem Information Source Group
//	@Description	Get the filesystem information for all connections in the source group.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	model.FSInfoList		"Successfully get information of the filesystem."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the filesystem."
//	@Router			/source_group/{sgId}/fs [get]
func GetFSInfoSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var fsInfoList model.FSInfoList

	for _, conn := range *list {
		fsInfo, err := doGetFSInfo(conn.ID)
		if err != nil || fsInfo == nil {
			// Not collected yet, unreadable, or stale — skip (logged in
			// doGetFSInfo). A stale result describes a path the connection no
			// longer has, so leaving it out is the same answer as "nothing
			// collected" and better than mixing it into the group's listing.
			continue
		}
		fsInfoList.FileSystems = append(fsInfoList.FileSystems, *fsInfo)
	}

	return c.JSONPretty(http.StatusOK, fsInfoList, " ")
}

// GetFSInfoRefined godoc
//
//	@ID				get-fs-info-refined
//	@Summary		Get Refined Filesystem Information
//	@Description	Get the refined filesystem information of the connection information.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	sourcemodel.SourceDataMigrationModel	"Successfully get refined information of the filesystem."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the filesystem."
//	@Router			/source_group/{sgId}/connection_info/{connId}/fs/refined [get]
func GetFSInfoRefined(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, _, err := sourceGroupConnection(sgID, connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	sourceFileSystem, err := doGetFSInfoRefined(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	// A single connection still returns the wrapper, so a consumer parses the
	// same shape at both levels — as GetInfraInfoRefined does for one node.
	var sourceDataMigrationModel sourcemodel.SourceDataMigrationModel
	sourceDataMigrationModel.SourceDataMigrationModel.FileSystems =
		append(sourceDataMigrationModel.SourceDataMigrationModel.FileSystems, *sourceFileSystem)

	return c.JSONPretty(http.StatusOK, sourceDataMigrationModel, " ")
}

// GetFSInfoSourceGroupRefined godoc
//
//	@ID				get-fs-info-source-group-refined
//	@Summary		Get Refined Filesystem Information Source Group
//	@Description	Get the refined filesystem information for all connections in the source group.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	sourcemodel.SourceDataMigrationModel	"Successfully get refined information of the filesystem."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the filesystem."
//	@Router			/source_group/{sgId}/fs/refined [get]
func GetFSInfoSourceGroupRefined(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var sourceDataMigrationModel sourcemodel.SourceDataMigrationModel

	for _, conn := range *list {
		sourceFileSystem, err := doGetFSInfoRefined(conn.ID)
		if err != nil || sourceFileSystem == nil {
			continue // not collected yet / unreadable / stale — skip (logged in doGetFSInfo)
		}
		sourceDataMigrationModel.SourceDataMigrationModel.FileSystems =
			append(sourceDataMigrationModel.SourceDataMigrationModel.FileSystems, *sourceFileSystem)
	}

	return c.JSONPretty(http.StatusOK, sourceDataMigrationModel, " ")
}

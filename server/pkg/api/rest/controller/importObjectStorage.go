package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	commonmodel "github.com/cloud-barista/cm-centipede/dmdl/common-model"
	sourcemodel "github.com/cloud-barista/cm-centipede/dmdl/source-model"
	transxex "github.com/cloud-barista/cm-centipede/transx-ex"
	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
)

// agentOSInspectReq is the body of the agent's POST /objectstorage. Its JSON
// tags have to match the agent's ObjectStorageInspectReq exactly; the two are
// separate types because the server and the agent are separate modules, so a
// mistyped tag shows up as a field silently lost at runtime rather than as a
// build failure.
type agentOSInspectReq struct {
	Bucket       string                        `json:"bucket"`
	Prefix       string                        `json:"prefix,omitempty"`
	Endpoint     string                        `json:"endpoint"`
	AccessKey    string                        `json:"access_key,omitempty"`
	SecretKey    string                        `json:"secret_key,omitempty"`
	Region       string                        `json:"region,omitempty"`
	UseSSL       bool                          `json:"use_ssl,omitempty"`
	BucketLookup string                        `json:"bucket_lookup,omitempty"`
	Metric       *transxex.ObjectStorageMetric `json:"metric,omitempty"`
}

func doImportObjectStorage(connID string, req model.ImportObjectStorageReq) (*model.SavedObjectStorageInfo, error) {
	total := time.Now()
	defer func() {
		logImportStep("objectstorage", connID, "total", total, "")
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
	// leaves no "importing" row behind. csp is named separately from the general
	// case because it does hold buckets — they are discovered through cb-spider
	// (see discover.go), which is a different path from this one.
	//
	// The bucket is not guarded here: it comes from the connection, where
	// registration required it and an update reads an empty value as "no change",
	// so a stored one cannot be blank.
	if serverCommon.IsCSPType(sourceGroup.Type) {
		return nil, errors.New("object storage inspection is not supported for csp source group")
	}
	if !serverCommon.IsMinIOType(sourceGroup.Type) {
		return nil, errors.New("object storage inspection is supported only for minio source group")
	}

	// The credentials live in OpenBao, and azure derives its endpoint from the
	// access key id, so the secrets have to be restored before resolving.
	if err := hydrateConnectionSecrets(connectionInfo); err != nil {
		return nil, err
	}
	if connectionInfo.OSAccessType == model.AccessTypeSSHTunnel && connectionInfo.IPAddress == "" {
		return nil, errors.New("ip_address is empty (required to reach the agent for ssh_tunnel access)")
	}

	// The provider table decides the endpoint for both paths below: per-provider
	// knowledge stays on the server and the agent just uses what it is given.
	endpoint, region, useSSL, bucketLookup, err := resolveS3Endpoint(sourceGroup, connectionInfo)
	if err != nil {
		return nil, err
	}

	oldSavedObjectStorageInfo, _ := dao.SavedObjectStorageInfoGet(connectionInfo.ID)

	if oldSavedObjectStorageInfo == nil {
		savedObjectStorageInfo := new(model.SavedObjectStorageInfo)
		savedObjectStorageInfo.ConnectionID = connectionInfo.ID
		savedObjectStorageInfo.Bucket = connectionInfo.OSScanBucket
		savedObjectStorageInfo.OSData = ""
		savedObjectStorageInfo.Status = model.SavedObjectStorageInfoStatusImporting
		savedObjectStorageInfo.SavedTime = time.Now()
		registered, err := dao.SavedObjectStorageInfoRegister(savedObjectStorageInfo)
		if err != nil {
			// A concurrent import of the same connection may have registered the row
			// between the get above and this create; carry on with that row.
			registered, _ = dao.SavedObjectStorageInfoGet(connectionInfo.ID)
			if registered == nil {
				errMsg := "Error occurred while getting object storage information." +
					" (ConnectionID = " + connectionInfo.ID + ")"
				logger.Println(logger.ERROR, false, errMsg)
				return nil, errors.New(errMsg)
			}
		}
		oldSavedObjectStorageInfo = registered
	}

	var data string

	if connectionInfo.OSAccessType == model.AccessTypeSSHTunnel {
		// The bucket is only reachable from the source host, so the agent runs the
		// inspect. The credentials travel as a JSON body on the session's stdin,
		// which keeps them out of the remote host's process list and out of both
		// sides' logs; nothing here has to survive a remote shell's quoting.
		agentReq := agentOSInspectReq{
			Bucket:       connectionInfo.OSScanBucket,
			Prefix:       connectionInfo.OSScanPrefix,
			Endpoint:     endpoint,
			AccessKey:    connectionInfo.OSAccessKeyId,
			SecretKey:    connectionInfo.OSSecretAccessKey,
			Region:       region,
			UseSSL:       useSSL,
			BucketLookup: bucketLookup,
			Metric:       osMetric(req.Metric),
		}

		// The reply is checked before it is stored: the agent reports a failed
		// inspect with a 4xx/5xx and an error body, which without the check would
		// reach here as a successful, empty collection. That check is also what
		// makes this branch fail like the direct one below, which has always had a
		// real error to look at. osResult only proves the reply has the right
		// shape — it is never read.
		var osResult sourcemodel.ObjectStorageInfo
		fetchStart := time.Now()
		data, err = requestAgentInspect(*connectionInfo, "/objectstorage", "object storage", agentReq, &osResult)
		logImportStep("objectstorage", connectionInfo.ID, "agent fetch", fetchStart, strconv.Itoa(len(data))+" bytes")
		if err != nil {
			oldSavedObjectStorageInfo.Status = failedUnlessStale(oldSavedObjectStorageInfo.Status,
				model.SavedObjectStorageInfoStatusStale, model.SavedObjectStorageInfoStatusFailed)
			_ = dao.SavedObjectStorageInfoUpdate(oldSavedObjectStorageInfo)
			errMsg := "Error occurred while getting object storage information." +
				" (ConnectionID = " + connectionInfo.ID + ", Error = " + err.Error() + ")"
			logger.Println(logger.ERROR, false, errMsg)
			return nil, errors.New(errMsg)
		}
	} else {
		// Direct access: the server reaches the endpoint itself, so no agent is
		// involved and none needs to be deployed.
		loc := transxex.StorageLocation{
			StorageType: transxex.StorageTypeObjectStorage,
			Path:        connectionInfo.OSScanBucket + "/" + connectionInfo.OSScanPrefix,
			ObjectStorage: &transxex.ObjectStorageAccess{
				AccessType: transxex.AccessTypeMinio,
				Minio: &transxex.S3MinioConfig{
					Endpoint:        endpoint,
					AccessKeyId:     connectionInfo.OSAccessKeyId,
					SecretAccessKey: connectionInfo.OSSecretAccessKey,
					Region:          region,
					UseSSL:          useSSL,
					BucketLookup:    bucketLookup,
				},
			},
		}
		// Left nil when nothing was selected, so the library keeps its own
		// defaults rather than reading a zeroed struct as "everything off".
		if metric := osMetric(req.Metric); metric != nil {
			loc.Metric = &transxex.StorageMetricOption{ObjectStorage: metric}
		}

		inspectStart := time.Now()
		result, err := transxex.InspectObjectStorage(loc)
		logImportStep("objectstorage", connectionInfo.ID, "direct inspect", inspectStart, "")
		if err != nil {
			oldSavedObjectStorageInfo.Status = failedUnlessStale(oldSavedObjectStorageInfo.Status,
				model.SavedObjectStorageInfoStatusStale, model.SavedObjectStorageInfoStatusFailed)
			_ = dao.SavedObjectStorageInfoUpdate(oldSavedObjectStorageInfo)
			errMsg := "Error occurred while getting object storage information." +
				" (ConnectionID = " + connectionInfo.ID + ", Error = " + err.Error() + ")"
			logger.Println(logger.ERROR, false, errMsg)
			return nil, errors.New(errMsg)
		}

		// Serialized here so that both paths store the result the agent would have
		// returned, keeping the two routes indistinguishable downstream. Only the
		// whitespace differs: the agent replies through JSONPretty, this is
		// compact, and get and refined parse either.
		encodedResult, err := json.Marshal(result)
		if err != nil {
			oldSavedObjectStorageInfo.Status = failedUnlessStale(oldSavedObjectStorageInfo.Status,
				model.SavedObjectStorageInfoStatusStale, model.SavedObjectStorageInfoStatusFailed)
			_ = dao.SavedObjectStorageInfoUpdate(oldSavedObjectStorageInfo)
			errMsg := "Error occurred while encoding the object storage information." +
				" (ConnectionID = " + connectionInfo.ID + ")"
			logger.Println(logger.ERROR, false, errMsg)
			return nil, errors.New(errMsg)
		}
		data = string(encodedResult)
	}

	// Bucket is written on both the register and the update path: a row created
	// by an earlier import carries the bucket of that import, and this is where
	// it catches up with the connection.
	oldSavedObjectStorageInfo.Bucket = connectionInfo.OSScanBucket
	oldSavedObjectStorageInfo.OSData = data
	oldSavedObjectStorageInfo.Status = model.SavedObjectStorageInfoStatusSuccess
	oldSavedObjectStorageInfo.SavedTime = time.Now()
	err = dao.SavedObjectStorageInfoUpdate(oldSavedObjectStorageInfo)
	if err != nil {
		errMsg := "Error occurred while saving the object storage information." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	return oldSavedObjectStorageInfo, nil
}

func doGetObjectStorageInfo(connID string) (*sourcemodel.ObjectStorageInfo, error) {
	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, err
	}

	savedObjectStorageInfo, err := dao.SavedObjectStorageInfoGet(connectionInfo.ID)
	if err != nil {
		errMsg := "Failed to get information of the object storage." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	// A stale result was collected under a bucket or prefix this connection no
	// longer has, so answering with it would describe the wrong target.
	// ObjectStorageInfo carries no status field, which leaves the consumer no way
	// to notice — hence a refusal rather than a flag. The row itself is kept, and
	// SavedObjectStorageInfo.Bucket is what lets it still say which bucket it
	// came from; only reading it as current is refused.
	if savedObjectStorageInfo.Status == model.SavedObjectStorageInfoStatusStale {
		errMsg := "The object storage information was collected for a different bucket or prefix." +
			" Run the import again. (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	var objectStorageInfo sourcemodel.ObjectStorageInfo
	err = json.Unmarshal([]byte(savedObjectStorageInfo.OSData), &objectStorageInfo)
	if err != nil {
		errMsg := "Error occurred while parsing object storage information." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	return &objectStorageInfo, nil
}

func doGetObjectStorageInfoRefined(connID string) (*sourcemodel.SourceObjectStorageModel, error) {
	objectStorageInfo, err := doGetObjectStorageInfo(connID)
	if err != nil {
		return nil, err
	}

	// Refining adds the connection reference and nothing else: the inspect result
	// is already the shape a consumer wants. ConnectionRef covers the connection
	// sources of every consumer of the model, so only the honeybee one is filled
	// here; the rest are omitempty and never reach the response.
	return &sourcemodel.SourceObjectStorageModel{
		Connection: commonmodel.ConnectionRef{
			Source:   commonmodel.ConnectionSourceHoneybee,
			Honeybee: &commonmodel.HoneybeeRef{ConnectionID: connID},
		},
		ObjectStorageInfo: *objectStorageInfo,
	}, nil
}

// ImportObjectStorage godoc
//
//	@ID				import-object-storage
//	@Summary		Import object storage
//	@Description	Import the object storage information of the connection information. The bucket and prefix inspected are the connection's os_scan_bucket and os_scan_prefix.
//	@Tags			[Import] Import source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Param			ImportObjectStorageReq body model.ImportObjectStorageReq false "Metric selection."
//	@Success		200	{object}	model.SavedObjectStorageInfo	"Successfully saved the object storage information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to save the object storage information"
//	@Router			/source_group/{sgId}/connection_info/{connId}/import/objectstorage [post]
func ImportObjectStorage(c echo.Context) error {
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

	importObjectStorageReq := new(model.ImportObjectStorageReq)
	err = c.Bind(importObjectStorageReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	savedObjectStorageInfo, err := doImportObjectStorage(connID, *importObjectStorageReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, savedObjectStorageInfo, " ")
}

// ImportObjectStorageSourceGroup godoc
//
//	@ID				import-object-storage-source-group
//	@Summary		Import object storage Source Group
//	@Description	Import object storage information for all connections in the source group. Each connection is inspected at its own os_scan_bucket and os_scan_prefix.
//	@Tags			[Import] Import source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			ImportObjectStorageReq body model.ImportObjectStorageReq false "Metric selection applied to every connection."
//	@Success		200	{object}	[]model.SavedObjectStorageInfo	"Successfully saved the object storage information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to save the object storage information"
//	@Router			/source_group/{sgId}/import/objectstorage [post]
func ImportObjectStorageSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	importObjectStorageReq := new(model.ImportObjectStorageReq)
	err = c.Bind(importObjectStorageReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var savedObjectStorageInfoList []model.SavedObjectStorageInfo

	for _, conn := range *list {
		// A failure fails the whole group, as the other group imports do. The
		// ErrOnpremK8sAgent skip they carry has no counterpart here: a minio group
		// holds no k8s connection to skip.
		savedObjectStorageInfo, err := doImportObjectStorage(conn.ID, *importObjectStorageReq)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		savedObjectStorageInfoList = append(savedObjectStorageInfoList, *savedObjectStorageInfo)
	}

	return c.JSONPretty(http.StatusOK, savedObjectStorageInfoList, " ")
}

// GetObjectStorageInfo godoc
//
//	@ID				get-object-storage-info
//	@Summary		Get Object Storage Information
//	@Description	Get the object storage information of the connection information.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	sourcemodel.ObjectStorageInfo	"Successfully get information of the object storage."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the object storage."
//	@Router			/source_group/{sgId}/connection_info/{connId}/objectstorage [get]
func GetObjectStorageInfo(c echo.Context) error {
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

	objectStorageInfo, err := doGetObjectStorageInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, objectStorageInfo, " ")
}

// GetObjectStorageInfoSourceGroup godoc
//
//	@ID				get-object-storage-info-source-group
//	@Summary		Get Object Storage Information Source Group
//	@Description	Get the object storage information for all connections in the source group.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	model.ObjectStorageInfoList	"Successfully get information of the object storage."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the object storage."
//	@Router			/source_group/{sgId}/objectstorage [get]
func GetObjectStorageInfoSourceGroup(c echo.Context) error {
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

	var objectStorageInfoList model.ObjectStorageInfoList

	for _, conn := range *list {
		objectStorageInfo, err := doGetObjectStorageInfo(conn.ID)
		if err != nil || objectStorageInfo == nil {
			// Not collected yet, unreadable, or stale — skip (logged in
			// doGetObjectStorageInfo). A stale result describes a bucket or prefix
			// the connection no longer has, so leaving it out is the same answer as
			// "nothing collected" and better than mixing it into the listing.
			continue
		}
		objectStorageInfoList.ObjectStorages = append(objectStorageInfoList.ObjectStorages, *objectStorageInfo)
	}

	return c.JSONPretty(http.StatusOK, objectStorageInfoList, " ")
}

// GetObjectStorageInfoRefined godoc
//
//	@ID				get-object-storage-info-refined
//	@Summary		Get Refined Object Storage Information
//	@Description	Get the refined object storage information of the connection information.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	sourcemodel.SourceDataMigrationModel	"Successfully get refined information of the object storage."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the object storage."
//	@Router			/source_group/{sgId}/connection_info/{connId}/objectstorage/refined [get]
func GetObjectStorageInfoRefined(c echo.Context) error {
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

	sourceObjectStorage, err := doGetObjectStorageInfoRefined(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	// A single connection still returns the wrapper, so a consumer parses the
	// same shape at both levels — as GetInfraInfoRefined does for one node.
	var sourceDataMigrationModel sourcemodel.SourceDataMigrationModel
	sourceDataMigrationModel.SourceDataMigrationModel.ObjectStorages =
		append(sourceDataMigrationModel.SourceDataMigrationModel.ObjectStorages, *sourceObjectStorage)

	return c.JSONPretty(http.StatusOK, sourceDataMigrationModel, " ")
}

// GetObjectStorageInfoSourceGroupRefined godoc
//
//	@ID				get-object-storage-info-source-group-refined
//	@Summary		Get Refined Object Storage Information Source Group
//	@Description	Get the refined object storage information for all connections in the source group.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	sourcemodel.SourceDataMigrationModel	"Successfully get refined information of the object storage."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the object storage."
//	@Router			/source_group/{sgId}/objectstorage/refined [get]
func GetObjectStorageInfoSourceGroupRefined(c echo.Context) error {
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
		sourceObjectStorage, err := doGetObjectStorageInfoRefined(conn.ID)
		if err != nil || sourceObjectStorage == nil {
			continue // not collected yet / unreadable / stale — skip (logged in doGetObjectStorageInfo)
		}
		sourceDataMigrationModel.SourceDataMigrationModel.ObjectStorages =
			append(sourceDataMigrationModel.SourceDataMigrationModel.ObjectStorages, *sourceObjectStorage)
	}

	return c.JSONPretty(http.StatusOK, sourceDataMigrationModel, " ")
}

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
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
)

// agentDBInspectReq is the body of the agent's POST /db. Its JSON tags have to
// match the agent's DBInspectReq exactly; the two are separate types because
// the server and the agent are separate modules, so a mistyped tag shows up as
// a field silently lost at runtime rather than as a build failure.
type agentDBInspectReq struct {
	DBType         string                     `json:"db_type"`
	DBName         string                     `json:"db_name,omitempty"`
	Host           string                     `json:"host"`
	Port           int                        `json:"port,omitempty"`
	Username       string                     `json:"username,omitempty"`
	Password       string                     `json:"password,omitempty"`
	ConnectTimeout int                        `json:"connect_timeout,omitempty"`
	TLSMode        string                     `json:"tls_mode,omitempty"`
	TLSCAPEM       string                     `json:"tls_ca_pem,omitempty"`
	AuthSource     string                     `json:"auth_source,omitempty"`
	PgSchema       []string                   `json:"pg_schema,omitempty"`
	ProviderName   string                     `json:"provider_name,omitempty"`
	Metric         *transxex.DBMSMetricOption `json:"metric,omitempty"`
}

// dbImportStatus maps how much of a database import succeeded onto its three
// states. A server holds many databases and an account may not be able to read
// them all, so "some worked" is a real outcome that a consumer has to see.
//
// Both access paths route their counts through here, so the same numbers mean
// the same thing whether the server or the agent did the collecting.
func dbImportStatus(collected, skipped int) string {
	switch {
	case collected == 0:
		return model.SavedDBInfoStatusFailed
	case skipped > 0:
		return model.SavedDBInfoStatusPartial
	default:
		return model.SavedDBInfoStatusSuccess
	}
}

// dbPortNumber reads the connection's port, which is stored as text but travels
// as a number. An empty port is 0, which both the library and the agent read as
// "the engine's default"; anything else that will not parse is the connection's
// problem and is reported as one rather than sent on to fail further away.
func dbPortNumber(port string) (int, error) {
	if port == "" {
		return 0, nil
	}

	return strconv.Atoi(port)
}

func doImportDB(connID string, req model.ImportDBReq) (*model.SavedDBInfo, error) {
	total := time.Now()
	defer func() {
		logImportStep("db", connID, "total", total, "")
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
	// leaves no "importing" row behind. csp is named separately because a csp
	// group can legitimately hold managed databases — they are simply not reached
	// this way. Without the second guard an onprem group would fall through and
	// fail somewhere inside the driver with a message about nothing in
	// particular.
	if serverCommon.IsCSPType(sourceGroup.Type) {
		return nil, errors.New("database inspection is not supported for csp source group")
	}
	if !serverCommon.IsDBType(sourceGroup.Type) {
		return nil, errors.New("database inspection is supported only for db source group")
	}

	// The password lives in OpenBao and the DB row holds a blank, so it has to be
	// restored before either path can connect.
	if err := hydrateConnectionSecrets(connectionInfo); err != nil {
		return nil, err
	}
	if connectionInfo.DBAccessType == model.AccessTypeSSHTunnel && connectionInfo.IPAddress == "" {
		return nil, errors.New("ip_address is empty (required to reach the agent for ssh_tunnel access)")
	}

	// Routed once here: the agent receives an option already narrowed to this
	// engine and needs no engine knowledge of its own.
	metric := dbMetric(connectionInfo.DBType, req.Metric)

	oldSavedDBInfo, _ := dao.SavedDBInfoGet(connectionInfo.ID)

	if oldSavedDBInfo == nil {
		savedDBInfo := new(model.SavedDBInfo)
		savedDBInfo.ConnectionID = connectionInfo.ID
		savedDBInfo.DBData = ""
		// A literal, unlike the four status constants in model/savedDBInfo.go:
		// those exist because their values are read outside this file (the
		// invalidation in connectionInfo.go, and the get handlers), while
		// "importing" is written here and nowhere else. The other domains spell
		// this the same way.
		savedDBInfo.Status = "importing"
		savedDBInfo.SavedTime = time.Now()
		registered, err := dao.SavedDBInfoRegister(savedDBInfo)
		if err != nil {
			// A concurrent import of the same connection may have registered the row
			// between the get above and this create; carry on with that row.
			registered, _ = dao.SavedDBInfoGet(connectionInfo.ID)
			if registered == nil {
				errMsg := "Error occurred while getting database information." +
					" (ConnectionID = " + connectionInfo.ID + ")"
				logger.Println(logger.ERROR, false, errMsg)
				return nil, errors.New(errMsg)
			}
		}
		oldSavedDBInfo = registered
	}

	port, err := dbPortNumber(connectionInfo.DBPort)
	if err != nil {
		oldSavedDBInfo.Status = failedUnlessStale(oldSavedDBInfo.Status,
			model.SavedDBInfoStatusStale, model.SavedDBInfoStatusFailed)
		_ = dao.SavedDBInfoUpdate(oldSavedDBInfo)
		errMsg := "Error occurred while reading the database port." +
			" (ConnectionID = " + connectionInfo.ID + ", Port = " + connectionInfo.DBPort + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	var data string
	var collected, skipped int
	// Why each skipped database was left out, so a collection that read nothing
	// can say why instead of only that it failed.
	var reasons []string

	if connectionInfo.DBAccessType == model.AccessTypeSSHTunnel {
		// The DBMS is only reachable from the source host, so the agent connects to
		// it directly from there. The credentials travel as a JSON body on the
		// session's stdin, which keeps them out of the remote host's process list
		// and out of both sides' logs; nothing here has to survive a remote
		// shell's quoting, the TLS CA's PEM newlines included.
		agentReq := agentDBInspectReq{
			DBType: connectionInfo.DBType,
			// Sent even when empty: that is exactly what tells the agent to
			// enumerate every user database.
			DBName:         connectionInfo.DBName,
			Host:           connectionInfo.DBHost,
			Port:           port,
			Username:       connectionInfo.DBUsername,
			Password:       connectionInfo.DBPassword,
			ConnectTimeout: connectionInfo.DBConnectTimeout,
			// Sent even when empty, which the agent reads as "disable".
			TLSMode:    connectionInfo.DBTLSMode,
			TLSCAPEM:   connectionInfo.DBTLSCAPEM,
			AuthSource: connectionInfo.DBAuthSource,
			PgSchema:   connectionInfo.DBPgSchema,
			// The agent has no source group of its own, so where the database is
			// hosted travels with the request like the rest of the location.
			ProviderName: sourceGroup.ProviderName,
			Metric:       metric,
		}

		// Parsed only to count what came back; the response itself is stored as-is.
		// The check inside requestAgentInspect has to happen before the count: an
		// ErrorResponse decodes cleanly into DBInspectResult, so an unchecked
		// failure would reach the collected == 0 branch below and report "no
		// database could be inspected" instead of the reason the agent gave.
		var agentResult model.DBInspectResult
		fetchStart := time.Now()
		data, err = requestAgentInspect(*connectionInfo, "/db", "database", agentReq, &agentResult)
		logImportStep("db", connectionInfo.ID, "agent fetch", fetchStart, strconv.Itoa(len(data))+" bytes")
		if err != nil {
			oldSavedDBInfo.Status = failedUnlessStale(oldSavedDBInfo.Status,
				model.SavedDBInfoStatusStale, model.SavedDBInfoStatusFailed)
			_ = dao.SavedDBInfoUpdate(oldSavedDBInfo)
			errMsg := "Error occurred while getting database information." +
				" (ConnectionID = " + connectionInfo.ID + ", Error = " + err.Error() + ")"
			logger.Println(logger.ERROR, false, errMsg)
			return nil, errors.New(errMsg)
		}
		collected, skipped = len(agentResult.Databases), len(agentResult.Skipped)
		for _, s := range agentResult.Skipped {
			reasons = append(reasons, s.Database+": "+s.Error)
		}
	} else {
		// Direct access: the server connects to the DBMS itself.
		loc := transxex.DBMSLocation{
			DBMSType: connectionInfo.DBType,
			Database: connectionInfo.DBName,
			// Empty unless the connection named schemas, and the connection can
			// only name them alongside a database (checkPgSchema) — so the
			// enumeration below never carries a selection into databases it was not
			// meant for.
			PgSchema: connectionInfo.DBPgSchema,
			// Where the database is hosted. A db source group always carries a
			// provider name, and transx-ex records it without reading it: no
			// routing, validation or connection decision depends on it. It is
			// filled because dmdl's DBConnConfig requires it and cm-centipede needs
			// the source's hosting location when choosing a target. An empty value
			// would mean "unspecified", which is not the same as on-premises.
			ProviderName: sourceGroup.ProviderName,
			AccessType:   transxex.AccessTypeDirect,
			Direct: &transxex.DirectConfig{
				Host:           connectionInfo.DBHost,
				Port:           port,
				Username:       connectionInfo.DBUsername,
				Password:       connectionInfo.DBPassword,
				ConnectTimeout: connectionInfo.DBConnectTimeout,
				TLSMode:        connectionInfo.DBTLSMode,
				TLSCAPEM:       connectionInfo.DBTLSCAPEM,
				AuthSource:     connectionInfo.DBAuthSource,
			},
			Metric: metric,
		}

		var databases []transxex.DBMSInfo
		var inspectErrors []transxex.DBMSInspectError

		inspectStart := time.Now()
		if connectionInfo.DBName == "" {
			// An empty db_name means every user database. The third return value is
			// non-nil only when nothing at all could be read — a single database
			// failing lands in the second one instead.
			databases, inspectErrors, err = transxex.InspectDBMSAll(loc)
			if err != nil {
				oldSavedDBInfo.Status = failedUnlessStale(oldSavedDBInfo.Status,
					model.SavedDBInfoStatusStale, model.SavedDBInfoStatusFailed)
				_ = dao.SavedDBInfoUpdate(oldSavedDBInfo)
				errMsg := "Error occurred while enumerating the databases." +
					" Set db_name on the connection to inspect a single database instead." +
					" (ConnectionID = " + connectionInfo.ID + ", Error = " + err.Error() + ")"
				logger.Println(logger.ERROR, false, errMsg)
				return nil, errors.New(errMsg)
			}
		} else {
			info, err := transxex.InspectDBMS(loc)
			if err != nil {
				oldSavedDBInfo.Status = failedUnlessStale(oldSavedDBInfo.Status,
					model.SavedDBInfoStatusStale, model.SavedDBInfoStatusFailed)
				_ = dao.SavedDBInfoUpdate(oldSavedDBInfo)
				errMsg := "Error occurred while getting database information." +
					" (ConnectionID = " + connectionInfo.ID + ", Error = " + err.Error() + ")"
				logger.Println(logger.ERROR, false, errMsg)
				return nil, errors.New(errMsg)
			}
			// Wrapped in a slice so a single target looks like an enumeration of one.
			databases = []transxex.DBMSInfo{*info}
		}
		logImportStep("db", connectionInfo.ID, "direct inspect", inspectStart,
			strconv.Itoa(len(databases))+" databases, "+strconv.Itoa(len(inspectErrors))+" skipped")

		// Marshalled from the library types so that both paths store the bytes the
		// agent would have produced. model.DBInspectResult mirrors this shape field
		// for field, so get and refined parse either one the same way and no
		// per-field conversion of the twelve inventories and three nesting levels
		// has to exist.
		encodedResult, err := json.Marshal(struct {
			Databases []transxex.DBMSInfo         `json:"databases"`
			Skipped   []transxex.DBMSInspectError `json:"skipped,omitempty"`
		}{Databases: databases, Skipped: inspectErrors})
		if err != nil {
			oldSavedDBInfo.Status = failedUnlessStale(oldSavedDBInfo.Status,
				model.SavedDBInfoStatusStale, model.SavedDBInfoStatusFailed)
			_ = dao.SavedDBInfoUpdate(oldSavedDBInfo)
			errMsg := "Error occurred while encoding the database information." +
				" (ConnectionID = " + connectionInfo.ID + ")"
			logger.Println(logger.ERROR, false, errMsg)
			return nil, errors.New(errMsg)
		}
		data = string(encodedResult)
		collected, skipped = len(databases), len(inspectErrors)
		for _, e := range inspectErrors {
			reasons = append(reasons, e.Database+": "+e.Message)
		}
	}

	status := dbImportStatus(collected, skipped)
	if status == model.SavedDBInfoStatusFailed {
		// Recorded and then returned as an error: a collection that read nothing is
		// not a 200 with an empty list. The result is still stored, since its
		// skipped list is the only full record of why each database failed —
		// except over a stale row, whose data has to keep describing the
		// selection it was collected for.
		if oldSavedDBInfo.Status != model.SavedDBInfoStatusStale {
			oldSavedDBInfo.DBData = data
		}
		oldSavedDBInfo.Status = failedUnlessStale(oldSavedDBInfo.Status,
			model.SavedDBInfoStatusStale, status)
		_ = dao.SavedDBInfoUpdate(oldSavedDBInfo)
		errMsg := "No database could be inspected." +
			" Set db_name on the connection to inspect a single database instead." +
			" (ConnectionID = " + connectionInfo.ID +
			", Skipped = " + agentResponseExcerpt(strings.Join(reasons, "; ")) + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	oldSavedDBInfo.DBData = data
	oldSavedDBInfo.Status = status
	oldSavedDBInfo.SavedTime = time.Now()
	err = dao.SavedDBInfoUpdate(oldSavedDBInfo)
	if err != nil {
		errMsg := "Error occurred while saving the database information." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	return oldSavedDBInfo, nil
}

func doGetDBInfo(connID string) (*model.DBInspectResult, error) {
	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, err
	}

	savedDBInfo, err := dao.SavedDBInfoGet(connectionInfo.ID)
	if err != nil {
		errMsg := "Failed to get information of the database." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	// A stale result was collected under a db_name or db_pg_schema selection this
	// connection no longer has, so answering with it would describe a different
	// scope. DBInspectResult carries no status field, which leaves the consumer
	// no way to notice — hence a refusal rather than a flag.
	//
	// "partial" is deliberately NOT refused: it is a result from THIS scope with
	// some databases missing, which the skipped list explains. Only "stale" means
	// "a result from a different scope". The row itself is kept either way, and
	// the databases[].database names inside db_data say which scope it was.
	if savedDBInfo.Status == model.SavedDBInfoStatusStale {
		errMsg := "The database information was collected for a different database or schema selection." +
			" Run the import again. (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	var dbInspectResult model.DBInspectResult
	err = json.Unmarshal([]byte(savedDBInfo.DBData), &dbInspectResult)
	if err != nil {
		errMsg := "Error occurred while parsing database information." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	return &dbInspectResult, nil
}

// doGetDBInfoRefined turns one connection into one refined model carrying every
// database it inspected. A connection points at a SERVER, so "one server plus a
// list of databases" is the shape both the collection above and the consumer
// downstream already use — the databases are passed through rather than
// flattened, which is why this returns a single SourceDBModel and not a slice.
//
// A partial collection still refines what it did read; the skipped entries are a
// collection diagnostic, not part of the refined model. ConnectionRef covers the
// connection sources of every consumer of the model, so only the honeybee one is
// filled here; the rest are omitempty and never reach the response.
func doGetDBInfoRefined(connID string) (*sourcemodel.SourceDBModel, error) {
	dbInspectResult, err := doGetDBInfo(connID)
	if err != nil {
		return nil, err
	}

	return &sourcemodel.SourceDBModel{
		Connection: commonmodel.ConnectionRef{
			Source:   commonmodel.ConnectionSourceHoneybee,
			Honeybee: &commonmodel.HoneybeeRef{ConnectionID: connID},
		},
		Databases: dbInspectResult.Databases,
	}, nil
}

// ImportDB godoc
//
//	@ID				import-db
//	@Summary		Import database
//	@Description	Import the database information of the connection information. With no db_name on the connection, every user database on the server is inspected.
//	@Tags			[Import] Import source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Param			ImportDBReq body model.ImportDBReq false "Metric selection."
//	@Success		200	{object}	model.SavedDBInfo		"Successfully saved the database information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to save the database information"
//	@Router			/source_group/{sgId}/connection_info/{connId}/import/db [post]
func ImportDB(c echo.Context) error {
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

	importDBReq := new(model.ImportDBReq)
	err = c.Bind(importDBReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	savedDBInfo, err := doImportDB(connID, *importDBReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, savedDBInfo, " ")
}

// ImportDBSourceGroup godoc
//
//	@ID				import-db-source-group
//	@Summary		Import database Source Group
//	@Description	Import database information for all connections in the source group. Each connection is inspected at its own db_name and db_pg_schema.
//	@Tags			[Import] Import source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			ImportDBReq body model.ImportDBReq false "Metric selection applied to every connection."
//	@Success		200	{object}	[]model.SavedDBInfo		"Successfully saved the database information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to save the database information"
//	@Router			/source_group/{sgId}/import/db [post]
func ImportDBSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	importDBReq := new(model.ImportDBReq)
	err = c.Bind(importDBReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var savedDBInfoList []model.SavedDBInfo

	// Sequential on purpose. One connection here is a whole database server, and
	// an enumeration opens a connection per database on it, so running the group
	// in parallel multiplies that against every member at once. A failure fails
	// the whole group, as the other group imports do; the ErrOnpremK8sAgent skip
	// they carry has no counterpart here, a db group holding no k8s connection.
	for _, conn := range *list {
		savedDBInfo, err := doImportDB(conn.ID, *importDBReq)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		savedDBInfoList = append(savedDBInfoList, *savedDBInfo)
	}

	return c.JSONPretty(http.StatusOK, savedDBInfoList, " ")
}

// GetDBInfo godoc
//
//	@ID				get-db-info
//	@Summary		Get Database Information
//	@Description	Get the database information of the connection information.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	model.DBInspectResult	"Successfully get information of the database."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the database."
//	@Router			/source_group/{sgId}/connection_info/{connId}/db [get]
func GetDBInfo(c echo.Context) error {
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

	dbInspectResult, err := doGetDBInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, dbInspectResult, " ")
}

// GetDBInfoSourceGroup godoc
//
//	@ID				get-db-info-source-group
//	@Summary		Get Database Information Source Group
//	@Description	Get the database information for all connections in the source group.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	model.DBInfoList		"Successfully get information of the database."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the database."
//	@Router			/source_group/{sgId}/db [get]
func GetDBInfoSourceGroup(c echo.Context) error {
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

	var dbInfoList model.DBInfoList

	for _, conn := range *list {
		dbInspectResult, err := doGetDBInfo(conn.ID)
		if err != nil || dbInspectResult == nil {
			// Not collected yet, unreadable, or stale — skip (logged in
			// doGetDBInfo). A stale result describes a selection the connection no
			// longer has, so leaving it out is the same answer as "nothing
			// collected". A partial one is NOT skipped: it describes this selection.
			continue
		}
		dbInfoList.Servers = append(dbInfoList.Servers, *dbInspectResult)
	}

	return c.JSONPretty(http.StatusOK, dbInfoList, " ")
}

// GetDBInfoRefined godoc
//
//	@ID				get-db-info-refined
//	@Summary		Get Refined Database Information
//	@Description	Get the refined database information of the connection information.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	sourcemodel.SourceDataMigrationModel	"Successfully get refined information of the database."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the database."
//	@Router			/source_group/{sgId}/connection_info/{connId}/db/refined [get]
func GetDBInfoRefined(c echo.Context) error {
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

	sourceDB, err := doGetDBInfoRefined(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	// A single connection still returns the wrapper, so a consumer parses the
	// same shape at both levels — as GetInfraInfoRefined does for one node.
	var sourceDataMigrationModel sourcemodel.SourceDataMigrationModel
	sourceDataMigrationModel.SourceDataMigrationModel.Databases =
		append(sourceDataMigrationModel.SourceDataMigrationModel.Databases, *sourceDB)

	return c.JSONPretty(http.StatusOK, sourceDataMigrationModel, " ")
}

// GetDBInfoSourceGroupRefined godoc
//
//	@ID				get-db-info-source-group-refined
//	@Summary		Get Refined Database Information Source Group
//	@Description	Get the refined database information for all connections in the source group.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	sourcemodel.SourceDataMigrationModel	"Successfully get refined information of the database."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the database."
//	@Router			/source_group/{sgId}/db/refined [get]
func GetDBInfoSourceGroupRefined(c echo.Context) error {
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
		sourceDB, err := doGetDBInfoRefined(conn.ID)
		if err != nil || sourceDB == nil {
			continue // not collected yet / unreadable / stale — skip (logged in doGetDBInfo)
		}
		sourceDataMigrationModel.SourceDataMigrationModel.Databases =
			append(sourceDataMigrationModel.SourceDataMigrationModel.Databases, *sourceDB)
	}

	return c.JSONPretty(http.StatusOK, sourceDataMigrationModel, " ")
}

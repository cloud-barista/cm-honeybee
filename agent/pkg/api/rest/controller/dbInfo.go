package controller

import (
	"net/http"

	transxex "github.com/cloud-barista/cm-centipede/transx-ex"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/common"
	"github.com/labstack/echo/v4"
)

// DBInspectResult is the response of the database inspect. Databases is always a
// list, even when a single database was targeted, so the honeybee server never
// has to handle two shapes. Skipped names the databases an enumeration could not
// read, which is how a partly successful collection reports itself.
//
// The fields are the library types verbatim, which is what keeps this in step
// with the server's model.DBInspectResult: the two are separate declarations in
// separate modules and only their JSON has to agree.
type DBInspectResult struct {
	Databases []transxex.DBMSInfo         `json:"databases"`
	Skipped   []transxex.DBMSInspectError `json:"skipped,omitempty"`
}

// DBInspectReq is the body of POST /db.
//
// The credentials belong in the body, not in a query string: the honeybee
// server reaches this endpoint by running curl on this host over its SSH
// channel, and anything in that command line is visible in the host's process
// list and in this agent's own access log. Percent-encoding would only keep a
// query from breaking the command line; it would not keep the value out of
// either log, so it is not an argument for passing these as query parameters.
//
// The numeric and structured fields arrive typed, so nothing here parses a
// string into a port, a timeout, a schema list or a metric selection.
type DBInspectReq struct {
	// DBType is the engine: mysql, mariadb, postgresql or mongodb. Required.
	DBType string `json:"db_type"`
	// DBName is the database to inspect. Empty means every user database on the
	// server.
	DBName string `json:"db_name,omitempty"`
	// Host of the database server. Required.
	Host string `json:"host"`
	// Port of the database server. 0 means the engine's default.
	Port int `json:"port,omitempty"`
	// Username of the database user.
	Username string `json:"username,omitempty"`
	// Password of the database user.
	Password string `json:"password,omitempty"`
	// ConnectTimeout in seconds. 0 means the driver default.
	ConnectTimeout int `json:"connect_timeout,omitempty"`
	// TLSMode is the transport security: disable, prefer, require, verify-ca or
	// verify-full. Empty means disable.
	TLSMode string `json:"tls_mode,omitempty"`
	// TLSCAPEM is the root CA in PEM form for verify-ca and verify-full. Empty
	// means the system trust store.
	TLSCAPEM string `json:"tls_ca_pem,omitempty"`
	// AuthSource is the authentication database. MongoDB only; empty means the
	// driver default.
	AuthSource string `json:"auth_source,omitempty"`
	// PgSchema is the schema selection. PostgreSQL only; omit to inspect every
	// user schema.
	PgSchema []string `json:"pg_schema,omitempty"`
	// ProviderName is where the database is hosted, as the source group records
	// it. Carried through to the result; nothing reads it.
	ProviderName string `json:"provider_name,omitempty"`
	// Metric is the metric selection, already routed to this engine by the
	// server.
	Metric *transxex.DBMSMetricOption `json:"metric,omitempty"`
}

// PostDBInfo godoc
//
//	@ID				post-db-info
//	@Summary		Get database information
//	@Description	Inspect a database server reachable from this host. With no db_name, every user database is inspected.
//	@Tags			[Data] Get data migration info
//	@Accept			json
//	@Produce		json
//	@Param			DBInspectReq body DBInspectReq true "Database inspect request."
//	@Success		200	{object}	controller.DBInspectResult	"Successfully get database information."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get database information."
//	@Router			/db [post]
func PostDBInfo(c echo.Context) error {
	var req DBInspectReq
	// The body is the whole request, so a body that will not parse is a bad
	// request rather than an internal error.
	if err := c.Bind(&req); err != nil {
		return common.ReturnErrorMsg(c, "Please provide a valid request body.")
	}

	if req.DBType == "" {
		return common.ReturnErrorMsg(c, "Please provide the db_type.")
	}

	if req.Host == "" {
		return common.ReturnErrorMsg(c, "Please provide the host.")
	}

	loc := transxex.DBMSLocation{
		DBMSType: req.DBType,
		Database: req.DBName,
		PgSchema: req.PgSchema,
		// Where the database is hosted, taken from the source group the server
		// resolved it from. transx-ex never reads it — it records what the caller
		// knows so that "where does this data live" has an answer.
		ProviderName: req.ProviderName,
		AccessType:   transxex.AccessTypeDirect,
		Direct: &transxex.DirectConfig{
			Host:           req.Host,
			Port:           req.Port,
			Username:       req.Username,
			Password:       req.Password,
			ConnectTimeout: req.ConnectTimeout,
			// Not validated here: InspectDBMS and InspectDBMSAll both run
			// validateLocation, which rejects an unknown mode, a CA given to a
			// mode that verifies nothing, both CA forms at once, MongoDB with
			// prefer, and content that is not PEM — all before any connection is
			// attempted.
			TLSMode:    req.TLSMode,
			TLSCAPEM:   req.TLSCAPEM,
			AuthSource: req.AuthSource,
		},
		Metric: req.Metric,
	}

	// An empty db_name is the signal to enumerate, matching the server's rule. A
	// database that fails on its own lands in Skipped instead of failing the call;
	// the error means nothing at all could be inspected.
	if loc.Database == "" {
		databases, skipped, err := transxex.InspectDBMSAll(loc)
		if err != nil {
			return common.ReturnInternalError(c, err, "Failed to get database information.")
		}
		return c.JSONPretty(http.StatusOK, DBInspectResult{Databases: databases, Skipped: skipped}, " ")
	}

	info, err := transxex.InspectDBMS(loc)
	if err != nil {
		return common.ReturnInternalError(c, err, "Failed to get database information.")
	}

	// Wrapped in a slice so a single target looks like an enumeration of one.
	return c.JSONPretty(http.StatusOK, DBInspectResult{Databases: []transxex.DBMSInfo{*info}}, " ")
}

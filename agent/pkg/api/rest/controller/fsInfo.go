package controller

import (
	"net/http"

	datamig "github.com/cloud-barista/cm-centipede/data-mig"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/common"
	"github.com/labstack/echo/v4"
)

// FSInspectReq is the body of POST /fs.
//
// Every field is omitempty, and an omitted field means "keep the default"
// rather than "use the zero value". That is why the honeybee server sends the
// scan depth unconditionally: a 0 disappears from the JSON, and a missing depth
// is read here as unlimited.
type FSInspectReq struct {
	// Path to inspect. Empty means "/".
	Path string `json:"path,omitempty"`
	// MaxDepth is the maximum scan depth. 0 means unlimited.
	MaxDepth int `json:"max_depth,omitempty"`
	// Metric selects the aggregates to compute. Omit to keep the library
	// defaults, which are not all false (folder mtime and permissions are on).
	Metric *datamig.FilesystemMetric `json:"metric,omitempty"`
}

// PostFSInfo godoc
//
//	@ID				post-fs-info
//	@Summary		Get filesystem information
//	@Description	Inspect a local filesystem path and return its folder listing together with the requested aggregates.
//	@Tags			[Data] Get data migration info
//	@Accept			json
//	@Produce		json
//	@Param			FSInspectReq body FSInspectReq true "Filesystem inspect request."
//	@Success		200	{object}	datamig.FSInspectResult	"Successfully get filesystem information."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get filesystem information."
//	@Router			/fs [post]
func PostFSInfo(c echo.Context) error {
	var req FSInspectReq
	// The body is the whole request, so a body that will not parse is a bad
	// request rather than an internal error.
	if err := c.Bind(&req); err != nil {
		return common.ReturnErrorMsg(c, "Please provide a valid request body.")
	}

	path := req.Path
	if path == "" {
		path = "/"
	}

	// A nil metric stays nil so the library keeps its own defaults; wrapping a
	// zeroed struct instead would turn folder mtime and permissions off.
	var metric *datamig.StorageMetricOption
	if req.Metric != nil {
		metric = &datamig.StorageMetricOption{Filesystem: req.Metric}
	}

	// Only MaxDepth is carried over. The scan depth is part of this API, the
	// include/exclude pipeline is not.
	loc := datamig.StorageLocation{
		StorageType: datamig.StorageTypeFilesystem,
		Path:        path,
		Filesystem:  &datamig.FilesystemAccess{AccessType: datamig.AccessTypeLocal},
		Filter:      &datamig.PathFilterOption{MaxDepth: req.MaxDepth},
		Metric:      metric,
	}

	result, err := datamig.InspectFilesystem(loc)
	if err != nil {
		return common.ReturnInternalError(c, err, "Failed to get filesystem information.")
	}

	return c.JSONPretty(http.StatusOK, result, " ")
}

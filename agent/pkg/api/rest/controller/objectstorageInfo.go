package controller

import (
	"net/http"
	"strings"

	transxex "github.com/cloud-barista/cm-centipede/transx-ex"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/common"
	"github.com/labstack/echo/v4"
)

// ObjectStorageInspectReq is the body of POST /objectstorage.
//
// The credentials belong in the body, not in a query string: the honeybee
// server reaches this endpoint by running curl on this host over its SSH
// channel, and anything in that command line is visible in the host's process
// list and in this agent's own access log.
//
// Nothing here is per-provider. The server resolves the endpoint, signing
// region, TLS choice and addressing style from its provider table and passes
// the results down, so this handler holds no knowledge of who is being talked
// to.
type ObjectStorageInspectReq struct {
	// Bucket to inspect. Required.
	Bucket string `json:"bucket"`
	// Prefix is the key prefix to scan under the bucket.
	Prefix string `json:"prefix,omitempty"`
	// Endpoint is the S3-compatible endpoint host. Required.
	Endpoint string `json:"endpoint"`
	// AccessKey is the access key id.
	AccessKey string `json:"access_key,omitempty"`
	// SecretKey is the secret access key.
	SecretKey string `json:"secret_key,omitempty"`
	// Region is the signing region. Empty is a choice, not an omission: minio-go
	// then asks the bucket for its location and signs with the answer.
	Region string `json:"region,omitempty"`
	// UseSSL tells whether to reach the endpoint over TLS.
	UseSSL bool `json:"use_ssl,omitempty"`
	// BucketLookup is the bucket addressing style: empty (auto), dns or path.
	BucketLookup string `json:"bucket_lookup,omitempty"`
	// Metric selects the aggregates to compute. Omit to keep the library
	// defaults, which are all off for object storage.
	Metric *transxex.ObjectStorageMetric `json:"metric,omitempty"`
}

// PostObjectStorageInfo godoc
//
//	@ID				post-object-storage-info
//	@Summary		Get object storage information
//	@Description	Inspect an S3-compatible bucket and return its prefix listing together with the requested aggregates.
//	@Tags			[Data] Get data migration info
//	@Accept			json
//	@Produce		json
//	@Param			ObjectStorageInspectReq body ObjectStorageInspectReq true "Object storage inspect request."
//	@Success		200	{object}	transxex.OSInspectResult	"Successfully get object storage information."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get object storage information."
//	@Router			/objectstorage [post]
func PostObjectStorageInfo(c echo.Context) error {
	var req ObjectStorageInspectReq
	// The body is the whole request, so a body that will not parse is a bad
	// request rather than an internal error.
	if err := c.Bind(&req); err != nil {
		return common.ReturnErrorMsg(c, "Please provide a valid request body.")
	}

	// Binding successfully and carrying a value are different things. Without
	// these two the library fails further in with a message that does not name
	// what was missing.
	if req.Bucket == "" {
		return common.ReturnErrorMsg(c, "Please provide the bucket.")
	}
	if req.Endpoint == "" {
		return common.ReturnErrorMsg(c, "Please provide the endpoint.")
	}

	var metric *transxex.StorageMetricOption
	if req.Metric != nil {
		metric = &transxex.StorageMetricOption{ObjectStorage: req.Metric}
	}

	loc := transxex.StorageLocation{
		StorageType: transxex.StorageTypeObjectStorage,
		// Object keys never start with a slash, so a leading one would make the
		// prefix match nothing; the server trims it at registration already.
		Path: req.Bucket + "/" + strings.TrimLeft(req.Prefix, "/"),
		ObjectStorage: &transxex.ObjectStorageAccess{
			AccessType: transxex.AccessTypeMinio,
			Minio: &transxex.S3MinioConfig{
				Endpoint:        req.Endpoint,
				AccessKeyId:     req.AccessKey,
				SecretAccessKey: req.SecretKey,
				Region:          req.Region,
				UseSSL:          req.UseSSL,
				BucketLookup:    req.BucketLookup,
			},
		},
		Metric: metric,
	}

	result, err := transxex.InspectObjectStorage(loc)
	if err != nil {
		return common.ReturnInternalError(c, err, "Failed to get object storage information.")
	}

	return c.JSONPretty(http.StatusOK, result, " ")
}

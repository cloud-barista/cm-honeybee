package controller

import (
	"net/http"

	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

// ListCSP godoc
//
//	@ID				list-csp
//	@Summary		List supported CSPs
//	@Description	Return the list of CSPs supported by the connected cb-spider.
//	@Tags			[CSP] Metadata
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	model.ListCSPRes		"List of CSP names"
//	@Failure		500	{object}	common.ErrorResponse	"Failed to query cb-spider"
//	@Router			/csp [get]
func ListCSP(c echo.Context) error {
	list, err := spider.ListCloudOS()
	if err != nil {
		return common.ReturnInternalError(c, err, "failed to list CSPs from cb-spider")
	}
	return c.JSONPretty(http.StatusOK, model.ListCSPRes{CSP: list}, " ")
}

// GetCSP godoc
//
//	@ID				get-csp
//	@Summary		Get CSP metadata
//	@Description	Return the credential keys, regions, and other metadata for the given CSP. Name is matched case-insensitively.
//	@Tags			[CSP] Metadata
//	@Accept			json
//	@Produce		json
//	@Param			name path string true "CSP name (case-insensitive, e.g. aws or AWS)"
//	@Success		200	{object}	model.CSPInfo			"CSP metadata"
//	@Failure		400	{object}	common.ErrorResponse	"Unsupported or missing CSP name"
//	@Failure		500	{object}	common.ErrorResponse	"Failed to query cb-spider"
//	@Router			/csp/{name} [get]
func GetCSP(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return common.ReturnErrorMsg(c, "Please provide the CSP name.")
	}

	canonical, err := spider.NormalizeProvider(name)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	meta, err := spider.GetCloudOSMetaInfo(canonical)
	if err != nil {
		return common.ReturnInternalError(c, err, "failed to get CSP metainfo from cb-spider")
	}

	defaultRegion := ""
	if len(meta.DefaultRegionToQuery) > 0 {
		defaultRegion = meta.DefaultRegionToQuery[0]
	}

	return c.JSONPretty(http.StatusOK, model.CSPInfo{
		Name:           canonical,
		CredentialKeys: meta.Credential,
		Regions:        meta.Region,
		DefaultRegion:  defaultRegion,
	}, " ")
}

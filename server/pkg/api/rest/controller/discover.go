package controller

import (
	"errors"
	"net/http"
	"strings"

	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

// DiscoverSourceGroupResources godoc
//
//	@ID				discover-source-group-resources
//	@Summary		Discover CSP resources for a SourceGroup
//	@Description	Lists VMs / K8s clusters / object-storage buckets reachable through the CSP connection bound to this SourceGroup. Used by the UI to populate ConnectionInfo selection.
//	@Tags			[CSP] Discovery
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup (must be type=csp)"
//	@Param			resource_type query string true "Resource type to discover (vm | k8s | object_storage)"
//	@Success		200	{object}	model.DiscoverRes		"Discovered resources"
//	@Failure		400	{object}	common.ErrorResponse	"Invalid request"
//	@Failure		500	{object}	common.ErrorResponse	"Discovery failed"
//	@Router			/source_group/{sgId}/discover [get]
func DiscoverSourceGroupResources(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sg, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if sg.Type != serverCommon.SourceGroupTypeCSP {
		return common.ReturnErrorMsg(c, "discovery is only supported for csp-type source groups.")
	}

	resourceType := strings.ToLower(strings.TrimSpace(c.QueryParam("resource_type")))
	if resourceType == "" {
		return common.ReturnErrorMsg(c, "resource_type query is required (vm | k8s | object_storage).")
	}

	// Register a temporary cb-spider connection for the duration of the discovery
	// call only — credentials are never persisted in cb-spider.
	var items []model.DiscoveredResource
	err = withSpiderConnection(sg, func(connName string) error {
		var derr error
		items, derr = discoverByType(connName, resourceType)
		return derr
	})
	if err != nil {
		return common.ReturnInternalError(c, err, "discovery failed")
	}
	return c.JSONPretty(http.StatusOK, model.DiscoverRes{Items: items}, " ")
}

func discoverByType(connName, resourceType string) ([]model.DiscoveredResource, error) {
	switch resourceType {
	case serverCommon.ResourceTypeVM:
		vms, err := spider.ListVM(connName)
		if err != nil {
			return nil, err
		}
		out := make([]model.DiscoveredResource, 0, len(vms))
		for _, vm := range vms {
			out = append(out, model.DiscoveredResource{
				ResourceType: serverCommon.ResourceTypeVM,
				ResourceID:   pickIIDName(vm.IId),
				Name:         vm.IId.NameId,
				Region:       vm.Region.RegionName,
				Extra: map[string]string{
					"vm_spec":   vm.VMSpecName,
					"public_ip": vm.PublicIP,
				},
			})
		}
		return out, nil
	case serverCommon.ResourceTypeK8s:
		clusters, err := spider.ListCluster(connName)
		if err != nil {
			return nil, err
		}
		out := make([]model.DiscoveredResource, 0, len(clusters))
		for _, cl := range clusters {
			out = append(out, model.DiscoveredResource{
				ResourceType: serverCommon.ResourceTypeK8s,
				ResourceID:   pickIIDName(cl.IId),
				Name:         cl.IId.NameId,
				Extra: map[string]string{
					"version": cl.Version,
					"status":  cl.Status,
				},
			})
		}
		return out, nil
	case serverCommon.ResourceTypeObjectStorage:
		buckets, err := spider.ListS3Buckets(connName)
		if err != nil {
			return nil, err
		}
		out := make([]model.DiscoveredResource, 0, len(buckets))
		for _, b := range buckets {
			out = append(out, model.DiscoveredResource{
				ResourceType: serverCommon.ResourceTypeObjectStorage,
				ResourceID:   b.Name,
				Name:         b.Name,
				Extra: map[string]string{
					"creation_date": b.CreationDate,
				},
			})
		}
		return out, nil
	default:
		return nil, errors.New("unsupported resource_type: " + resourceType + " (expected vm | k8s | object_storage)")
	}
}

func pickIIDName(iid spider.IID) string {
	if iid.NameId != "" {
		return iid.NameId
	}
	return iid.SystemId
}

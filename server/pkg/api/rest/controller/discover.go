package controller

import (
	"errors"
	"net/http"
	"strconv"
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
//	@Description	Lists VMs / K8s clusters / object-storage buckets / network load balancers reachable through the CSP connection bound to this SourceGroup. Used by the UI to populate ConnectionInfo selection.
//	@Tags			[CSP] Discovery
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup (must be type=csp)"
//	@Param			resource_type query string true "Resource type to discover (vm | k8s | object_storage | nlb)"
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
		return common.ReturnErrorMsg(c, "resource_type query is required (vm | k8s | object_storage | nlb).")
	}

	// Register a temporary cb-spider connection for the duration of the discovery
	// call only - credentials are never persisted in cb-spider.
	var items []model.DiscoveredResource
	err = withSpiderConnection(sg, "", func(connName string) error {
		var derr error
		items, derr = discoverByType(connName, resourceType)
		return derr
	})
	if err != nil {
		// A driver that has no handler for this resource type is not a failure of
		// the request - report it as such instead of leaking cb-spider's 500.
		var unsupported errDriverUnsupported
		if errors.As(err, &unsupported) {
			return c.JSONPretty(http.StatusOK, model.DiscoverRes{
				Items:       []model.DiscoveredResource{},
				Unsupported: true,
				Reason:      unsupported.Error(),
			}, " ")
		}
		return common.ReturnInternalError(c, err, "discovery failed")
	}
	return c.JSONPretty(http.StatusOK, model.DiscoverRes{Items: items}, " ")
}

// errDriverUnsupported marks a resource type the connection's driver implements
// no handler for. It travels up from discoverByType so the HTTP layer can answer
// 200 + Unsupported rather than 500.
type errDriverUnsupported struct{ msg string }

func (e errDriverUnsupported) Error() string { return e.msg }

func discoverByType(connName, resourceType string) ([]model.DiscoveredResource, error) {
	switch resourceType {
	case serverCommon.ResourceTypeVM:
		// ListAllVMInfo, not ListVM: the latter lists only what cb-spider
		// manages, which never includes a source VM cb-spider did not create.
		vms, err := spider.ListAllVMInfo(connName)
		if err != nil {
			return nil, err
		}
		out := make([]model.DiscoveredResource, 0, len(vms))
		for _, vm := range vms {
			out = append(out, model.DiscoveredResource{
				ResourceType: serverCommon.ResourceTypeVM,
				ResourceID:   pickIIDSystem(vm.IId),
				Name:         vm.IId.NameId,
				// VMInfo.Region carries the driver-level shape, which fills
				// Region/Zone and leaves RegionName empty - see RegionInfo.
				Region: firstNonEmpty(vm.Region.Region, vm.Region.RegionName),
				Extra: map[string]string{
					"vm_spec":   vm.VMSpecName,
					"public_ip": vm.PublicIP,
					"zone":      vm.Region.Zone,
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
				ResourceID:   pickIIDSystem(cl.IId),
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
	case serverCommon.ResourceTypeNLB:
		// Oracle's driver errors out in CreateNLBHandler(), which makes
		// /allnlbinfo answer 500. Ask what the driver supports first.
		capability, err := spider.GetDriverCapability(connName)
		if err != nil {
			return nil, err
		}
		if !capability.NLBHandler {
			return nil, errDriverUnsupported{
				msg: "this source group's CSP has no NLB support in its cb-spider driver",
			}
		}
		nlbs, err := spider.ListAllNLBInfo(connName)
		if err != nil {
			return nil, err
		}
		out := make([]model.DiscoveredResource, 0, len(nlbs))
		for _, n := range nlbs {
			// Reported verbatim: this is the survey step, and several drivers
			// hardcode or omit these fields. Normalising them is the collection
			// step's job (nlbInfoToNLB) - doing it in both places guarantees the
			// two drift apart.
			out = append(out, model.DiscoveredResource{
				ResourceType: serverCommon.ResourceTypeNLB,
				ResourceID:   pickIIDSystem(n.IId),
				Name:         n.IId.NameId,
				Extra: map[string]string{
					"type":  n.Type,
					"scope": n.Scope,
					// SystemId fallback: AWS builds VpcIID from the raw driver
					// value and sets only SystemId (irs.IID{SystemId: *nlbResInfo.VpcId}),
					// so reading NameId alone leaves this blank for every AWS NLB.
					"vpc":      pickIIDSystem(n.VpcIID),
					"listener": n.Listener.Protocol + "/" + n.Listener.Port,
					"endpoint": firstNonEmpty(n.Listener.DNSName, n.Listener.IP),
					"vm_count": strconv.Itoa(len(n.VMGroup.VMs)),
				},
			})
		}
		return out, nil
	default:
		return nil, errors.New("unsupported resource_type: " + resourceType + " (expected vm | k8s | object_storage | nlb)")
	}
}

// firstNonEmpty returns the first argument that is not blank.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// pickIIDSystem returns the CSP's own identifier for a resource - an AWS
// instance id, an ARN, an Azure ARM id - which is what ResourceID must carry:
// collection feeds that value straight back to cb-spider, and the CSP-native
// lookups reject anything else (AWS answers "InvalidInstanceID.Malformed" to a
// Name tag). NameId is a display name: mutable, not unique, and empty on some
// drivers, so it serves only as a fallback for drivers that leave SystemId blank.
func pickIIDSystem(iid spider.IID) string {
	if iid.SystemId != "" {
		return iid.SystemId
	}
	return iid.NameId
}

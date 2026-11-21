package controller

import (
	"errors"
	"net/http"

	inframodel "github.com/cloud-barista/cm-beetle/imdl/on-premise-model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

// GetBucketInfoRefined godoc
//
//	@ID				get-bucket-info-refined
//	@Summary		Get Refined Bucket Information
//	@Description	Get the bucket of a CSP object-storage connection in the source model cm-beetle recommends a target bucket from. The response is the same as data/refined of the connection. The values are those collected by import/bucket or import/infra. sourceCloud.region is the region the bucket was found in, or the source group's region where the CSP cannot tell.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	storagemodel.SourceObjectStorageModel	"Successfully get refined information of the bucket."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the bucket."
//	@Router			/source_group/{sgId}/connection_info/{connId}/bucket/refined [get]
func GetBucketInfoRefined(c echo.Context) error {
	return GetDataInfoRefined(c)
}

// GetBucketInfoSourceGroupRefined godoc
//
//	@ID				get-bucket-info-source-group-refined
//	@Summary		Get Refined Bucket Information Source Group
//	@Description	Get every collected bucket of the source group's CSP object-storage connections in the source model cm-beetle recommends target buckets from. The response is the same as data/refined of the group. Connections without a successfully collected bucket are left out, as are buckets found in a region other than the group's. sourceCloud.region is the group's region.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	storagemodel.SourceObjectStorageModel	"Successfully get refined information of the buckets."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the buckets."
//	@Router			/source_group/{sgId}/bucket/refined [get]
func GetBucketInfoSourceGroupRefined(c echo.Context) error {
	return GetDataInfoSourceGroupRefined(c)
}

// errK8sNotCollected means the connection has no Kubernetes cluster whose last
// collection succeeded.
var errK8sNotCollected = errors.New("no kubernetes cluster was collected successfully for this connection;" +
	" run import/k8s for a csp k8s connection, or import/kubernetes for a control plane node")

// refinedK8sCluster builds the k8s/refined response of a cluster taken from
// the connection. k8sCluster is built as infra/refined builds it; a managed
// cluster also carries the csp section of its stored record.
func refinedK8sCluster(connID string, info *kubernetes.Kubernetes, managed bool) model.RefinedK8sClusterModel {
	m := model.RefinedK8sClusterModel{
		ConnectionID: connID,
		Managed:      managed,
		K8sCluster:   buildK8sCluster(info),
	}
	if managed {
		if rec, err := doGetKubernetesInfo(connID); err == nil && rec != nil {
			m.CSP = rec.CSP
		}
	}

	return m
}

// GetK8sInfoRefined godoc
//
//	@ID				get-k8s-info-refined
//	@Summary		Get Refined K8s Information
//	@Description	Get the Kubernetes cluster collected from the connection: a CSP-managed cluster of a csp k8s connection, or the cluster the agent collected on a control plane node. k8sCluster is what infra/refined of the connection gives as k8sCluster. A managed cluster also carries a csp section with its node groups. A connection without a successfully collected cluster is a bad request.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	model.RefinedK8sClusterModel	"Successfully get refined information of the cluster."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the cluster."
//	@Router			/source_group/{sgId}/connection_info/{connId}/k8s/refined [get]
func GetK8sInfoRefined(c echo.Context) error {
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

	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	k8sInfo := tryGetKubernetesInfo(connID)
	if !hasK8sClusterData(k8sInfo) {
		return common.ReturnErrorMsg(c, errK8sNotCollected.Error()+" (ConnectionID = "+connID+")")
	}

	return c.JSONPretty(http.StatusOK,
		refinedK8sCluster(connID, k8sInfo, isCSPManagedK8s(sourceGroup, connectionInfo)), " ")
}

// GetK8sInfoSourceGroupRefined godoc
//
//	@ID				get-k8s-info-source-group-refined
//	@Summary		Get Refined K8s Information Source Group
//	@Description	Get the Kubernetes cluster of the source group, the one infra/refined of the group gives as k8sCluster. A cluster the agent collected is taken before a CSP-managed one; a managed one also carries a csp section with its node groups. A group without a successfully collected cluster gives k8sCluster null.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	model.RefinedK8sClusterModel	"Successfully get refined information of the cluster."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the cluster."
//	@Router			/source_group/{sgId}/k8s/refined [get]
func GetK8sInfoSourceGroupRefined(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connID, k8sInfo, managed := groupK8sClusterSource(sourceGroup, *list)
	if k8sInfo == nil {
		return c.JSONPretty(http.StatusOK, model.RefinedK8sClusterModel{}, " ")
	}

	return c.JSONPretty(http.StatusOK, refinedK8sCluster(connID, k8sInfo, managed), " ")
}

func refinedNLBModel(nlbs []inframodel.NlbProperty) model.RefinedNLBModel {
	if nlbs == nil {
		nlbs = []inframodel.NlbProperty{}
	}

	return model.RefinedNLBModel{NLBs: nlbs}
}

// GetNLBInfoRefined godoc
//
//	@ID				get-nlb-info-refined
//	@Summary		Get Refined NLB Information
//	@Description	Get the load balancers of the connection, the list infra/refined of the connection gives as nlbs: the NLBs found in HAProxy on a host, or the CSP NLB of a csp nlb connection with its backend servers given the private IPs of the group's CSP VMs. A host whose infra has not been collected is a bad request, as in infra/refined.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	model.RefinedNLBModel	"Successfully get refined information of the load balancers."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the load balancers."
//	@Router			/source_group/{sgId}/connection_info/{connId}/nlb/refined [get]
func GetNLBInfoRefined(c echo.Context) error {
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

	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	onpremiseInfra, err := connRefinedOnpremInfra(sourceGroup, connectionInfo)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, refinedNLBModel(onpremiseInfra.NLBs), " ")
}

// GetNLBInfoSourceGroupRefined godoc
//
//	@ID				get-nlb-info-source-group-refined
//	@Summary		Get Refined NLB Information Source Group
//	@Description	Get the load balancers of the source group, the list infra/refined of the group gives as nlbs: the NLBs found in HAProxy on the refined nodes, then the CSP NLBs of the csp nlb connections whose last import succeeded, their backend servers given the private IPs of the group's CSP VMs. A group without any gives an empty list.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	model.RefinedNLBModel	"Successfully get refined information of the load balancers."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the load balancers."
//	@Router			/source_group/{sgId}/nlb/refined [get]
func GetNLBInfoSourceGroupRefined(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	onpremiseInfra := groupRefinedOnpremInfra(sourceGroup, *list)

	return c.JSONPretty(http.StatusOK, refinedNLBModel(onpremiseInfra.NLBs), " ")
}

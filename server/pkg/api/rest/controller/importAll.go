package controller

import (
	"errors"
	"net/http"
	"strconv"

	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

// importStep is one collection a group import runs for a connection.
type importStep struct {
	kind string
	run  func(connID string) error
}

// connImportPlan is the collections a group import runs for one connection.
type connImportPlan struct {
	conn  model.ConnectionInfo
	steps []importStep
}

// errConnectionImportFailed is returned for a connection none of whose
// collections succeeded. It carries the entry of the connection, so the
// response still lists every collection with its own error.
type errConnectionImportFailed struct {
	result    model.ImportConnectionResult
	firstKind string
	first     error
}

func (e *errConnectionImportFailed) Error() string {
	return e.firstKind + " import failed: " + e.first.Error()
}

// importAllOptions are the request options of import/all: those of the infra
// import and of the software import it runs.
type importAllOptions struct {
	infra               importInfraOptions
	showDefaultPackages bool
}

func importAllOptionsFrom(c echo.Context) (importAllOptions, error) {
	infraOpts, err := importInfraOptionsFrom(c)
	if err != nil {
		return importAllOptions{}, err
	}
	showDefaultPackages, _ := strconv.ParseBool(c.QueryParam("show_default_packages"))

	return importAllOptions{infra: infraOpts, showDefaultPackages: showDefaultPackages}, nil
}

// cspImportKind names the kind a CSP collection of the resource type reports.
// A VM is collected as infra.
func cspImportKind(resourceType string) string {
	switch resourceType {
	case serverCommon.ResourceTypeK8s:
		return model.ImportKindKubernetes
	case serverCommon.ResourceTypeObjectStorage:
		return model.ImportKindBucket
	case serverCommon.ResourceTypeRDBMS:
		return model.ImportKindRDBMS
	case serverCommon.ResourceTypeNLB:
		return model.ImportKindNLB
	}

	return model.ImportKindInfra
}

// infraImportStep is the infra import of a connection. For a CSP connection
// it is the CSP collection of the resource, reported under the resource's kind.
func infraImportStep(sourceGroup *model.SourceGroup, conn *model.ConnectionInfo, opts importInfraOptions) importStep {
	kind := model.ImportKindInfra
	if serverCommon.IsCSPType(sourceGroup.Type) {
		kind = cspImportKind(conn.ResourceType)
	}

	return importStep{kind: kind, run: func(connID string) error {
		_, err := doImportInfra(connID, opts)
		return err
	}}
}

// agentImportSteps are the collections made through the agent.
func agentImportSteps(showDefaultPackages bool) []importStep {
	return []importStep{
		{kind: model.ImportKindSoftware, run: func(connID string) error {
			_, err := doImportSoftware(connID, showDefaultPackages)
			return err
		}},
		{kind: model.ImportKindKubernetes, run: func(connID string) error {
			_, err := doImportKubernetes(connID)
			return err
		}},
		{kind: model.ImportKindHelm, run: func(connID string) error {
			_, err := doImportHelm(connID)
			return err
		}},
	}
}

// importAllPlans picks what import/all collects of each connection, with the
// same target selection the group imports by kind use: the infra import of
// infraImportTargets, which is the CSP collection on a csp group, and the agent
// imports of agentImportTargets. A connection with nothing to collect, an
// on-prem k8s one, is left out.
func importAllPlans(sourceGroup *model.SourceGroup, conns []model.ConnectionInfo, opts importAllOptions) []connImportPlan {
	inInfra := map[string]bool{}
	for _, c := range infraImportTargets(sourceGroup, conns) {
		inInfra[c.ID] = true
	}
	inAgent := map[string]bool{}
	for _, c := range agentImportTargets(sourceGroup, conns) {
		inAgent[c.ID] = true
	}

	plans := []connImportPlan{}
	for i := range conns {
		conn := conns[i]
		var steps []importStep
		if inInfra[conn.ID] {
			steps = append(steps, infraImportStep(sourceGroup, &conn, opts.infra))
		}
		if inAgent[conn.ID] {
			steps = append(steps, agentImportSteps(opts.showDefaultPackages)...)
		}
		if len(steps) == 0 {
			continue
		}
		plans = append(plans, connImportPlan{conn: conn, steps: steps})
	}

	return plans
}

// cspKindImportPlans picks the csp connections of the resource type, each
// collected from the CSP.
func cspKindImportPlans(sourceGroup *model.SourceGroup, conns []model.ConnectionInfo, resourceType string,
	opts importInfraOptions) []connImportPlan {
	plans := []connImportPlan{}
	for i := range conns {
		conn := conns[i]
		if conn.ResourceType != resourceType {
			continue
		}
		plans = append(plans, connImportPlan{conn: conn,
			steps: []importStep{infraImportStep(sourceGroup, &conn, opts)}})
	}

	return plans
}

// runImportSteps runs every collection of the connection, a failing one not
// stopping the next. The connection fails only when none succeeded.
func runImportSteps(sourceGroup *model.SourceGroup, plan connImportPlan) (*model.ImportConnectionResult, error) {
	conn := plan.conn
	fillResourceType(sourceGroup, &conn)
	result := model.ImportConnectionResult{
		ConnectionID: conn.ID,
		ResourceType: conn.ResourceType,
		Results:      make([]model.ImportKindResult, 0, len(plan.steps)),
	}

	var failed *errConnectionImportFailed
	succeeded := 0
	for _, step := range plan.steps {
		if err := step.run(conn.ID); err != nil {
			result.Results = append(result.Results, model.ImportKindResult{
				Kind: step.kind, Status: model.ConnectionInfoStatusFailed, ErrorMessage: err.Error(),
			})
			if failed == nil {
				failed = &errConnectionImportFailed{firstKind: step.kind, first: err}
			}
			continue
		}
		succeeded++
		result.Results = append(result.Results, model.ImportKindResult{
			Kind: step.kind, Status: model.ConnectionInfoStatusSuccess,
		})
	}

	if succeeded == 0 && failed != nil {
		failed.result = result
		return nil, failed
	}

	return &result, nil
}

// runImportPlans runs the plans of a group import connection by connection
// through importEach: a failing connection is listed and the rest go on, and
// the import fails only when no connection had a collection succeed.
func runImportPlans(sourceGroup *model.SourceGroup, plans []connImportPlan) ([]model.ImportConnectionResult, error) {
	byID := make(map[string]connImportPlan, len(plans))
	targets := make([]model.ConnectionInfo, 0, len(plans))
	for _, p := range plans {
		byID[p.conn.ID] = p
		targets = append(targets, p.conn)
	}

	return importEach(targets, func(connID string) (*model.ImportConnectionResult, error) {
		return runImportSteps(sourceGroup, byID[connID])
	}, func(connID string, err error) model.ImportConnectionResult {
		var failed *errConnectionImportFailed
		if errors.As(err, &failed) {
			return failed.result
		}
		conn := byID[connID].conn
		fillResourceType(sourceGroup, &conn)
		return model.ImportConnectionResult{ConnectionID: connID, ResourceType: conn.ResourceType,
			Results: []model.ImportKindResult{}}
	})
}

// importAllNotApplicable explains why import/all does not take the source
// group, or returns nil when it does. It collects what a migration is planned
// from, the sources of onprem and csp groups. An fs, db or minio group is a
// data source whose own import gives the data migration model of cm-centipede,
// so the caller is pointed to it instead of getting an empty success.
func importAllNotApplicable(sourceGroup *model.SourceGroup) error {
	switch {
	case serverCommon.IsOnpremType(sourceGroup.Type), serverCommon.IsCSPType(sourceGroup.Type):
		return nil
	case serverCommon.IsFSType(sourceGroup.Type):
		return errors.New("import/all collects onprem and csp source groups; use import/fs for an fs source group")
	case serverCommon.IsDBType(sourceGroup.Type):
		return errors.New("import/all collects onprem and csp source groups; use import/db for a db source group")
	case serverCommon.IsMinIOType(sourceGroup.Type):
		return errors.New("import/all collects onprem and csp source groups; use import/objectstorage for a minio source group")
	}

	return errors.New("import/all collects onprem and csp source groups; source group type '" +
		sourceGroup.Type + "' is not supported")
}

// ImportAllSourceGroup godoc
//
//	@ID				import-all-source-group
//	@Summary		Import All Source Group
//	@Description	Collect every connection of an onprem or csp source group the way its type is collected. A csp resource other than a VM is collected from the CSP and reported as kubernetes, bucket, rdbms or nlb. A VM is collected as infra (on a csp group the CSP side, and through the agent when it has an ip_address), and a host the agent can reach also as software, kubernetes and helm, as the group imports of those kinds pick them. An on-prem k8s connection has nothing to collect and is left out. A failing collection is listed with status failed and does not stop the others; the request fails only when no collection of any connection succeeded. fs, db and minio groups are refused with the import to use instead (import/fs, import/db, import/objectstorage).
//	@Tags			[Import] Import source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			count_objects query bool false "For object-storage connections, count each bucket's objects and total size by listing every object. Default true."
//	@Param			show_default_packages query bool false "Enable for show all packages include default packages. (Default: false)"
//	@Success		200	{object}	[]model.ImportConnectionResult	"The collections run for each connection and their results"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to collect the source group"
//	@Router			/source_group/{sgId}/import/all [post]
func ImportAllSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	if err := importAllNotApplicable(sourceGroup); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	opts, err := importAllOptionsFrom(c)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	results, err := runImportPlans(sourceGroup, importAllPlans(sourceGroup, *list, opts))
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, results, " ")
}

// importCSPKindSourceGroup collects from the CSP the connections of one
// resource type of a csp source group. name is the kind as the route names it.
func importCSPKindSourceGroup(c echo.Context, name string, resourceType string, opts importInfraOptions) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	if !serverCommon.IsCSPType(sourceGroup.Type) {
		return common.ReturnErrorMsg(c, "import/"+name+" collects the "+resourceType+
			" connections of a csp source group; the source group type is '"+sourceGroup.Type+"'")
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	results, err := runImportPlans(sourceGroup, cspKindImportPlans(sourceGroup, *list, resourceType, opts))
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, results, " ")
}

// ImportBucketSourceGroup godoc
//
//	@ID				import-bucket-source-group
//	@Summary		Import Bucket Source Group
//	@Description	Collect from the CSP the object_storage connections of a csp source group. The other connections are left alone. A failing connection is listed with status failed and does not stop the others; the request fails only when every connection fails. A source group that is not csp is refused.
//	@Tags			[Import] Import source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			count_objects query bool false "Count each bucket's objects and total size by listing every object. Default true."
//	@Success		200	{object}	[]model.ImportConnectionResult	"The result of each bucket connection"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to collect the buckets"
//	@Router			/source_group/{sgId}/import/bucket [post]
func ImportBucketSourceGroup(c echo.Context) error {
	opts, err := importInfraOptionsFrom(c)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return importCSPKindSourceGroup(c, "bucket", serverCommon.ResourceTypeObjectStorage, opts)
}

// ImportRDBMSSourceGroup godoc
//
//	@ID				import-rdbms-source-group
//	@Summary		Import RDBMS Source Group
//	@Description	Collect from the CSP the rdbms connections (managed databases) of a csp source group. The other connections are left alone. A failing connection is listed with status failed and does not stop the others; the request fails only when every connection fails. A source group that is not csp is refused.
//	@Tags			[Import] Import source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	[]model.ImportConnectionResult	"The result of each rdbms connection"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to collect the managed databases"
//	@Router			/source_group/{sgId}/import/rdbms [post]
func ImportRDBMSSourceGroup(c echo.Context) error {
	return importCSPKindSourceGroup(c, "rdbms", serverCommon.ResourceTypeRDBMS, importInfraOptions{CountObjects: true})
}

// ImportK8sSourceGroup godoc
//
//	@ID				import-k8s-source-group
//	@Summary		Import K8s Source Group
//	@Description	Collect from the CSP the k8s connections (managed clusters) of a csp source group. The other connections are left alone; a cluster on VMs is collected through the agent by import/kubernetes. A failing connection is listed with status failed and does not stop the others; the request fails only when every connection fails. A source group that is not csp is refused.
//	@Tags			[Import] Import source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	[]model.ImportConnectionResult	"The result of each k8s connection"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to collect the managed clusters"
//	@Router			/source_group/{sgId}/import/k8s [post]
func ImportK8sSourceGroup(c echo.Context) error {
	return importCSPKindSourceGroup(c, "k8s", serverCommon.ResourceTypeK8s, importInfraOptions{CountObjects: true})
}

// ImportNLBSourceGroup godoc
//
//	@ID				import-nlb-source-group
//	@Summary		Import NLB Source Group
//	@Description	Collect from the CSP the nlb connections of a csp source group. The other connections are left alone. A failing connection is listed with status failed and does not stop the others; the request fails only when every connection fails. A source group that is not csp is refused.
//	@Tags			[Import] Import source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	[]model.ImportConnectionResult	"The result of each nlb connection"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to collect the load balancers"
//	@Router			/source_group/{sgId}/import/nlb [post]
func ImportNLBSourceGroup(c echo.Context) error {
	return importCSPKindSourceGroup(c, "nlb", serverCommon.ResourceTypeNLB, importInfraOptions{CountObjects: true})
}

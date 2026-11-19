package controller

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// ErrAgentNotApplicable is returned when an agent-backed import is asked for a
// CSP connection that has no in-guest agent to reach: a resource other than a
// VM, or a VM registered without an IP address. The import is refused before it
// touches any record, so data the CSP collection stored for the connection
// survives.
var ErrAgentNotApplicable = errors.New("agent-based collection is not applicable")

// isHostConnection reports whether the connection may stand for a machine, and
// so become a node of the refined infra. Only the connections known to hold
// none are excluded: an on-prem k8s cluster, reached by its kubeconfig, and a
// CSP resource other than a VM (a bucket, a database, a cluster or a load
// balancer). Connections of the other source group types are hosts as before.
func isHostConnection(sourceGroup *model.SourceGroup, connectionInfo *model.ConnectionInfo) bool {
	if isOnpremK8s(sourceGroup, connectionInfo) {
		return false
	}
	if serverCommon.IsCSPType(sourceGroup.Type) {
		return connectionInfo.ResourceType == serverCommon.ResourceTypeVM
	}

	return true
}

// agentNotApplicable explains why the connection cannot be collected through
// the agent, or returns nil when it can. The agent runs inside a VM and is
// reached over SSH. Only the connections known to lack one are refused: an
// on-prem k8s cluster, a CSP resource other than a VM, and a CSP VM registered
// without an IP address. Connections of the other source group types are left
// to the agent call as before.
func agentNotApplicable(sourceGroup *model.SourceGroup, connectionInfo *model.ConnectionInfo) error {
	if isOnpremK8s(sourceGroup, connectionInfo) {
		return ErrOnpremK8sAgent
	}

	if !serverCommon.IsCSPType(sourceGroup.Type) {
		return nil
	}

	if connectionInfo.ResourceType != serverCommon.ResourceTypeVM {
		return fmt.Errorf("%w: a csp %s connection has no host to run the agent on,"+
			" its CSP-side information is collected by import/infra", ErrAgentNotApplicable,
			connectionInfo.ResourceType)
	}

	if strings.TrimSpace(connectionInfo.IPAddress) == "" {
		return fmt.Errorf("%w: the csp vm connection has no ip_address to reach the agent over SSH,"+
			" its CSP-side information is collected by import/infra", ErrAgentNotApplicable)
	}

	return nil
}

// isAgentCollectable reports whether software, kubernetes, helm and data can be
// collected from the connection through the agent.
func isAgentCollectable(sourceGroup *model.SourceGroup, connectionInfo *model.ConnectionInfo) bool {
	return agentNotApplicable(sourceGroup, connectionInfo) == nil
}

// infraImportTargets picks the connections a group infra import collects. Every
// connection takes part except an on-prem k8s one, which has neither an agent
// nor a CSP behind it.
func infraImportTargets(sourceGroup *model.SourceGroup, conns []model.ConnectionInfo) []model.ConnectionInfo {
	targets := make([]model.ConnectionInfo, 0, len(conns))
	for i := range conns {
		if isOnpremK8s(sourceGroup, &conns[i]) {
			continue
		}
		targets = append(targets, conns[i])
	}

	return targets
}

// agentImportTargets picks the connections a group agent import collects.
// Calling the agent on any other connection only ends in an SSH failure.
func agentImportTargets(sourceGroup *model.SourceGroup, conns []model.ConnectionInfo) []model.ConnectionInfo {
	targets := make([]model.ConnectionInfo, 0, len(conns))
	for i := range conns {
		if !isAgentCollectable(sourceGroup, &conns[i]) {
			continue
		}
		targets = append(targets, conns[i])
	}

	return targets
}

// importEach runs one import per target and keeps going past a failing
// connection, so one unreachable source does not leave the rest of the group
// uncollected. A failed connection still gets an entry, built by failed from
// its error, so the response accounts for every target and says why. The first error is returned only when no
// target succeeded; no target at all is an empty result, not an error.
func importEach[T any](targets []model.ConnectionInfo, do func(connID string) (*T, error),
	failed func(connID string, err error) T) ([]T, error) {
	results := make([]T, 0, len(targets))
	var firstErr error
	succeeded := 0

	for _, conn := range targets {
		result, err := do(conn.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			results = append(results, failed(conn.ID, err))
			continue
		}
		succeeded++
		results = append(results, *result)
	}

	if len(targets) > 0 && succeeded == 0 {
		return nil, firstErr
	}

	return results, nil
}

// failedSavedInfraInfo is the entry of a connection whose infra import failed.
// It carries the stored record when there is one, so the response holds the
// same thing a later GET returns, with the status and the error of this import.
func failedSavedInfraInfo(connID string, importErr error) model.SavedInfraInfo {
	r := model.SavedInfraInfo{ConnectionID: connID}
	if rec, err := dao.SavedInfraInfoGet(connID); err == nil && rec != nil {
		r = *rec
	}
	r.Status = model.ConnectionInfoStatusFailed
	r.ErrorMessage = importErr.Error()

	return r
}

func failedSavedSoftwareInfo(connID string, importErr error) model.SavedSoftwareInfo {
	r := model.SavedSoftwareInfo{ConnectionID: connID}
	if rec, err := dao.SavedSoftwareInfoGet(connID); err == nil && rec != nil {
		r = *rec
	}
	r.Status = model.ConnectionInfoStatusFailed
	r.ErrorMessage = importErr.Error()

	return r
}

func failedSavedKubernetesInfo(connID string, importErr error) model.SavedKubernetesInfo {
	r := model.SavedKubernetesInfo{ConnectionID: connID}
	if rec, err := dao.SavedKubernetesInfoGet(connID); err == nil && rec != nil {
		r = *rec
	}
	r.Status = model.ConnectionInfoStatusFailed
	r.ErrorMessage = importErr.Error()

	return r
}

func failedSavedHelmInfo(connID string, importErr error) model.SavedHelmInfo {
	r := model.SavedHelmInfo{ConnectionID: connID}
	if rec, err := dao.SavedHelmInfoGet(connID); err == nil && rec != nil {
		r = *rec
	}
	r.Status = model.ConnectionInfoStatusFailed
	r.ErrorMessage = importErr.Error()

	return r
}

func failedSavedDataInfo(connID string, importErr error) model.SavedDataInfo {
	r := model.SavedDataInfo{ConnectionID: connID}
	if rec, err := dao.SavedDataInfoGet(connID); err == nil && rec != nil {
		r = *rec
	}
	r.Status = model.ConnectionInfoStatusFailed
	r.ErrorMessage = importErr.Error()

	return r
}

// connInfra is the stored infra of one connection that becomes a refined node.
type connInfra struct {
	connID string
	infra  *infra.Infra
}

// refinedInfraSources reads the infra of the group connections that become
// nodes of the refined infra. Only host connections are nodes: the record of a
// bucket, a database or a cluster holds no machine. A host that has not been
// collected, whose last import failed, or whose record cannot be read is left
// out, as the group GET infra does, instead of failing the whole group. The
// exception is a CSP VM whose CSP collection succeeded and only the agent call
// failed: it is a node of its CSP data alone.
func refinedInfraSources(sourceGroup *model.SourceGroup, conns []model.ConnectionInfo) []connInfra {
	var sources []connInfra

	for i := range conns {
		conn := &conns[i]
		if !isHostConnection(sourceGroup, conn) {
			continue
		}

		rec, err := dao.SavedInfraInfoGet(conn.ID)
		if err != nil || rec == nil {
			continue
		}

		var infraInfo *infra.Infra
		switch {
		case rec.Status == model.ConnectionInfoStatusSuccess:
			if strings.TrimSpace(rec.InfraData) == "" && strings.TrimSpace(rec.CSPData) == "" {
				continue
			}
			infraInfo, err = doGetInfraInfo(conn.ID)
		case cspNodeAfterAgentFailure(sourceGroup, conn, rec):
			infraInfo, err = cspOnlyInfra(rec.CSPData)
		default:
			continue
		}
		if err != nil || infraInfo == nil {
			continue
		}
		sources = append(sources, connInfra{connID: conn.ID, infra: infraInfo})
	}

	return sources
}

// connSoftware is the stored software of one connection.
type connSoftware struct {
	connID   string
	software *software.Software
}

// refinedSoftwareSources reads the software of the group connections the agent
// collected it from. A connection the agent cannot reach, or whose software
// has not been collected successfully, is left out.
func refinedSoftwareSources(sourceGroup *model.SourceGroup, conns []model.ConnectionInfo) []connSoftware {
	var sources []connSoftware

	for i := range conns {
		conn := &conns[i]
		if !isAgentCollectable(sourceGroup, conn) {
			continue
		}

		rec, err := dao.SavedSoftwareInfoGet(conn.ID)
		if err != nil || rec == nil {
			continue
		}
		if rec.Status != model.ConnectionInfoStatusSuccess {
			continue
		}

		softwareInfo, err := doGetSoftwareInfo(conn.ID)
		if err != nil || softwareInfo == nil {
			continue
		}
		sources = append(sources, connSoftware{connID: conn.ID, software: softwareInfo})
	}

	return sources
}

package controller

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/data"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// keyValueListToMap flattens a spider KeyValue list into a map for easy lookup.
func keyValueListToMap(in []spider.KeyValue) map[string]string {
	out := make(map[string]string, len(in))
	for _, kv := range in {
		out[kv.Key] = kv.Value
	}
	return out
}

// buildCSPInfo maps cb-spider's VMInfo into the CSP-side infra.CSPInfo: the
// provider-observable VM facts (spec, image, region/zone, public/private IP,
// disks, tags) plus the names of the attached VPC/subnet/security groups.
func buildCSPInfo(sg *model.SourceGroup, vm *spider.VMInfo) infra.CSPInfo {
	kvMap := keyValueListToMap(vm.KeyValueList)
	platform := vm.Platform
	if platform == "" {
		platform = kvMap["Architecture"]
	}

	rootDiskSize := uint(0)
	if v, err := strconv.ParseUint(strings.TrimSpace(vm.RootDiskSize), 10, 64); err == nil {
		rootDiskSize = uint(v)
	}

	dataDisks := make([]string, 0, len(vm.DataDiskIIDs))
	for _, d := range vm.DataDiskIIDs {
		dataDisks = append(dataDisks, d.NameId)
	}

	tags := map[string]string{}
	for _, t := range vm.TagList {
		tags[t.Key] = t.Value
	}

	csp := infra.CSPInfo{
		Provider:  sg.ProviderName,
		Region:    vm.Region.Region,
		Zone:      vm.Region.Zone,
		Name:      vm.IId.NameId,
		ID:        vm.IId.SystemId,
		VMSpec:    vm.VMSpecName,
		Image:     vm.ImageIId.NameId,
		Platform:  platform,
		PublicIP:  vm.PublicIP,
		PrivateIP: vm.PrivateIP,
		RootDisk: infra.Disk{
			Name: vm.RootDeviceName,
			Type: vm.RootDiskType,
			Size: rootDiskSize,
		},
		DataDisks: dataDisks,
		Tags:      tags,
		StartTime: vm.StartTime,
		Network: infra.CSPNetwork{
			Subnet: vm.SubnetIID.NameId,
		},
	}

	// VPC/subnet/security-group names come from the VM info. Their full detail
	// (VPC CIDR/subnets, SG rules) is NOT fetched here: cb-spider has no live
	// "get" for an existing, unmanaged VPC/SG (GetCSPResourceInfo supports only
	// VM/DISK), so obtaining detail would require a register→get→unregister dance
	// against cb-spider. Tracked as a follow-up.
	csp.Network.VPC.Name = vm.VpcIID.NameId
	for _, sgIID := range vm.SecurityGroupIIds {
		if sgIID.NameId == "" {
			continue
		}
		csp.Network.SecurityGroups = append(csp.Network.SecurityGroups, infra.CSPSecurityGroup{Name: sgIID.NameId})
	}

	return csp
}

// clusterInfoToK8s maps spider.ClusterInfo into the agent's kubernetes.Kubernetes
// shape — primarily node counts derived from NodeGroupList desired sizes.
func clusterInfoToK8s(cl *spider.ClusterInfo) kubernetes.Kubernetes {
	worker := 0
	for _, ng := range cl.NodeGroupList {
		worker += ng.DesiredNodeSize
	}
	return kubernetes.Kubernetes{
		NodeCount: kubernetes.NodeCount{
			Total:  worker,
			Worker: worker,
		},
	}
}

// bucketToData maps an S3 bucket into the agent's data.DataInfo shape, reusing
// the MinIO sub-structure as the object-storage carrier.
func bucketToData(b *spider.S3BucketInfo) data.DataInfo {
	return data.DataInfo{
		MinIO: &data.MinIOData{
			Address: b.Region,
			Buckets: []data.MinioBucket{{Name: b.Name}},
		},
	}
}

// upsertSavedCSPData writes the CSP-side VM info into SavedInfraInfo.csp_data,
// leaving infra_data (the agent-collected data) untouched. gorm's Updates skips
// zero-value fields, so preserving the loaded record's InfraData means an
// existing agent import is not overwritten.
func upsertSavedCSPData(connID string, csp infra.CSPInfo) error {
	raw, err := json.Marshal(csp)
	if err != nil {
		return err
	}
	if existing, _ := dao.SavedInfraInfoGet(connID); existing != nil {
		existing.CSPData = string(raw)
		existing.Status = model.ConnectionInfoStatusSuccess
		existing.SavedTime = time.Now()
		return dao.SavedInfraInfoUpdate(existing)
	}
	rec := &model.SavedInfraInfo{
		ConnectionID: connID,
		CSPData:      string(raw),
		Status:       model.ConnectionInfoStatusSuccess,
		SavedTime:    time.Now(),
	}
	_, err = dao.SavedInfraInfoRegister(rec)
	return err
}

// upsertSavedK8s writes (or replaces) SavedKubernetesInfo for a connection.
func upsertSavedK8s(connID string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	rec := &model.SavedKubernetesInfo{
		ConnectionID:   connID,
		KubernetesData: string(raw),
		Status:         model.ConnectionInfoStatusSuccess,
		SavedTime:      time.Now(),
	}
	if existing, _ := dao.SavedKubernetesInfoGet(connID); existing != nil {
		return dao.SavedKubernetesInfoUpdate(rec)
	}
	_, err = dao.SavedKubernetesInfoRegister(rec)
	return err
}

// upsertSavedData writes (or replaces) SavedDataInfo for a connection.
func upsertSavedData(connID string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	rec := &model.SavedDataInfo{
		ConnectionID: connID,
		DataData:     string(raw),
		Status:       model.ConnectionInfoStatusSuccess,
		SavedTime:    time.Now(),
	}
	if existing, _ := dao.SavedDataInfoGet(connID); existing != nil {
		return dao.SavedDataInfoUpdate(rec)
	}
	_, err = dao.SavedDataInfoRegister(rec)
	return err
}

// cspVMIdentifier returns the identifier cb-spider expects for a VM lookup.
// cb-spider's "GET /cspvm/:Id" path param is not URL-decoded, so a full CSP
// resource ID that contains "/" (e.g. an Azure ARM ID) gets mangled into a
// double-encoded request. cb-spider's drivers identify a VM by its name within
// the connection's region/resource-group anyway, so we send the last path
// segment (the VM name) — ".../virtualMachines/ish-test" -> "ish-test".
// Slash-less IDs (e.g. an AWS instance id) pass through unchanged.
func cspVMIdentifier(resourceID string) string {
	id := strings.TrimRight(strings.TrimSpace(resourceID), "/")
	if i := strings.LastIndex(id, "/"); i >= 0 {
		return id[i+1:]
	}
	return id
}

// refreshCSPConnection contacts cb-spider for the resource described by ci and
// stores the adapted result in the relevant Saved*Info table.
func refreshCSPConnection(sg *model.SourceGroup, ci *model.ConnectionInfo) error {
	if ci.ResourceID == "" {
		return errors.New("resource_id is empty")
	}

	// Register a temporary cb-spider connection for the duration of this call only —
	// credentials are never persisted in cb-spider.
	return withSpiderConnection(sg, func(connName string) error {
		switch ci.ResourceType {
		case "vm":
			vm, err := spider.GetCSPVM(connName, cspVMIdentifier(ci.ResourceID))
			if err != nil {
				return err
			}
			return upsertSavedCSPData(ci.ID, buildCSPInfo(sg, vm))
		case "k8s":
			cl, err := spider.GetCluster(connName, ci.ResourceID)
			if err != nil {
				return err
			}
			return upsertSavedK8s(ci.ID, clusterInfoToK8s(cl))
		case "object_storage":
			b, err := spider.GetS3BucketLocation(connName, ci.ResourceID)
			if err != nil {
				return err
			}
			return upsertSavedData(ci.ID, bucketToData(b))
		default:
			return errors.New("unsupported resource_type: " + ci.ResourceType)
		}
	})
}

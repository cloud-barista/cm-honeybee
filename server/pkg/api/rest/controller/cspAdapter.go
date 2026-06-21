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

// vmInfoToInfra maps a spider VMInfo into the on-prem infra.Infra shape so
// downstream consumers get a uniform structure. Fields that have no equivalent
// in the CSP world are left zero.
func vmInfoToInfra(vm *spider.VMInfo) infra.Infra {
	kvMap := keyValueListToMap(vm.KeyValueList)
	osName := vm.Platform
	if osName == "" {
		osName = kvMap["Architecture"]
	}

	rootDiskSize := uint(0)
	if v, err := strconv.ParseUint(strings.TrimSpace(vm.RootDiskSize), 10, 64); err == nil {
		rootDiskSize = uint(v)
	}

	return infra.Infra{
		Compute: infra.Compute{
			OS: infra.System{
				OS: infra.OS{
					PrettyName: vm.VMSpecName,
					Name:       osName,
				},
				Node: infra.Node{
					Hostname: vm.IId.NameId,
				},
			},
			ComputeResource: infra.ComputeResource{
				RootDisk: infra.Disk{
					Name: vm.RootDeviceName,
					Type: vm.RootDiskType,
					Size: rootDiskSize,
				},
			},
		},
	}
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

// upsertSavedInfra writes (or replaces) SavedInfraInfo for a connection.
func upsertSavedInfra(connID string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	rec := &model.SavedInfraInfo{
		ConnectionID: connID,
		InfraData:    string(raw),
		Status:       model.ConnectionInfoStatusSuccess,
		SavedTime:    time.Now(),
	}
	if existing, _ := dao.SavedInfraInfoGet(connID); existing != nil {
		return dao.SavedInfraInfoUpdate(rec)
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
			vm, err := spider.GetVM(connName, ci.ResourceID)
			if err != nil {
				return err
			}
			return upsertSavedInfra(ci.ID, vmInfoToInfra(vm))
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

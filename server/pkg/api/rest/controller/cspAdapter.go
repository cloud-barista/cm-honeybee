package controller

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	storagemodel "github.com/cloud-barista/cm-beetle/imdl/storage-model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/data"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"
	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
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
// disks, tags) plus the attached VPC/subnet/security groups.
//
// Drivers fill a VM's IIDs unevenly: aws, alibaba and ncp report the VPC,
// subnet, data disks and (alibaba) image and security groups with a SystemId
// and an empty NameId, because a resource cb-spider did not create has no
// cb-spider name. Every IID is therefore matched and named by NameId or
// SystemId, never by NameId alone.
func buildCSPInfo(conn spider.Conn, sg *model.SourceGroup, vm *spider.VMInfo) infra.CSPInfo {
	csp := buildCSPInfoWith(driverNetworkLookup(conn), sg, vm)
	addCSPDetail(driverDetailLookup(conn), vm, &csp)
	return csp
}

// cspDetailLookup is how addCSPDetail reaches the driver for the VM's state,
// key pair, data disks, spec and image; tests replace it. A nil func means the
// driver has no handler for it and the field is skipped.
type cspDetailLookup struct {
	GetVMStatus func(spider.IID) (string, error)
	GetKeyPair  func(spider.IID) (*spider.KeyPairInfo, error)
	GetDisk     func(spider.IID) (*spider.DiskInfo, error)
	GetVMSpec   func(string) (*spider.VMSpecInfo, error)
	GetImage    func(spider.IID) (*spider.ImageInfo, error)
}

func driverDetailLookup(conn spider.Conn) cspDetailLookup {
	lookup := cspDetailLookup{
		GetVMStatus: func(iid spider.IID) (string, error) { return spider.GetVMStatus(conn, iid) },
		GetKeyPair:  func(iid spider.IID) (*spider.KeyPairInfo, error) { return spider.GetKeyPair(conn, iid) },
		GetDisk:     func(iid spider.IID) (*spider.DiskInfo, error) { return spider.GetDisk(conn, iid) },
		GetVMSpec:   func(name string) (*spider.VMSpecInfo, error) { return spider.GetVMSpec(conn, name) },
		GetImage:    func(iid spider.IID) (*spider.ImageInfo, error) { return spider.GetImage(conn, iid) },
	}

	capability, err := spider.GetDriverCapability(conn.Provider)
	if err != nil {
		logger.Println(logger.WARN, true, "CSP: driver capability unknown "+connLabel(conn)+", asking every handler: "+err.Error())
		return lookup
	}
	if !capability.VMHandler {
		lookup.GetVMStatus = nil
	}
	if !capability.KeyPairHandler {
		lookup.GetKeyPair = nil
	}
	if !capability.DiskHandler {
		lookup.GetDisk = nil
	}
	if !capability.VMSpecHandler {
		lookup.GetVMSpec = nil
	}
	if !capability.ImageHandler {
		lookup.GetImage = nil
	}
	return lookup
}

// atoiOrZero reads a driver's numeric string. "-1" (not exposed), "default",
// "NA" and other non-numbers become 0.
func atoiOrZero(s string) uint {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v < 0 {
		return 0
	}
	return uint(v)
}

// naToEmpty drops the "NA" cb-spider drivers report for an unknown name.
func naToEmpty(s string) string {
	if strings.EqualFold(strings.TrimSpace(s), "NA") {
		return ""
	}
	return s
}

// specArchKeys are the KeyValueList keys under which drivers copy the CSP's
// own spec record (cb-spider's VMSpecInfo has no architecture field): aws
// ProcessorInfo ({SupportedArchitectures:[x86_64],...}), gcp Architecture,
// alibaba CpuArchitecture and ibm VcpuArchitecture ({...,value:amd64}).
var specArchKeys = []string{"ProcessorInfo", "Architecture", "CpuArchitecture", "VcpuArchitecture"}

var archToken = regexp.MustCompile(`(?i)\b(x86_64|amd64|arm64|aarch64|i386|x86|arm|s390x|ppc64le)\b`)

// specArchitecture picks the architecture out of a spec's KeyValueList, or
// returns "" when the driver does not report one (azure, ncp, nhn and
// openstack do not).
func specArchitecture(kvs []spider.KeyValue) string {
	for _, key := range specArchKeys {
		for _, kv := range kvs {
			if kv.Key != key {
				continue
			}
			// aws lists i386 next to x86_64 for some types; the 64-bit
			// one is what the VM runs.
			found := archToken.FindAllString(kv.Value, -1)
			for _, m := range found {
				switch strings.ToLower(m) {
				case "i386", "x86", "arm":
				default:
					return strings.ToLower(m)
				}
			}
			if len(found) > 0 {
				return strings.ToLower(found[0])
			}
		}
	}
	return ""
}

func toCSPVMSpec(s *spider.VMSpecInfo) *infra.CSPVMSpec {
	out := &infra.CSPVMSpec{
		Name:         s.Name,
		VCPUs:        atoiOrZero(s.VCpu.Count),
		MemoryMiB:    atoiOrZero(s.MemSizeMiB),
		DiskGB:       atoiOrZero(s.DiskSizeGB),
		Architecture: specArchitecture(s.KeyValueList),
	}
	for _, g := range s.Gpu {
		out.GPUs = append(out.GPUs, infra.CSPGPU{
			Count:    atoiOrZero(g.Count),
			Vendor:   naToEmpty(g.Mfr),
			Model:    naToEmpty(g.Model),
			MemoryGB: atoiOrZero(g.MemSizeGB),
		})
	}
	if len(out.GPUs) == 0 {
		for _, a := range s.Accelerator {
			if !strings.EqualFold(a.Type, "GPU") {
				continue
			}
			out.GPUs = append(out.GPUs, infra.CSPGPU{
				Count:    atoiOrZero(a.Count),
				Vendor:   naToEmpty(a.Mfr),
				Model:    naToEmpty(a.Model),
				MemoryGB: atoiOrZero(a.MemSizeGB),
			})
		}
	}
	return out
}

// addCSPDetail fills the CSPInfo fields that need a driver call beyond GetVM.
// Every call is best-effort: a failure is logged and leaves only its own field
// empty, so the import still stores what GetVM reported.
func addCSPDetail(lookup cspDetailLookup, vm *spider.VMInfo, csp *infra.CSPInfo) {
	vmLabel := "VM " + iidString(vm.IId)

	if lookup.GetVMStatus != nil {
		if status, err := lookup.GetVMStatus(vm.IId); err != nil {
			logger.Println(logger.WARN, true, "CSP: "+vmLabel+": status not resolved: "+err.Error())
		} else {
			csp.Status = status
		}
	}

	// The key pair's name comes from the VM itself and is kept even when the
	// key pair lookup fails.
	if name := iidLabel(vm.KeyPairIId); name != "" {
		kp := &infra.CSPKeyPair{Name: name, ID: vm.KeyPairIId.SystemId}
		if lookup.GetKeyPair != nil {
			if k, err := lookup.GetKeyPair(vm.KeyPairIId); err != nil {
				logger.Println(logger.WARN, true, "CSP: "+vmLabel+": key pair "+iidString(vm.KeyPairIId)+" not resolved, fingerprint left empty: "+err.Error())
			} else {
				if kp.ID == "" {
					kp.ID = k.IId.SystemId
				}
				kp.Fingerprint = k.Fingerprint
			}
		}
		csp.KeyPair = kp
	}

	if lookup.GetDisk != nil {
		for _, iid := range vm.DataDiskIIDs {
			if iidLabel(iid) == "" {
				continue
			}
			d, err := lookup.GetDisk(iid)
			if err != nil {
				logger.Println(logger.WARN, true, "CSP: "+vmLabel+": data disk "+iidString(iid)+" not resolved: "+err.Error())
				continue
			}
			name := iidLabel(iid)
			if iid.NameId == "" && d.IId.NameId != "" {
				name = d.IId.NameId
			}
			id := iid.SystemId
			if id == "" {
				id = d.IId.SystemId
			}
			csp.DataDiskDetails = append(csp.DataDiskDetails, infra.CSPDataDisk{
				Name:   name,
				ID:     id,
				Size:   atoiOrZero(d.DiskSize),
				Type:   d.DiskType,
				Status: d.Status,
			})
		}
	}

	if lookup.GetVMSpec != nil && vm.VMSpecName != "" {
		if s, err := lookup.GetVMSpec(vm.VMSpecName); err != nil {
			logger.Println(logger.WARN, true, "CSP: "+vmLabel+": VM spec '"+vm.VMSpecName+"' not resolved: "+err.Error())
		} else {
			csp.VMSpecDetail = toCSPVMSpec(s)
		}
	}

	if lookup.GetImage != nil && iidLabel(vm.ImageIId) != "" {
		if img, err := lookup.GetImage(vm.ImageIId); err != nil {
			logger.Println(logger.WARN, true, "CSP: "+vmLabel+": image "+iidString(vm.ImageIId)+" not resolved: "+err.Error())
		} else {
			name := img.Name
			if name == "" {
				name = iidLabel(vm.ImageIId)
			}
			csp.ImageDetail = &infra.CSPImage{
				Name:           name,
				ID:             img.IId.SystemId,
				OSDistribution: naToEmpty(img.OSDistribution),
				OSArchitecture: naToEmpty(img.OSArchitecture),
				OSPlatform:     naToEmpty(img.OSPlatform),
				OSDiskType:     naToEmpty(img.OSDiskType),
				OSDiskSizeGB:   atoiOrZero(img.OSDiskSizeGB),
				Status:         naToEmpty(img.ImageStatus),
			}
		}
	}
}

// cspNetworkLookup is how buildCSPInfoWith reaches the driver for VPC and
// security group detail; tests replace it.
type cspNetworkLookup struct {
	GetVPC            func(spider.IID) (*spider.VPCInfo, error)
	ListVPC           func() ([]spider.VPCInfo, error)
	GetSecurityGroup  func(spider.IID) (*spider.SecurityGroupInfo, error)
	ListSecurityGroup func() ([]spider.SecurityGroupInfo, error)
}

func driverNetworkLookup(conn spider.Conn) cspNetworkLookup {
	return cspNetworkLookup{
		GetVPC:            func(iid spider.IID) (*spider.VPCInfo, error) { return spider.GetVPC(conn, iid) },
		ListVPC:           func() ([]spider.VPCInfo, error) { return spider.ListVPC(conn) },
		GetSecurityGroup:  func(iid spider.IID) (*spider.SecurityGroupInfo, error) { return spider.GetSecurityGroup(conn, iid) },
		ListSecurityGroup: func() ([]spider.SecurityGroupInfo, error) { return spider.ListSecurityGroup(conn) },
	}
}

// iidLabel names a resource by its NameId, or by its SystemId when the driver
// left the NameId empty.
func iidLabel(iid spider.IID) string {
	if iid.NameId != "" {
		return iid.NameId
	}
	return iid.SystemId
}

// iidMatches reports whether two IIDs name the same resource. Empty fields
// never match, so IIDs that both lack a NameId are compared by SystemId only.
func iidMatches(a, b spider.IID) bool {
	return (a.SystemId != "" && a.SystemId == b.SystemId) || (a.NameId != "" && a.NameId == b.NameId)
}

func iidString(iid spider.IID) string {
	return "{NameId: '" + iid.NameId + "', SystemId: '" + iid.SystemId + "'}"
}

// findVPC asks the driver for the VPC by the VM's own VpcIID first, which
// avoids listing every VPC (gcp's listing ran past the 60s driver timeout).
// When Get fails, the listing is matched by NameId or SystemId instead.
func findVPC(lookup cspNetworkLookup, iid spider.IID) (*spider.VPCInfo, error) {
	v, getErr := lookup.GetVPC(iid)
	if getErr == nil && v != nil {
		return v, nil
	}

	vpcs, err := lookup.ListVPC()
	if err != nil {
		return nil, errors.New("get: " + errString(getErr) + ", list: " + err.Error())
	}
	for i := range vpcs {
		if iidMatches(vpcs[i].IId, iid) {
			return &vpcs[i], nil
		}
	}
	return nil, errors.New("get: " + errString(getErr) + ", list: no VPC matches")
}

// securityGroupFinder looks security groups up as findVPC does, listing at most
// once for all of a VM's groups that Get could not resolve.
type securityGroupFinder struct {
	lookup  cspNetworkLookup
	listed  bool
	all     []spider.SecurityGroupInfo
	listErr error
}

func (f *securityGroupFinder) find(iid spider.IID) (*spider.SecurityGroupInfo, error) {
	g, getErr := f.lookup.GetSecurityGroup(iid)
	if getErr == nil && g != nil {
		return g, nil
	}

	if !f.listed {
		f.all, f.listErr = f.lookup.ListSecurityGroup()
		f.listed = true
	}
	if f.listErr != nil {
		return nil, errors.New("get: " + errString(getErr) + ", list: " + f.listErr.Error())
	}
	for i := range f.all {
		if iidMatches(f.all[i].IId, iid) {
			return &f.all[i], nil
		}
	}
	return nil, errors.New("get: " + errString(getErr) + ", list: no security group matches")
}

func errString(err error) string {
	if err == nil {
		return "empty result"
	}
	return err.Error()
}

func buildCSPInfoWith(lookup cspNetworkLookup, sg *model.SourceGroup, vm *spider.VMInfo) infra.CSPInfo {
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
		if label := iidLabel(d); label != "" {
			dataDisks = append(dataDisks, label)
		}
	}

	tags := map[string]string{}
	for _, t := range vm.TagList {
		tags[t.Key] = t.Value
	}

	vmLabel := "VM " + iidString(vm.IId)

	csp := infra.CSPInfo{
		Provider:  strings.ToLower(sg.ProviderName),
		Region:    vm.Region.Region,
		Zone:      vm.Region.Zone,
		Name:      vm.IId.NameId,
		ID:        vm.IId.SystemId,
		VMSpec:    vm.VMSpecName,
		Image:     iidLabel(vm.ImageIId),
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
			Subnet: iidLabel(vm.SubnetIID),
		},
	}

	// VPC detail (CIDR, subnets). Best-effort: a lookup failure keeps the
	// VM's own identifiers and leaves the detail empty.
	csp.Network.VPC.Name = iidLabel(vm.VpcIID)
	if csp.Network.VPC.Name == "" {
		logger.Println(logger.WARN, true, "CSP: "+vmLabel+" reports no VPC")
	} else if v, err := findVPC(lookup, vm.VpcIID); err != nil {
		logger.Println(logger.WARN, true, "CSP: "+vmLabel+": VPC "+iidString(vm.VpcIID)+" not resolved, CIDR and subnets left empty: "+err.Error())
	} else {
		if vm.VpcIID.NameId == "" && v.IId.NameId != "" {
			csp.Network.VPC.Name = v.IId.NameId
		}
		csp.Network.VPC.CIDR = v.IPv4_CIDR
		subnetFound := false
		for _, sn := range v.SubnetInfoList {
			csp.Network.VPC.Subnets = append(csp.Network.VPC.Subnets, infra.CSPSubnet{
				Name: iidLabel(sn.IId),
				CIDR: sn.IPv4_CIDR,
				Zone: sn.Zone,
			})
			if iidMatches(sn.IId, vm.SubnetIID) {
				subnetFound = true
				if vm.SubnetIID.NameId == "" && sn.IId.NameId != "" {
					csp.Network.Subnet = sn.IId.NameId
				}
			}
		}
		if csp.Network.Subnet != "" && !subnetFound {
			logger.Println(logger.WARN, true, "CSP: "+vmLabel+": subnet "+iidString(vm.SubnetIID)+" not found in VPC "+iidString(v.IId))
		}
	}

	// Security groups with their rules. A group that cannot be resolved is
	// still reported by its identifier.
	finder := &securityGroupFinder{lookup: lookup}
	for _, sgIID := range vm.SecurityGroupIIds {
		name := iidLabel(sgIID)
		if name == "" {
			logger.Println(logger.WARN, true, "CSP: "+vmLabel+" reports a security group with no identifier")
			continue
		}
		out := infra.CSPSecurityGroup{Name: name}
		if g, err := finder.find(sgIID); err != nil {
			logger.Println(logger.WARN, true, "CSP: "+vmLabel+": security group "+iidString(sgIID)+" not resolved, rules left empty: "+err.Error())
		} else {
			if sgIID.NameId == "" && g.IId.NameId != "" {
				out.Name = g.IId.NameId
			}
			for _, r := range g.SecurityRules {
				out.Rules = append(out.Rules, infra.CSPSecurityRule{
					Direction: r.Direction,
					Protocol:  r.IPProtocol,
					FromPort:  r.FromPort,
					ToPort:    r.ToPort,
					CIDR:      r.CIDR,
				})
			}
		}
		csp.Network.SecurityGroups = append(csp.Network.SecurityGroups, out)
	}

	return csp
}

// clusterInfoToK8s maps spider.ClusterInfo into the record stored for a CSP
// k8s connection. The cluster name and version go where refined reads them
// from an agent-collected cluster, so a managed cluster becomes the refined
// k8sCluster as well. Node counts are the desired sizes of the node groups,
// all of them workers: the control plane of a managed cluster is the CSP's.
//
// The pod and service CIDRs stay empty: cb-spider's ClusterInfo.Network holds
// the VPC, subnets and security groups only.
func clusterInfoToK8s(cl *spider.ClusterInfo) model.CSPKubernetesRecord {
	worker := 0
	groups := make([]model.CSPKubernetesNodeGroup, 0, len(cl.NodeGroupList))
	for _, ng := range cl.NodeGroupList {
		worker += ng.DesiredNodeSize
		groups = append(groups, model.CSPKubernetesNodeGroup{
			Name:            iidLabel(ng.IId),
			VMSpec:          naToEmpty(ng.VMSpecName),
			Image:           naToEmpty(iidLabel(ng.ImageIID)),
			RootDiskType:    naToEmpty(ng.RootDiskType),
			RootDiskSize:    naToEmpty(ng.RootDiskSize),
			OnAutoScaling:   ng.OnAutoScaling,
			DesiredNodeSize: ng.DesiredNodeSize,
			MinNodeSize:     ng.MinNodeSize,
			MaxNodeSize:     ng.MaxNodeSize,
			Status:          ng.Status,
		})
	}

	return model.CSPKubernetesRecord{
		Kubernetes: kubernetes.Kubernetes{
			Cluster: kubernetes.Cluster{
				Name:    iidLabel(cl.IId),
				Version: cl.Version,
			},
			NodeCount: kubernetes.NodeCount{
				Total:  worker,
				Worker: worker,
			},
		},
		CSP: &model.CSPKubernetesCluster{
			SystemID:    cl.IId.SystemId,
			Status:      cl.Status,
			CreatedTime: clusterCreatedTime(cl.CreatedTime),
			NodeGroups:  groups,
		},
	}
}

// clusterCreatedTime drops the zero time cb-spider reports for a cluster whose
// driver did not fill CreatedTime, so an unknown time reads as absent rather
// than as year 1.
func clusterCreatedTime(s string) string {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err == nil && t.IsZero() {
		return ""
	}
	return s
}

// bucketToData maps a CSP bucket into the agent's data.DataInfo shape, reusing
// the MinIO sub-structure as the object-storage carrier. Address carries the
// region the bucket lives in, or the source group's region where the CSP
// cannot tell.
func bucketToData(region string, d *spider.S3BucketDetail) data.DataInfo {
	return data.DataInfo{
		MinIO: &data.MinIOData{
			Address: region,
			Buckets: []data.MinioBucket{{
				Name:      d.Name,
				Objects:   d.ObjectCount,
				Versioned: d.VersioningEnabled,
			}},
			Errors: d.Errors,
		},
	}
}

// bucketToSourceProperty maps a CSP bucket onto the source model cm-beetle
// recommends a target bucket from.
func bucketToSourceProperty(d *spider.S3BucketDetail) storagemodel.SourceObjectStorageProperty {
	p := storagemodel.SourceObjectStorageProperty{BucketName: d.Name}
	p.VersioningEnabled = d.VersioningEnabled
	p.CORSEnabled = len(d.CORSRules) > 0
	for _, r := range d.CORSRules {
		p.CORSRule = append(p.CORSRule, storagemodel.CORSRule{
			AllowedOrigin: r.AllowedOrigin,
			AllowedMethod: r.AllowedMethod,
			AllowedHeader: r.AllowedHeader,
			ExposeHeader:  r.ExposeHeader,
			MaxAgeSeconds: r.MaxAgeSeconds,
		})
	}
	p.EncryptionEnabled = d.EncryptionEnabled
	p.IsPublic = d.IsPublic
	if d.ObjectsCounted {
		p.TotalSizeBytes = d.TotalSizeBytes
		p.ObjectCount = d.ObjectCount
	}
	p.Tags = d.Tags
	p.CreationDate = d.CreationDate
	return p
}

// cspBucketRecord is what SavedDataInfo.csp_data holds for a CSP bucket: the
// source property cm-beetle reads, plus the region the bucket lives in, which
// the property has no field for. Region is empty where the CSP cannot tell
// (spider.GetS3BucketRegions) and in records stored before it was kept.
type cspBucketRecord struct {
	storagemodel.SourceObjectStorageProperty
	Region string `json:"region,omitempty"`
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

// upsertSavedRDBMS writes (or replaces) SavedRDBMSInfo for a connection.
func upsertSavedRDBMS(connID string, payload model.RDBMSInfo) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	rec := &model.SavedRDBMSInfo{
		ConnectionID: connID,
		RDBMSData:    string(raw),
		Status:       model.ConnectionInfoStatusSuccess,
		SavedTime:    time.Now(),
	}
	if existing, _ := dao.SavedRDBMSInfoGet(connID); existing != nil {
		return dao.SavedRDBMSInfoUpdate(rec)
	}
	_, err = dao.SavedRDBMSInfoRegister(rec)
	return err
}

// upsertSavedNLB writes (or replaces) SavedNLBInfo for a connection.
func upsertSavedNLB(connID string, record model.CSPNLBRecord) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	rec := &model.SavedNLBInfo{
		ConnectionID: connID,
		NLBData:      string(raw),
		Status:       model.ConnectionInfoStatusSuccess,
		SavedTime:    time.Now(),
	}
	if existing, _ := dao.SavedNLBInfoGet(connID); existing != nil {
		return dao.SavedNLBInfoUpdate(rec)
	}
	_, err = dao.SavedNLBInfoRegister(rec)
	return err
}

// upsertSavedData writes (or replaces) SavedDataInfo for a connection. cspData,
// when not nil, is stored as the CSP-reported bucket.
func upsertSavedData(connID string, payload any, cspData any) error {
	return upsertSavedDataStatus(connID, payload, cspData, model.ConnectionInfoStatusSuccess)
}

// upsertSavedDataStatus is upsertSavedData with the status to store, so a
// collection that read nothing keeps its errors on record without passing for
// a success.
func upsertSavedDataStatus(connID string, payload any, cspData any, status string) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var cspRaw []byte
	if cspData != nil {
		if cspRaw, err = json.Marshal(cspData); err != nil {
			return err
		}
	}
	rec := &model.SavedDataInfo{
		ConnectionID: connID,
		DataData:     string(raw),
		CSPData:      string(cspRaw),
		Status:       status,
		SavedTime:    time.Now(),
	}
	if existing, _ := dao.SavedDataInfoGet(connID); existing != nil {
		return dao.SavedDataInfoUpdate(rec)
	}
	_, err = dao.SavedDataInfoRegister(rec)
	return err
}

// storeBucketDetail saves what was read of a bucket. A bucket none of whose
// items could be read is saved as failed, its errors kept as the record of why,
// and returned as an error, the way a database import that read nothing is
// (dbImportStatus). Some items failing is still a success: the rest were read
// and the failures are listed in the stored errors.
//
// address is what the data record's Address shows; region is the bucket's
// region as the CSP reported it, "" when it cannot tell.
func storeBucketDetail(connID string, address string, region string, d *spider.S3BucketDetail) error {
	status := model.ConnectionInfoStatusSuccess
	if d.NothingRead() {
		status = model.ConnectionInfoStatusFailed
	}

	record := cspBucketRecord{SourceObjectStorageProperty: bucketToSourceProperty(d), Region: region}
	if err := upsertSavedDataStatus(connID, bucketToData(address, d), record, status); err != nil {
		return err
	}

	if d.NothingRead() {
		return errors.New("none of the details of bucket '" + d.Name + "' could be read: " +
			agentResponseExcerpt(strings.Join(d.Errors, "; ")))
	}

	return nil
}

// markCSPResourceFailed records a failed CSP collection in the table the
// resource keeps its result in, so the last success stored there is not read
// as current. The stored data is kept: GET still shows what was collected
// last, while refined leaves the record out. A connection that has no record
// yet gets a failed one. A VM keeps its result in SavedInfraInfo, which the
// infra import marks itself.
func markCSPResourceFailed(ci *model.ConnectionInfo) error {
	now := time.Now()

	switch ci.ResourceType {
	case serverCommon.ResourceTypeK8s:
		if existing, _ := dao.SavedKubernetesInfoGet(ci.ID); existing != nil {
			return dao.SavedKubernetesInfoUpdate(&model.SavedKubernetesInfo{
				ConnectionID: ci.ID, Status: model.ConnectionInfoStatusFailed})
		}
		_, err := dao.SavedKubernetesInfoRegister(&model.SavedKubernetesInfo{
			ConnectionID: ci.ID, Status: model.ConnectionInfoStatusFailed, SavedTime: now})
		return err
	case serverCommon.ResourceTypeObjectStorage:
		if existing, _ := dao.SavedDataInfoGet(ci.ID); existing != nil {
			return dao.SavedDataInfoUpdate(&model.SavedDataInfo{
				ConnectionID: ci.ID, Status: model.ConnectionInfoStatusFailed})
		}
		_, err := dao.SavedDataInfoRegister(&model.SavedDataInfo{
			ConnectionID: ci.ID, Status: model.ConnectionInfoStatusFailed, SavedTime: now})
		return err
	case serverCommon.ResourceTypeRDBMS:
		if existing, _ := dao.SavedRDBMSInfoGet(ci.ID); existing != nil {
			return dao.SavedRDBMSInfoUpdate(&model.SavedRDBMSInfo{
				ConnectionID: ci.ID, Status: model.ConnectionInfoStatusFailed})
		}
		_, err := dao.SavedRDBMSInfoRegister(&model.SavedRDBMSInfo{
			ConnectionID: ci.ID, Status: model.ConnectionInfoStatusFailed, SavedTime: now})
		return err
	case serverCommon.ResourceTypeNLB:
		if existing, _ := dao.SavedNLBInfoGet(ci.ID); existing != nil {
			return dao.SavedNLBInfoUpdate(&model.SavedNLBInfo{
				ConnectionID: ci.ID, Status: model.ConnectionInfoStatusFailed})
		}
		_, err := dao.SavedNLBInfoRegister(&model.SavedNLBInfo{
			ConnectionID: ci.ID, Status: model.ConnectionInfoStatusFailed, SavedTime: now})
		return err
	default:
		return nil
	}
}

// cspVMIdentifier returns the identifier a driver's GetVM expects. cb-spider's
// drivers identify a VM by its name within the connection's
// region/resource-group, so a full CSP resource ID that contains "/" (e.g. an
// Azure ARM ID) is cut down to its last path segment (the VM name) -
// ".../virtualMachines/ish-test" -> "ish-test". Slash-less IDs (e.g. an AWS
// instance id) pass through unchanged. This is also what the REST client sent
// to cb-spider's GET /cspvm/{Id}, so stored resource IDs resolve as before.
func cspVMIdentifier(resourceID string) string {
	id := strings.TrimRight(strings.TrimSpace(resourceID), "/")
	if i := strings.LastIndex(id, "/"); i >= 0 {
		return id[i+1:]
	}
	return id
}

// ErrCSPResourceNotFound marks a lookup the CSP answered without the
// resource: the listing came back and nothing in it matched. A resource the CSP
// could not be asked about (timeout, credential, unreachable endpoint) is not
// marked, as nothing is known about it then.
var ErrCSPResourceNotFound = errors.New("the csp resource was not found")

// ErrBucketRegionMismatch marks a bucket that exists but lives outside the
// source group's region (bucketRegionMismatch).
var ErrBucketRegionMismatch = errors.New("the bucket is outside the source group's region")

// cspResourceError carries one of the sentinels above under its own message,
// so that the messages stay as they were while errors.Is can tell them apart.
type cspResourceError struct {
	kind error
	msg  string
}

func (e cspResourceError) Error() string { return e.msg }

func (e cspResourceError) Is(target error) bool { return target == e.kind }

func cspNotFound(msg string) error {
	return cspResourceError{kind: ErrCSPResourceNotFound, msg: msg}
}

// cspResourceRejected reports whether a check of the resource failed because
// the resource cannot be the connection's, rather than because the CSP could
// not be asked. Such a connection is refused at registration; the others are
// registered with connection_status failed and checked again on refresh.
func cspResourceRejected(err error) bool {
	return errors.Is(err, ErrCSPResourceNotFound) || errors.Is(err, ErrBucketRegionMismatch)
}

// The VM calls of the CSP, kept in variables so tests can stand in for it.
var (
	getCSPVM  = spider.GetCSPVM
	listCSPVM = spider.ListVM
)

// findVMByID asks the driver for the VM, and when that fails, tells a VM that
// does not exist from a driver that could not answer by the listing. The
// drivers report a missing VM in their own words (aws InvalidInstanceID.NotFound
// or .Malformed, others a 404 of their SDK), so the error text is not read:
// the VM is missing only when the listing succeeds and holds no VM of that
// identifier. Otherwise the error of GetVM is returned as it was.
func findVMByID(conn spider.Conn, resourceID string) (*spider.VMInfo, error) {
	vm, getErr := getCSPVM(conn, cspVMIdentifier(resourceID))
	if getErr == nil {
		return vm, nil
	}

	vms, err := listCSPVM(conn)
	if err != nil || matchVM(vms, resourceID) != nil {
		return nil, getErr
	}

	return nil, cspNotFound("vm '" + resourceID + "' was not found " + connLabel(conn) + " (" + getErr.Error() + ")")
}

// matchVM picks the VM whose SystemId or NameId is resourceID or the name
// cspVMIdentifier cuts out of it. The comparison ignores case, as an Azure
// resource ID does; a match too many only leaves the GetVM error in place,
// whereas a match missed would refuse a VM that exists.
func matchVM(vms []spider.VMInfo, resourceID string) *spider.VMInfo {
	ids := []string{strings.TrimSpace(resourceID), cspVMIdentifier(resourceID)}
	for i := range vms {
		for _, id := range ids {
			if id == "" {
				continue
			}
			iid := vms[i].IId
			if strings.EqualFold(iid.SystemId, id) || strings.EqualFold(iid.NameId, id) {
				return &vms[i]
			}
		}
	}

	return nil
}

// findClusterByID looks a Kubernetes cluster up by the identifier discovery
// reported for it.
//
// The list is fetched and filtered here, as it was when honeybee went through
// cb-spider's REST API: /cluster/{Name} matched on the NameId of cb-spider's
// meta-DB, which never held a cluster cb-spider did not create. SystemId is matched first because
// it is the identifier a CSP-only cluster carries; NameId is accepted as well
// for a cluster cb-spider does manage.
func findClusterByID(conn spider.Conn, resourceID string) (*spider.ClusterInfo, error) {
	clusters, err := spider.ListCluster(conn)
	if err != nil {
		return nil, err
	}

	if cl := matchCluster(clusters, resourceID); cl != nil {
		return cl, nil
	}

	return nil, cspNotFound("cluster '" + resourceID + "' was not found " + connLabel(conn))
}

// matchCluster picks the cluster whose identifier is resourceID. SystemId is
// tried first because it is the identifier a CSP-only cluster carries; an empty
// NameId never matches, so a listing full of them cannot match an empty id.
func matchCluster(clusters []spider.ClusterInfo, resourceID string) *spider.ClusterInfo {
	if resourceID == "" {
		return nil
	}

	for i := range clusters {
		iid := clusters[i].IId
		if iid.SystemId == resourceID || (iid.NameId != "" && iid.NameId == resourceID) {
			return &clusters[i]
		}
	}

	return nil
}

// findRDBMSByID looks a managed database instance up by the identifier
// discovery reported for it. It lists and filters, as findClusterByID does,
// rather than calling the driver's GetRDBMS: drivers' GetRDBMS takes a SystemId,
// but discovery falls back to NameId when a driver leaves SystemId blank
// (pickIIDSystem), and only a listing can resolve both.
func findRDBMSByID(conn spider.Conn, resourceID string) (*spider.RDBMSInfo, error) {
	instances, err := spider.ListRDBMS(conn)
	if err != nil {
		return nil, err
	}

	if db := matchRDBMS(instances, resourceID); db != nil {
		return db, nil
	}

	return nil, cspNotFound("rdbms '" + resourceID + "' was not found " + connLabel(conn))
}

// matchRDBMS picks the instance whose identifier is resourceID, SystemId first
// and then NameId, by the same rule as matchCluster; an empty id never matches.
func matchRDBMS(instances []spider.RDBMSInfo, resourceID string) *spider.RDBMSInfo {
	if resourceID == "" {
		return nil
	}

	for i := range instances {
		iid := instances[i].IId
		if iid.SystemId == resourceID || (iid.NameId != "" && iid.NameId == resourceID) {
			return &instances[i]
		}
	}

	return nil
}

// collectRDBMS looks the instance up and attaches its spec size. The spec
// lookup is best-effort: without it the refined model leaves CPU and memory
// empty for cm-beetle to fill in, which is still a usable result, whereas
// failing here would lose the engine, version and storage as well.
func collectRDBMS(conn spider.Conn, resourceID string) (model.RDBMSInfo, error) {
	db, err := findRDBMSByID(conn, resourceID)
	if err != nil {
		return model.RDBMSInfo{}, err
	}

	payload := model.RDBMSInfo{RDBMS: db}
	spec, err := spider.GetDBSpec(conn, db.DBEngine, db.DBSpec)
	if err != nil {
		logger.Println(logger.WARN, true, "CSP: failed to get db spec '"+db.DBSpec+"' "+connLabel(conn)+": "+err.Error())
	} else {
		payload.DBSpec = spec
	}

	return payload, nil
}

// The NLB calls of the CSP, kept in variables so tests can stand in for it.
var (
	getCSPNLB  = spider.GetNLB
	listCSPNLB = spider.ListNLB
)

// findNLBByID looks an NLB up by the identifier discovery reported for it,
// its SystemId or, where the driver leaves that blank, its NameId
// (pickIIDSystem). The driver is asked for that one NLB first, as findVPC
// does, which avoids listing every NLB. The drivers do not agree on which
// field of the IID GetNLB reads (spider.GetNLB), so when the answer is an
// error or another NLB, the listing is matched by SystemId and then NameId
// instead, as findRDBMSByID does.
func findNLBByID(conn spider.Conn, resourceID string) (*spider.NLBInfo, error) {
	n, getErr := getCSPNLB(conn, spider.IID{SystemId: resourceID})
	if getErr == nil && n != nil && matchNLB([]spider.NLBInfo{*n}, resourceID) != nil {
		return n, nil
	}

	nlbs, err := listCSPNLB(conn)
	if err != nil {
		return nil, errors.New("get: " + errString(getErr) + ", list: " + err.Error())
	}
	if found := matchNLB(nlbs, resourceID); found != nil {
		return found, nil
	}

	return nil, cspNotFound("nlb '" + resourceID + "' was not found " + connLabel(conn))
}

// matchNLB picks the NLB whose identifier is resourceID, SystemId first and
// then NameId, by the same rule as matchCluster; an empty id never matches.
func matchNLB(nlbs []spider.NLBInfo, resourceID string) *spider.NLBInfo {
	if resourceID == "" {
		return nil
	}

	for i := range nlbs {
		iid := nlbs[i].IId
		if iid.SystemId == resourceID || (iid.NameId != "" && iid.NameId == resourceID) {
			return &nlbs[i]
		}
	}

	return nil
}

// The NLB types of the refined record.
const (
	nlbTypePublic   = "PUBLIC"
	nlbTypeInternal = "INTERNAL"
)

// normalizeNLBType reads whether an NLB faces the internet as PUBLIC or
// INTERNAL, or "" when the driver does not tell. The drivers spell it
// differently and some do not report it in Type at all: aws always reports
// PUBLIC and keeps the load balancer's own scheme (internet-facing |
// internal) in KeyValueList, alibaba reports an intranet NLB as PRIVATE, and
// gcp copies its forwarding rule scheme (EXTERNAL | INTERNAL) when it fills
// Type.
func normalizeNLBType(n *spider.NLBInfo) string {
	t := n.Type
	if scheme := strings.TrimSpace(keyValueListToMap(n.KeyValueList)["Scheme"]); scheme != "" {
		t = scheme
	}

	switch strings.ToUpper(strings.TrimSpace(t)) {
	case "PUBLIC", "EXTERNAL", "INTERNET-FACING":
		return nlbTypePublic
	case "INTERNAL", "PRIVATE":
		return nlbTypeInternal
	default:
		return ""
	}
}

// nonNegative turns the -1 a driver reports for a value it does not support
// into 0.
func nonNegative(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func upperTrim(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// nlbInfoToNLB maps spider.NLBInfo into the record stored for a CSP nlb
// connection. The values drivers spell differently are brought to one form
// here (normalizeNLBType, upper-case scope and protocols); discovery reports
// them verbatim. A backend VM the driver reports with no identifier is
// dropped.
func nlbInfoToNLB(n *spider.NLBInfo) model.CSPNLBRecord {
	vms := make([]model.CSPNLBVM, 0, len(n.VMGroup.VMs))
	for _, vm := range n.VMGroup.VMs {
		if iidLabel(vm) == "" {
			continue
		}
		vms = append(vms, model.CSPNLBVM{SystemID: vm.SystemId, Name: vm.NameId})
	}

	return model.CSPNLBRecord{
		SystemID: n.IId.SystemId,
		Name:     iidLabel(n.IId),
		VPC:      iidLabel(n.VpcIID),
		Type:     normalizeNLBType(n),
		Scope:    upperTrim(n.Scope),
		Listener: model.CSPNLBListener{
			Protocol: upperTrim(n.Listener.Protocol),
			IP:       strings.TrimSpace(n.Listener.IP),
			Port:     strings.TrimSpace(n.Listener.Port),
			DNSName:  strings.TrimSpace(n.Listener.DNSName),
		},
		VMGroup: model.CSPNLBVMGroup{
			Protocol: upperTrim(n.VMGroup.Protocol),
			Port:     strings.TrimSpace(n.VMGroup.Port),
			VMs:      vms,
		},
		HealthChecker: model.CSPNLBHealthChecker{
			Protocol:  upperTrim(n.HealthChecker.Protocol),
			Port:      strings.TrimSpace(n.HealthChecker.Port),
			Interval:  nonNegative(n.HealthChecker.Interval),
			Timeout:   nonNegative(n.HealthChecker.Timeout),
			Threshold: nonNegative(n.HealthChecker.Threshold),
		},
		CreatedTime: clusterCreatedTime(n.CreatedTime),
	}
}

// The bucket calls of the CSP, kept in variables so tests can stand in for it.
var (
	listS3Buckets       = spider.ListS3Buckets
	getS3BucketLocation = spider.GetS3BucketLocation
	getS3BucketRegions  = spider.GetS3BucketRegions
	getS3BucketDetail   = spider.GetS3BucketDetail
)

// findBucketByName looks an object storage bucket up the same way clusters are
// looked up: the listing goes to the CSP, so existence is decided there.
//
// The region is not in that listing, so it is read from GetS3BucketLocation as
// a best effort. It stays empty for a CSP that cannot tell and when the lookup
// fails, the way discovery lists such a bucket: a bucket is refused only when
// its region is known to differ from the group's (bucketRegionMismatch).
func findBucketByName(conn spider.Conn, bucketName string) (*spider.S3BucketInfo, error) {
	buckets, err := listS3Buckets(conn)
	if err != nil {
		return nil, err
	}

	found := matchBucket(buckets, bucketName)
	if found == nil {
		return nil, cspNotFound("bucket '" + bucketName + "' was not found " + connLabel(conn))
	}

	loc, err := getS3BucketLocation(conn, bucketName)
	if err != nil {
		logger.Println(logger.WARN, false, "CSP: the region of bucket '"+bucketName+"' is left unknown "+
			connLabel(conn)+": "+err.Error())
	} else if loc != nil {
		found.Region = loc.Region
	}

	return found, nil
}

// bucketRegionMismatch refuses a bucket that lives outside the source group's
// region: the group stands for one region, and its refined model names that
// region as the bucket's source. An unknown region ("") is not refused.
func bucketRegionMismatch(bucketName string, bucketRegion string, groupRegion string) error {
	if bucketRegion == "" || strings.EqualFold(bucketRegion, groupRegion) {
		return nil
	}

	return cspResourceError{kind: ErrBucketRegionMismatch, msg: "bucket '" + bucketName + "' is in region '" +
		bucketRegion + "', source group region is '" + groupRegion + "'"}
}

// checkBucket finds the bucket and refuses it when it lives outside the
// group's region (conn.Region).
func checkBucket(conn spider.Conn, bucketName string) (*spider.S3BucketInfo, error) {
	b, err := findBucketByName(conn, bucketName)
	if err != nil {
		return nil, err
	}
	if err := bucketRegionMismatch(b.Name, b.Region, conn.Region); err != nil {
		return nil, err
	}

	return b, nil
}

// collectBucket reads the bucket's details at the endpoint of the region it
// lives in and stores them with that region.
func collectBucket(conn spider.Conn, connID string, bucketName string, countObjects bool) error {
	b, err := checkBucket(conn, bucketName)
	if err != nil {
		return err
	}

	detailConn := conn
	if b.Region != "" {
		detailConn.Region = b.Region
	}
	d, err := getS3BucketDetail(detailConn, b.Name, countObjects)
	if err != nil {
		return err
	}
	if d.CreationDate == "" {
		d.CreationDate = b.CreationDate
	}
	for _, e := range d.Errors {
		logger.Println(logger.WARN, false, "bucket '"+b.Name+"': "+e+" (ConnectionID="+connID+")")
	}

	return storeBucketDetail(connID, detailConn.Region, b.Region, d)
}

// connLabel names the CSP connection in an error message; it never includes
// the credential.
func connLabel(conn spider.Conn) string {
	label := "in " + strings.ToLower(conn.Provider) + " region '" + conn.Region + "'"
	if conn.Zone != "" {
		label += " zone '" + conn.Zone + "'"
	}
	return label
}

// matchBucket picks the bucket named bucketName out of a listing.
func matchBucket(buckets []spider.S3BucketInfo, bucketName string) *spider.S3BucketInfo {
	if bucketName == "" {
		return nil
	}

	for i := range buckets {
		if buckets[i].Name == bucketName {
			return &buckets[i]
		}
	}

	return nil
}

// checkCSPConnection verifies that the CSP driver can identify the resource described
// by ci, WITHOUT persisting anything. This backs connection_status on
// registration/refresh, and the refusal of a resource that is not there when a
// connection is registered or pointed at another resource (cspResourceRejected).
// Actual data collection + persistence is done by import (refreshCSPConnection).
func checkCSPConnection(sg *model.SourceGroup, ci *model.ConnectionInfo) error {
	if ci.ResourceID == "" {
		return errors.New("resource_id is empty")
	}

	return withCSPConn(sg, ci.Zone, func(conn spider.Conn) error {
		return checkCSPResource(conn, ci)
	})
}

// checkCSPResource is the driver side of checkCSPConnection, given the
// connection to the CSP.
func checkCSPResource(conn spider.Conn, ci *model.ConnectionInfo) error {
	switch ci.ResourceType {
	case "vm":
		_, err := findVMByID(conn, ci.ResourceID)
		return err
	case "k8s":
		_, err := findClusterByID(conn, ci.ResourceID)
		return err
	case "object_storage":
		_, err := checkBucket(conn, ci.ResourceID)
		return err
	case serverCommon.ResourceTypeRDBMS:
		_, err := findRDBMSByID(conn, ci.ResourceID)
		return err
	case serverCommon.ResourceTypeNLB:
		_, err := findNLBByID(conn, ci.ResourceID)
		return err
	default:
		return errors.New("unsupported resource_type: " + ci.ResourceType)
	}
}

// refreshCSPConnection asks the CSP driver for the resource described by ci and
// stores the adapted result in the relevant Saved*Info table. Used by import
// (collection + persistence), not by the status-only refresh path.
func refreshCSPConnection(sg *model.SourceGroup, ci *model.ConnectionInfo, opts importInfraOptions) error {
	if ci.ResourceID == "" {
		return errors.New("resource_id is empty")
	}

	// The credential goes to the driver in memory for this call only.
	return withCSPConn(sg, ci.Zone, func(conn spider.Conn) error {
		return collectCSPResource(conn, sg, ci, opts)
	})
}

// collectCSPResource is the driver side of refreshCSPConnection, given the
// connection to the CSP.
func collectCSPResource(conn spider.Conn, sg *model.SourceGroup, ci *model.ConnectionInfo, opts importInfraOptions) error {
	switch ci.ResourceType {
	case "vm":
		vm, err := getCSPVM(conn, cspVMIdentifier(ci.ResourceID))
		if err != nil {
			return err
		}
		return upsertSavedCSPData(ci.ID, buildCSPInfo(conn, sg, vm))
	case "k8s":
		cl, err := findClusterByID(conn, ci.ResourceID)
		if err != nil {
			return err
		}
		return upsertSavedK8s(ci.ID, clusterInfoToK8s(cl))
	case "object_storage":
		return collectBucket(conn, ci.ID, ci.ResourceID, opts.CountObjects)
	case serverCommon.ResourceTypeRDBMS:
		payload, err := collectRDBMS(conn, ci.ResourceID)
		if err != nil {
			return err
		}
		return upsertSavedRDBMS(ci.ID, payload)
	case serverCommon.ResourceTypeNLB:
		n, err := findNLBByID(conn, ci.ResourceID)
		if err != nil {
			return err
		}
		return upsertSavedNLB(ci.ID, nlbInfoToNLB(n))
	default:
		return errors.New("unsupported resource_type: " + ci.ResourceType)
	}
}

package controller

import (
	"encoding/json"
	"errors"
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
	return buildCSPInfoWith(driverNetworkLookup(conn), sg, vm)
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

// bucketToData maps a CSP bucket into the agent's data.DataInfo shape, reusing
// the MinIO sub-structure as the object-storage carrier. Address carries the
// region the bucket was looked up in.
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

// upsertSavedData writes (or replaces) SavedDataInfo for a connection. cspData,
// when not nil, is stored as the CSP-reported bucket.
func upsertSavedData(connID string, payload any, cspData any) error {
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
		Status:       model.ConnectionInfoStatusSuccess,
		SavedTime:    time.Now(),
	}
	if existing, _ := dao.SavedDataInfoGet(connID); existing != nil {
		return dao.SavedDataInfoUpdate(rec)
	}
	_, err = dao.SavedDataInfoRegister(rec)
	return err
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

	return nil, errors.New("cluster '" + resourceID + "' was not found " + connLabel(conn))
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

	return nil, errors.New("rdbms '" + resourceID + "' was not found " + connLabel(conn))
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

// findBucketByName looks an object storage bucket up the same way clusters are
// looked up: the listing goes to the CSP, so existence is decided there.
//
// The region is not in that listing, so it is read from GetS3BucketLocation as
// a best effort, which leaves it empty (see its comment) rather than failing
// the whole lookup.
func findBucketByName(conn spider.Conn, bucketName string) (*spider.S3BucketInfo, error) {
	buckets, err := spider.ListS3Buckets(conn)
	if err != nil {
		return nil, err
	}

	found := matchBucket(buckets, bucketName)
	if found == nil {
		return nil, errors.New("bucket '" + bucketName + "' was not found " + connLabel(conn))
	}

	if loc, err := spider.GetS3BucketLocation(conn, bucketName); err == nil && loc != nil {
		found.Region = loc.Region
	}

	return found, nil
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
// registration/refresh — those paths only report reachability. Actual data
// collection + persistence is done by import (refreshCSPConnection).
func checkCSPConnection(sg *model.SourceGroup, ci *model.ConnectionInfo) error {
	if ci.ResourceID == "" {
		return errors.New("resource_id is empty")
	}

	return withCSPConn(sg, ci.Zone, func(conn spider.Conn) error {
		switch ci.ResourceType {
		case "vm":
			_, err := spider.GetCSPVM(conn, cspVMIdentifier(ci.ResourceID))
			return err
		case "k8s":
			_, err := findClusterByID(conn, ci.ResourceID)
			return err
		case "object_storage":
			_, err := findBucketByName(conn, ci.ResourceID)
			return err
		case serverCommon.ResourceTypeRDBMS:
			_, err := findRDBMSByID(conn, ci.ResourceID)
			return err
		default:
			return errors.New("unsupported resource_type: " + ci.ResourceType)
		}
	})
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
		switch ci.ResourceType {
		case "vm":
			vm, err := spider.GetCSPVM(conn, cspVMIdentifier(ci.ResourceID))
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
			b, err := findBucketByName(conn, ci.ResourceID)
			if err != nil {
				return err
			}
			d, err := spider.GetS3BucketDetail(conn, b.Name, opts.CountObjects)
			if err != nil {
				return err
			}
			if d.CreationDate == "" {
				d.CreationDate = b.CreationDate
			}
			for _, e := range d.Errors {
				logger.Println(logger.WARN, false, "bucket '"+b.Name+"': "+e+" (ConnectionID="+ci.ID+")")
			}
			return upsertSavedData(ci.ID, bucketToData(conn.Region, d), bucketToSourceProperty(d))
		case serverCommon.ResourceTypeRDBMS:
			payload, err := collectRDBMS(conn, ci.ResourceID)
			if err != nil {
				return err
			}
			return upsertSavedRDBMS(ci.ID, payload)
		default:
			return errors.New("unsupported resource_type: " + ci.ResourceType)
		}
	})
}

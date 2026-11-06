package spider

import (
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

func convertList[T any](in any) ([]T, error) {
	out, err := convertJSON[[]T](in)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []T{}
	}
	return out, nil
}

// ListVM returns every VM the CSP has, as cb-spider's GET /allvminfo does.
// cb-spider's GET /vm lists only what its meta-DB holds, which never
// includes a source VM cb-spider did not create.
func ListVM(c Conn) ([]VMInfo, error) {
	return withTimeout(driverCallTimeout, func() ([]VMInfo, error) { return listVM(c) })
}

func listVM(c Conn) ([]VMInfo, error) {
	conn, err := connect(c)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateVMHandler()
	if err != nil {
		return nil, err
	}
	list, err := handler.ListVM()
	if err != nil {
		return nil, err
	}
	return convertList[VMInfo](list)
}

// GetCSPVM fetches a VM by its CSP native ID, as cb-spider's GET /cspvm/{Id}
// does: the driver is asked live, with no name lookup in a meta-DB.
func GetCSPVM(c Conn, cspID string) (*VMInfo, error) {
	return withTimeout(driverCallTimeout, func() (*VMInfo, error) { return getCSPVM(c, cspID) })
}

func getCSPVM(c Conn, cspID string) (*VMInfo, error) {
	if err := mustNonEmpty("Id", cspID); err != nil {
		return nil, err
	}
	conn, err := connect(c)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateVMHandler()
	if err != nil {
		return nil, err
	}
	info, err := handler.GetVM(cres.IID{SystemId: cspID})
	if err != nil {
		return nil, err
	}
	out, err := convertJSON[VMInfo](info)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListVPC returns every VPC the CSP has WITH full detail (CIDR, subnets), as
// cb-spider's GET /allvpcinfo does.
func ListVPC(c Conn) ([]VPCInfo, error) {
	return withTimeout(driverCallTimeout, func() ([]VPCInfo, error) { return listVPC(c) })
}

func listVPC(c Conn) ([]VPCInfo, error) {
	conn, err := connect(c)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateVPCHandler()
	if err != nil {
		return nil, err
	}
	list, err := handler.ListVPC()
	if err != nil {
		return nil, err
	}
	return convertList[VPCInfo](list)
}

// GetVPC fetches one VPC with full detail by the IID a driver reported for
// it (e.g. a VM's VpcIID). The IID is passed through as is: aws, alibaba, ncp,
// gcp and nhn read its SystemId, while azure, ibm and openstack also accept a
// NameId.
func GetVPC(c Conn, iid IID) (*VPCInfo, error) {
	return withTimeout(driverCallTimeout, func() (*VPCInfo, error) { return getVPC(c, iid) })
}

func getVPC(c Conn, iid IID) (*VPCInfo, error) {
	if iid.NameId == "" && iid.SystemId == "" {
		return nil, mustNonEmpty("VpcIID", "")
	}
	conn, err := connect(c)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateVPCHandler()
	if err != nil {
		return nil, err
	}
	info, err := handler.GetVPC(cres.IID{NameId: iid.NameId, SystemId: iid.SystemId})
	if err != nil {
		return nil, err
	}
	out, err := convertJSON[VPCInfo](info)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetSecurityGroup fetches one security group with its rules by the IID a
// driver reported for it (e.g. one of a VM's SecurityGroupIIds), passed
// through as GetVPC does.
func GetSecurityGroup(c Conn, iid IID) (*SecurityGroupInfo, error) {
	return withTimeout(driverCallTimeout, func() (*SecurityGroupInfo, error) { return getSecurityGroup(c, iid) })
}

func getSecurityGroup(c Conn, iid IID) (*SecurityGroupInfo, error) {
	if iid.NameId == "" && iid.SystemId == "" {
		return nil, mustNonEmpty("SecurityGroupIID", "")
	}
	conn, err := connect(c)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateSecurityHandler()
	if err != nil {
		return nil, err
	}
	info, err := handler.GetSecurity(cres.IID{NameId: iid.NameId, SystemId: iid.SystemId})
	if err != nil {
		return nil, err
	}
	out, err := convertJSON[SecurityGroupInfo](info)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListSecurityGroup returns every security group the CSP has, as
// cb-spider's GET /allsecuritygroupinfo does.
func ListSecurityGroup(c Conn) ([]SecurityGroupInfo, error) {
	return withTimeout(driverCallTimeout, func() ([]SecurityGroupInfo, error) { return listSecurityGroup(c) })
}

func listSecurityGroup(c Conn) ([]SecurityGroupInfo, error) {
	conn, err := connect(c)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateSecurityHandler()
	if err != nil {
		return nil, err
	}
	list, err := handler.ListSecurity()
	if err != nil {
		return nil, err
	}
	return convertList[SecurityGroupInfo](list)
}

// ListNLB returns every NLB the CSP has, as cb-spider's GET /allnlbinfo does.
//
// The values are raw driver output: cb-spider's transformArgsToUpper() never
// runs on this path, so Type/Scope/Protocol keep whatever case the driver
// produced. Normalising that is the caller's job - see nlbInfoToNLB().
func ListNLB(c Conn) ([]NLBInfo, error) {
	return withTimeout(driverCallTimeout, func() ([]NLBInfo, error) { return listNLB(c) })
}

func listNLB(c Conn) ([]NLBInfo, error) {
	conn, err := connect(c)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateNLBHandler()
	if err != nil {
		return nil, err
	}
	list, err := handler.ListNLB()
	if err != nil {
		return nil, err
	}
	return convertList[NLBInfo](list)
}

// ListCluster returns every Kubernetes cluster the CSP has, as
// cb-spider's GET /allclusterinfo does. A cluster cb-spider did not create
// carries an empty NameId, so SystemId is the only identifier it has.
func ListCluster(c Conn) ([]ClusterInfo, error) {
	return withTimeout(driverCallTimeout, func() ([]ClusterInfo, error) { return listCluster(c) })
}

func listCluster(c Conn) ([]ClusterInfo, error) {
	conn, err := connect(c)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateClusterHandler()
	if err != nil {
		return nil, err
	}
	list, err := handler.ListCluster()
	if err != nil {
		return nil, err
	}
	return convertList[ClusterInfo](list)
}

// GetCluster fetches a Kubernetes cluster by its CSP SystemId.
func GetCluster(c Conn, systemID string) (*ClusterInfo, error) {
	return withTimeout(driverCallTimeout, func() (*ClusterInfo, error) { return getCluster(c, systemID) })
}

func getCluster(c Conn, systemID string) (*ClusterInfo, error) {
	if err := mustNonEmpty("Id", systemID); err != nil {
		return nil, err
	}
	conn, err := connect(c)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateClusterHandler()
	if err != nil {
		return nil, err
	}
	info, err := handler.GetCluster(cres.IID{SystemId: systemID})
	if err != nil {
		return nil, err
	}
	out, err := convertJSON[ClusterInfo](info)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListRDBMS returns every managed database instance the CSP has. Like ListVM,
// it asks the driver directly, so instances cb-spider did not create are
// included.
func ListRDBMS(c Conn) ([]RDBMSInfo, error) {
	return withTimeout(driverCallTimeout, func() ([]RDBMSInfo, error) { return listRDBMS(c) })
}

func listRDBMS(c Conn) ([]RDBMSInfo, error) {
	conn, err := connect(c)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateRDBMSHandler()
	if err != nil {
		return nil, err
	}
	list, err := handler.ListRDBMS()
	if err != nil {
		return nil, err
	}
	return convertList[RDBMSInfo](list)
}

// GetDBSpec fetches the vCPU and memory size of a managed database spec, which
// RDBMSInfo names (DBSpec) but does not describe.
func GetDBSpec(c Conn, dbEngine, name string) (*DBSpecInfo, error) {
	return withTimeout(driverCallTimeout, func() (*DBSpecInfo, error) { return getDBSpec(c, dbEngine, name) })
}

func getDBSpec(c Conn, dbEngine, name string) (*DBSpecInfo, error) {
	if err := mustNonEmpty("DBEngine", dbEngine); err != nil {
		return nil, err
	}
	if err := mustNonEmpty("Name", name); err != nil {
		return nil, err
	}
	conn, err := connect(c)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateDBSpecHandler()
	if err != nil {
		return nil, err
	}
	info, err := handler.GetDBSpec(dbEngine, name)
	if err != nil {
		return nil, err
	}
	out, err := convertJSON[DBSpecInfo](info)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetVMStatus asks the driver for a VM's current state (Running, Suspended,
// ...). The IID is passed through as GetVPC does, so the VM's IID as GetVM
// reported it serves drivers that read either field.
func GetVMStatus(c Conn, iid IID) (string, error) {
	return withTimeout(driverCallTimeout, func() (string, error) { return getVMStatus(c, iid) })
}

func getVMStatus(c Conn, iid IID) (string, error) {
	if iid.NameId == "" && iid.SystemId == "" {
		return "", mustNonEmpty("VMIID", "")
	}
	conn, err := connect(c)
	if err != nil {
		return "", err
	}
	handler, err := conn.CreateVMHandler()
	if err != nil {
		return "", err
	}
	status, err := handler.GetVMStatus(cres.IID{NameId: iid.NameId, SystemId: iid.SystemId})
	if err != nil {
		return "", err
	}
	return string(status), nil
}

// GetKeyPair fetches a key pair by the IID a VM reported for it. Only the
// identifier and fingerprint are kept; see KeyPairInfo.
func GetKeyPair(c Conn, iid IID) (*KeyPairInfo, error) {
	return withTimeout(driverCallTimeout, func() (*KeyPairInfo, error) { return getKeyPair(c, iid) })
}

func getKeyPair(c Conn, iid IID) (*KeyPairInfo, error) {
	if iid.NameId == "" && iid.SystemId == "" {
		return nil, mustNonEmpty("KeyPairIID", "")
	}
	conn, err := connect(c)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateKeyPairHandler()
	if err != nil {
		return nil, err
	}
	info, err := handler.GetKey(cres.IID{NameId: iid.NameId, SystemId: iid.SystemId})
	if err != nil {
		return nil, err
	}
	return &KeyPairInfo{
		IId:         IID{NameId: info.IId.NameId, SystemId: info.IId.SystemId},
		Fingerprint: info.Fingerprint,
	}, nil
}

// GetDisk fetches a disk by the IID a VM reported for it (one of DataDiskIIDs).
func GetDisk(c Conn, iid IID) (*DiskInfo, error) {
	return withTimeout(driverCallTimeout, func() (*DiskInfo, error) { return getDisk(c, iid) })
}

func getDisk(c Conn, iid IID) (*DiskInfo, error) {
	if iid.NameId == "" && iid.SystemId == "" {
		return nil, mustNonEmpty("DiskIID", "")
	}
	conn, err := connect(c)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateDiskHandler()
	if err != nil {
		return nil, err
	}
	info, err := handler.GetDisk(cres.IID{NameId: iid.NameId, SystemId: iid.SystemId})
	if err != nil {
		return nil, err
	}
	out, err := convertJSON[DiskInfo](info)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetVMSpec fetches the size behind a VM spec name (VMInfo.VMSpecName) in the
// connection's region.
func GetVMSpec(c Conn, name string) (*VMSpecInfo, error) {
	return withTimeout(driverCallTimeout, func() (*VMSpecInfo, error) { return getVMSpec(c, name) })
}

func getVMSpec(c Conn, name string) (*VMSpecInfo, error) {
	if err := mustNonEmpty("Name", name); err != nil {
		return nil, err
	}
	conn, err := connect(c)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateVMSpecHandler()
	if err != nil {
		return nil, err
	}
	info, err := handler.GetVMSpec(name)
	if err != nil {
		return nil, err
	}
	out, err := convertJSON[VMSpecInfo](info)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetImage fetches a public image by the IID a VM reported for it
// (VMInfo.ImageIId).
func GetImage(c Conn, iid IID) (*ImageInfo, error) {
	return withTimeout(driverCallTimeout, func() (*ImageInfo, error) { return getImage(c, iid) })
}

func getImage(c Conn, iid IID) (*ImageInfo, error) {
	if iid.NameId == "" && iid.SystemId == "" {
		return nil, mustNonEmpty("ImageIID", "")
	}
	conn, err := connect(c)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateImageHandler()
	if err != nil {
		return nil, err
	}
	info, err := handler.GetImage(cres.IID{NameId: iid.NameId, SystemId: iid.SystemId})
	if err != nil {
		return nil, err
	}
	out, err := convertJSON[ImageInfo](info)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

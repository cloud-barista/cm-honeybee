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

// ListSecurityGroup returns every security group the CSP has, as
// cb-spider's GET /allsecuritygroupinfo does.
func ListSecurityGroup(c Conn) ([]SecurityGroupInfo, error) {
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

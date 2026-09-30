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

// DirectListVM returns every VM the CSP has, as ListAllVMInfo does.
func DirectListVM(c Conn) ([]VMInfo, error) {
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

// DirectGetCSPVM fetches a VM by its CSP native ID, as GetCSPVM does.
func DirectGetCSPVM(c Conn, cspID string) (*VMInfo, error) {
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

// DirectListVPC returns every VPC the CSP has, as ListAllVPCInfo does.
func DirectListVPC(c Conn) ([]VPCInfo, error) {
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

// DirectListSecurityGroup returns every security group the CSP has, as
// ListAllSecurityGroupInfo does.
func DirectListSecurityGroup(c Conn) ([]SecurityGroupInfo, error) {
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

// DirectListNLB returns every NLB the CSP has, as ListAllNLBInfo does. The
// values are raw driver output; see ListAllNLBInfo.
func DirectListNLB(c Conn) ([]NLBInfo, error) {
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

// DirectListCluster returns every Kubernetes cluster the CSP has, as
// ListAllClusterInfo does.
func DirectListCluster(c Conn) ([]ClusterInfo, error) {
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

// DirectGetCluster fetches a Kubernetes cluster by its CSP SystemId.
func DirectGetCluster(c Conn, systemID string) (*ClusterInfo, error) {
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

package spider

// SubnetInfo mirrors cb-spider's SubnetInfo (subset used by honeybee).
type SubnetInfo struct {
	IId       IID    `json:"IId"`
	IPv4_CIDR string `json:"IPv4_CIDR"`
	Zone      string `json:"Zone"`
}

// VPCInfo mirrors cb-spider's VPCInfo (subset used by honeybee).
type VPCInfo struct {
	IId            IID          `json:"IId"`
	IPv4_CIDR      string       `json:"IPv4_CIDR"`
	SubnetInfoList []SubnetInfo `json:"SubnetInfoList"`
}

// SecurityRuleInfo mirrors cb-spider's SecurityRuleInfo.
type SecurityRuleInfo struct {
	Direction  string `json:"Direction"`
	IPProtocol string `json:"IPProtocol"`
	FromPort   string `json:"FromPort"`
	ToPort     string `json:"ToPort"`
	CIDR       string `json:"CIDR"`
}

// SecurityGroupInfo mirrors cb-spider's SecurityInfo (subset used by honeybee).
type SecurityGroupInfo struct {
	IId           IID                `json:"IId"`
	VpcIID        IID                `json:"VpcIID"`
	SecurityRules []SecurityRuleInfo `json:"SecurityRules"`
}

type allVPCInfoResp struct {
	AllListInfo struct {
		MappedInfoList  []VPCInfo `json:"MappedInfoList"`
		OnlyCSPInfoList []VPCInfo `json:"OnlyCSPInfoList"`
	} `json:"AllListInfo"`
}

type allSGInfoResp struct {
	AllListInfo struct {
		MappedInfoList  []SecurityGroupInfo `json:"MappedInfoList"`
		OnlyCSPInfoList []SecurityGroupInfo `json:"OnlyCSPInfoList"`
	} `json:"AllListInfo"`
}

// ListAllVPCInfo returns every VPC visible through the connection WITH full
// detail (CIDR, subnets), including VPCs that cb-spider does not manage. This is
// how honeybee obtains detail for a source VM's existing VPC: cb-spider has no
// live "get by CSP id" for an unmanaged VPC, but the all-info listing queries
// the CSP live and returns the unmanaged ones under OnlyCSPInfoList.
func ListAllVPCInfo(connectionName string) ([]VPCInfo, error) {
	if err := mustNonEmpty("ConnectionName", connectionName); err != nil {
		return nil, err
	}
	var r allVPCInfoResp
	if err := do("GET", "/allvpcinfo?ConnectionName="+encodePath(connectionName), nil, &r); err != nil {
		return nil, err
	}
	return append(append([]VPCInfo{}, r.AllListInfo.MappedInfoList...), r.AllListInfo.OnlyCSPInfoList...), nil
}

// ListAllSecurityGroupInfo returns every security group visible through the
// connection WITH full detail (rules), including unmanaged ones (OnlyCSPInfoList).
func ListAllSecurityGroupInfo(connectionName string) ([]SecurityGroupInfo, error) {
	if err := mustNonEmpty("ConnectionName", connectionName); err != nil {
		return nil, err
	}
	var r allSGInfoResp
	if err := do("GET", "/allsecuritygroupinfo?ConnectionName="+encodePath(connectionName), nil, &r); err != nil {
		return nil, err
	}
	return append(append([]SecurityGroupInfo{}, r.AllListInfo.MappedInfoList...), r.AllListInfo.OnlyCSPInfoList...), nil
}

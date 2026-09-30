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

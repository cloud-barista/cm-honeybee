package spider

// KeyValue mirrors spider.KeyValue.
type KeyValue struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// IID mirrors spider.IID.
type IID struct {
	NameId   string `json:"NameId"`
	SystemId string `json:"SystemId"`
}

// CloudOSMetaInfo mirrors spider.cim.CloudOSMetaInfo.
type CloudOSMetaInfo struct {
	Credential           []string `json:"Credential"`
	CredentialCSP        []string `json:"CredentialCSP"`
	Region               []string `json:"Region"`
	DefaultRegionToQuery []string `json:"DefaultRegionToQuery"`
	RootDiskType         []string `json:"RootDiskType"`
	RootDiskSize         []string `json:"RootDiskSize"`
	DiskType             []string `json:"DiskType"`
	DiskSize             []string `json:"DiskSize"`
	IdMaxLength          []string `json:"IdMaxLength"`
}

// CredentialInfo mirrors spider.cim.CredentialInfo.
type CredentialInfo struct {
	CredentialName   string     `json:"CredentialName"`
	ProviderName     string     `json:"ProviderName"`
	KeyValueInfoList []KeyValue `json:"KeyValueInfoList"`
}

// RegionInfo mirrors spider.cim.RegionInfo.
type RegionInfo struct {
	RegionName        string     `json:"RegionName"`
	ProviderName      string     `json:"ProviderName"`
	AvailableZoneList []string   `json:"AvailableZoneList,omitempty"`
	KeyValueInfoList  []KeyValue `json:"KeyValueInfoList"`
	// Region/Zone are the driver-level fields present in VMInfo.Region
	// (e.g. {"Region":"koreacentral","Zone":"1"}); they are empty in the
	// region-registration/list shape that uses RegionName/KeyValueInfoList.
	Region string `json:"Region,omitempty"`
	Zone   string `json:"Zone,omitempty"`
}

// ConnectionConfigInfo mirrors spider.cim.ConnectionConfigInfo.
type ConnectionConfigInfo struct {
	ConfigName     string `json:"ConfigName"`
	ProviderName   string `json:"ProviderName"`
	DriverName     string `json:"DriverName"`
	CredentialName string `json:"CredentialName"`
	RegionName     string `json:"RegionName"`
}

// VMInfo is a subset of spider.VMInfo.
type VMInfo struct {
	IId               IID        `json:"IId"`
	ImageIId          IID        `json:"ImageIId"`
	VMSpecName        string     `json:"VMSpecName"`
	VpcIID            IID        `json:"VpcIID"`
	SubnetIID         IID        `json:"SubnetIID"`
	NetworkInterface  string     `json:"NetworkInterface"`
	PublicIP          string     `json:"PublicIP"`
	PublicDNS         string     `json:"PublicDNS"`
	PrivateIP         string     `json:"PrivateIP"`
	PrivateDNS        string     `json:"PrivateDNS"`
	SecurityGroupIIds []IID      `json:"SecurityGroupIIds"`
	RootDiskType      string     `json:"RootDiskType"`
	RootDiskSize      string     `json:"RootDiskSize"`
	RootDeviceName    string     `json:"RootDeviceName"`
	DataDiskIIDs      []IID      `json:"DataDiskIIDs"`
	VMUserId          string     `json:"VMUserId"`
	StartTime         string     `json:"StartTime"`
	Region            RegionInfo `json:"Region"`
	Platform          string     `json:"Platform"`
	AccessPoint       string     `json:"AccessPoint"`
	KeyValueList      []KeyValue `json:"KeyValueList"`
	TagList           []KeyValue `json:"TagList"`
}

// ClusterInfo is a subset of spider.ClusterInfo.
type ClusterInfo struct {
	IId           IID             `json:"IId"`
	Version       string          `json:"Version"`
	Status        string          `json:"Status"`
	CreatedTime   string          `json:"CreatedTime"`
	Network       any             `json:"Network,omitempty"`
	NodeGroupList []NodeGroupInfo `json:"NodeGroupList,omitempty"`
	AccessInfo    any             `json:"AccessInfo,omitempty"`
	Addons        any             `json:"Addons,omitempty"`
	KeyValueList  []KeyValue      `json:"KeyValueList,omitempty"`
	TagList       []KeyValue      `json:"TagList,omitempty"`
}

// NodeGroupInfo is a minimal representation of spider.NodeGroupInfo.
type NodeGroupInfo struct {
	IId             IID    `json:"IId"`
	ImageIID        IID    `json:"ImageIID"`
	VMSpecName      string `json:"VMSpecName"`
	RootDiskType    string `json:"RootDiskType"`
	RootDiskSize    string `json:"RootDiskSize"`
	OnAutoScaling   bool   `json:"OnAutoScaling"`
	DesiredNodeSize int    `json:"DesiredNodeSize"`
	MinNodeSize     int    `json:"MinNodeSize"`
	MaxNodeSize     int    `json:"MaxNodeSize"`
	Status          string `json:"Status"`
}

// BucketIID mirrors spider.BucketIID - stripped down to the fields we use.
type BucketIID struct {
	NameId   string `json:"NameId,omitempty"`
	SystemId string `json:"SystemId,omitempty"`
}

// S3BucketInfo aggregates the fields cb-spider reports per bucket.
type S3BucketInfo struct {
	Name         string `json:"Name"`
	CreationDate string `json:"CreationDate"`
	Region       string `json:"Region,omitempty"`
}

// NLBInfo mirrors cb-spider's NLBInfo
// (cloud-driver/interfaces/resources/NLBHandler.go).
//
// CreatedTime is a time.Time on the spider side; we keep the RFC3339 string it
// marshals to, because an unset value arrives as "0001-01-01T00:00:00Z" and we
// want to drop it rather than parse it.
type NLBInfo struct {
	IId           IID               `json:"IId"`
	VpcIID        IID               `json:"VpcIID"`
	Type          string            `json:"Type"`  // PUBLIC | INTERNAL - see normalizeNLBType
	Scope         string            `json:"Scope"` // REGION | GLOBAL
	Listener      ListenerInfo      `json:"Listener"`
	VMGroup       VMGroupInfo       `json:"VMGroup"`
	HealthChecker HealthCheckerInfo `json:"HealthChecker"`
	CreatedTime   string            `json:"CreatedTime"`
	TagList       []KeyValue        `json:"TagList,omitempty"`
	KeyValueList  []KeyValue        `json:"KeyValueList,omitempty"`
}

// ListenerInfo is the frontend of an NLB.
type ListenerInfo struct {
	Protocol string `json:"Protocol"`
	// IP is empty on several drivers, and AWS joins multiple AZ addresses with
	// commas, so it is not directly parseable as a single address.
	IP           string     `json:"IP"`
	Port         string     `json:"Port"`
	DNSName      string     `json:"DNSName"`
	CspID        string     `json:"CspID,omitempty"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty"`
}

// VMGroupInfo is the backend of an NLB. cb-spider declares VMs as *[]IID; a
// plain slice decodes both the array and a JSON null, so the pointer is not
// needed here.
type VMGroupInfo struct {
	Protocol     string     `json:"Protocol"`
	Port         string     `json:"Port"`
	VMs          []IID      `json:"VMs"`
	CspID        string     `json:"CspID,omitempty"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty"`
}

// HealthCheckerInfo is the health check attached to an NLB's VM group.
type HealthCheckerInfo struct {
	Protocol  string `json:"Protocol"`
	Port      string `json:"Port"`
	Interval  int    `json:"Interval"`
	Timeout   int    `json:"Timeout"` // Azure reports -1 for "not supported"
	Threshold int    `json:"Threshold"`

	CspID        string     `json:"CspID,omitempty"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty"`
}

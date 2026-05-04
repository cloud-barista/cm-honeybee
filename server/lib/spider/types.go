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
	IId              IID        `json:"IId"`
	ImageIId         IID        `json:"ImageIId"`
	VMSpecName       string     `json:"VMSpecName"`
	VpcIID           IID        `json:"VpcIID"`
	SubnetIID        IID        `json:"SubnetIID"`
	NetworkInterface string     `json:"NetworkInterface"`
	PublicIP         string     `json:"PublicIP"`
	PublicDNS        string     `json:"PublicDNS"`
	PrivateIP        string     `json:"PrivateIP"`
	PrivateDNS       string     `json:"PrivateDNS"`
	RootDiskType     string     `json:"RootDiskType"`
	RootDiskSize     string     `json:"RootDiskSize"`
	RootDeviceName   string     `json:"RootDeviceName"`
	DataDiskIIDs     []IID      `json:"DataDiskIIDs"`
	VMUserId         string     `json:"VMUserId"`
	StartTime        string     `json:"StartTime"`
	Region           RegionInfo `json:"Region"`
	Platform         string     `json:"Platform"`
	AccessPoint      string     `json:"AccessPoint"`
	KeyValueList     []KeyValue `json:"KeyValueList"`
	TagList          []KeyValue `json:"TagList"`
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

// BucketIID mirrors spider.BucketIID — stripped down to the fields we use.
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

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

// RDBMSInfo is a subset of cb-spider's RDBMSInfo
// (cloud-driver/interfaces/resources/RDBMSHandler.go).
//
// MasterUserPassword is deliberately absent: cb-spider only uses it on create,
// and leaving the field out of this struct means a driver that ever echoes it
// back cannot get it decoded, stored or served.
//
// CreatedTime is a time.Time on the spider side; it is kept as the RFC3339
// string it marshals to, as NLBInfo does.
type RDBMSInfo struct {
	IId    IID `json:"IId"`
	VpcIID IID `json:"VpcIID"`

	DBEngine        string `json:"DBEngine"` // mysql | mariadb | postgresql - AWS reports its own "postgres"
	DBEngineVersion string `json:"DBEngineVersion"`

	DBSpec         string `json:"DBSpec"`                   // CSP instance class/type, e.g. db.t3.medium
	DBInstanceType string `json:"DBInstanceType,omitempty"` // meaning differs per CSP; not a role

	StorageType string `json:"StorageType,omitempty"`
	StorageSize string `json:"StorageSize"` // GB
	Iops        string `json:"Iops,omitempty"`

	MasterUserName      string `json:"MasterUserName"`
	HighAvailability    bool   `json:"HighAvailability,omitempty"`
	BackupRetentionDays int    `json:"BackupRetentionDays,omitempty"`
	PublicAccess        bool   `json:"PublicAccess,omitempty"`
	Endpoint            string `json:"Endpoint,omitempty"` // "host:port" on most drivers, host only on some
	Encryption          bool   `json:"Encryption,omitempty"`
	DeletionProtection  bool   `json:"DeletionProtection,omitempty"`

	Status       string     `json:"Status,omitempty"`
	CreatedTime  string     `json:"CreatedTime,omitempty"`
	TagList      []KeyValue `json:"TagList,omitempty"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty"`
}

// VCpuInfo mirrors cb-spider's VCpuInfo. Count and ClockGHz are "-1" when the
// CSP does not expose them.
type VCpuInfo struct {
	Count    string `json:"Count"`
	ClockGHz string `json:"ClockGHz,omitempty"`
}

// DBSpecInfo is a subset of cb-spider's DBSpecInfo
// (cloud-driver/interfaces/resources/DBSpecHandler.go): the compute size behind
// an RDBMSInfo.DBSpec name, which RDBMSInfo itself does not carry.
type DBSpecInfo struct {
	Region     string   `json:"Region"`
	DBEngine   string   `json:"DBEngine"`
	Name       string   `json:"Name"`
	VCpu       VCpuInfo `json:"VCpu"`
	MemSizeMiB string   `json:"MemSizeMiB"` // "-1" when the CSP's unit cannot be confirmed
}

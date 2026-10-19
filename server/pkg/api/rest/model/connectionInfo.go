package model

const ConnectionInfoMaxLength = 200

const (
	ConnectionInfoStatusSuccess = "success"
	ConnectionInfoStatusFailed  = "failed"
)

// Access types of a db or minio connection: how the source is reached.
const (
	// AccessTypeDirect reaches the source from the honeybee server.
	AccessTypeDirect = "direct"
	// AccessTypeSSHTunnel reaches the source through the agent on the SSH host
	// given by the connection's SSH fields.
	AccessTypeSSHTunnel = "ssh-tunnel"
)

type ConnectionInfo struct {
	ID            string `gorm:"primaryKey" json:"id" validate:"required"`
	Name          string `gorm:"index:,column:name,unique;type:text collate nocase" json:"name" mapstructure:"name" validate:"required"`
	Description   string `gorm:"column:description" json:"description"`
	SourceGroupID string `gorm:"column:source_group_id" json:"source_group_id" validate:"required"`

	// SSH fields — required when parent SourceGroup.Type == "ssh".
	IPAddress  string `gorm:"column:ip_address" json:"ip_address,omitempty"`
	SSHPort    string `gorm:"column:ssh_port" json:"ssh_port,omitempty"`
	User       string `gorm:"column:user" json:"user,omitempty"`
	Password   string `gorm:"column:password" json:"password,omitempty"`
	PrivateKey string `gorm:"column:private_key" json:"private_key,omitempty"`
	PublicKey  string `gorm:"column:public_key" json:"public_key,omitempty"`

	// Kubeconfig — on-prem k8s credential (Type onprem + ResourceType k8s).
	Kubeconfig string `gorm:"column:kubeconfig" json:"kubeconfig,omitempty"`

	// CSP fields — required when parent SourceGroup.Type == "csp".
	// ResourceType: "vm" | "k8s" | "object_storage".
	ResourceType string `gorm:"column:resource_type" json:"resource_type,omitempty"`
	ResourceID   string `gorm:"column:resource_id" json:"resource_id,omitempty"`
	// Zone overrides the CSP zone for this connection's driver lookups.
	// Empty → the zone in the source group's region_name "<region>/<zone>", if any
	// (see splitRegionZone).
	Zone string `gorm:"column:zone" json:"zone,omitempty"`

	// FS fields — used when parent SourceGroup.Type == "fs". The host is reached
	// through the SSH fields above.
	// FSScanPath is the absolute path collected on the host. Empty → "/home".
	FSScanPath string `gorm:"column:fs_scan_path" json:"fs_scan_path,omitempty"`

	// DB fields — required when parent SourceGroup.Type == "db".
	// DBType: "mysql" | "mariadb" | "postgresql" | "mongodb".
	DBType string `gorm:"column:db_type" json:"db_type,omitempty"`
	// DBName selects one database. Empty → every user database on the server.
	DBName string `gorm:"column:db_name" json:"db_name,omitempty"`
	// DBAccessType: "direct" | "ssh-tunnel". ssh-tunnel goes through the SSH fields.
	DBAccessType string `gorm:"column:db_access_type" json:"db_access_type,omitempty"`
	DBHost       string `gorm:"column:db_host" json:"db_host,omitempty"`
	DBPort       string `gorm:"column:db_port" json:"db_port,omitempty"`
	DBUsername   string `gorm:"column:db_username" json:"db_username,omitempty"`
	// DBPassword is stored in OpenBao; the column stays empty.
	DBPassword string `gorm:"column:db_password" json:"db_password,omitempty"`
	// DBConnectTimeout is in seconds. 0 → the driver default.
	DBConnectTimeout int `gorm:"column:db_connect_timeout" json:"db_connect_timeout,omitempty"`
	// DBTLSMode: "disable" | "prefer" | "require" | "verify-ca" | "verify-full".
	// Empty → disable.
	DBTLSMode string `gorm:"column:db_tls_mode" json:"db_tls_mode,omitempty"`
	// DBTLSCAPEM is the CA certificate (PEM) verify-ca and verify-full check the
	// server against. A certificate is public, so it is stored as plain text.
	DBTLSCAPEM string `gorm:"column:db_tls_ca_pem" json:"db_tls_ca_pem,omitempty"`
	// DBAuthSource is the MongoDB authentication database. Not a secret, plain
	// text. Empty → "admin".
	DBAuthSource string `gorm:"column:db_auth_source" json:"db_auth_source,omitempty"`
	// DBPgSchema selects PostgreSQL schemas within DBName. Empty → every user schema.
	DBPgSchema []string `gorm:"column:db_pg_schema;serializer:json" json:"db_pg_schema,omitempty"`

	// MinIO fields — required when parent SourceGroup.Type == "minio". The
	// endpoint is derived from the source group's provider_name and region_name.
	// OSAccessType: "direct" | "ssh-tunnel". ssh-tunnel goes through the SSH fields.
	OSAccessType string `gorm:"column:os_access_type" json:"os_access_type,omitempty"`
	// OSEndpoint is the S3 endpoint (host[:port] or http(s) URL). Required for
	// onprem and openstack, overrides the derived endpoint otherwise, ignored for
	// azure (derived from the account name in OSAccessKeyId).
	OSEndpoint string `gorm:"column:os_endpoint" json:"os_endpoint,omitempty"`
	// OSAccessKeyId and OSSecretAccessKey are stored in OpenBao; the columns stay
	// empty.
	OSAccessKeyId     string `gorm:"column:os_access_key_id" json:"os_access_key_id,omitempty"`
	OSSecretAccessKey string `gorm:"column:os_secret_access_key" json:"os_secret_access_key,omitempty"`
	// OSUseSSL selects HTTPS for onprem and openstack endpoints. A scheme on
	// OSEndpoint takes precedence.
	OSUseSSL bool `gorm:"column:os_use_ssl" json:"os_use_ssl,omitempty"`
	// OSScanBucket is the bucket collected. On tencent it must carry the AppId
	// suffix ("<name>-<AppId>").
	OSScanBucket string `gorm:"column:os_scan_bucket" json:"os_scan_bucket,omitempty"`
	// OSScanPrefix narrows the collection to keys under this prefix. Empty → the
	// whole bucket.
	OSScanPrefix string `gorm:"column:os_scan_prefix" json:"os_scan_prefix,omitempty"`

	ConnectionStatus        string `gorm:"column:connection_status" json:"connection_status"`
	ConnectionFailedMessage string `gorm:"column:connection_failed_message" json:"connection_failed_message"`
	AgentStatus             string `gorm:"column:agent_status" json:"agent_status"`
	AgentFailedMessage      string `gorm:"column:agent_failed_message" json:"agent_failed_message"`
}

type CreateConnectionInfoReq struct {
	Name        string `json:"name" mapstructure:"name" validate:"required"`
	Description string `json:"description"`

	// SSH fields — required when parent SourceGroup.Type == "ssh".
	IPAddress  string `json:"ip_address,omitempty"`
	SSHPort    string `json:"ssh_port,omitempty"`
	User       string `json:"user,omitempty"`
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`

	// Kubeconfig — required when on-prem + ResourceType "k8s".
	Kubeconfig string `json:"kubeconfig,omitempty"`

	// CSP fields — required when parent SourceGroup.Type == "csp".
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   string `json:"resource_id,omitempty"`
	// Zone (optional) overrides the CSP zone for this connection. Empty → default.
	Zone string `json:"zone,omitempty"`

	// FS fields — used when parent SourceGroup.Type == "fs".
	// FSScanPath (optional) is the absolute path to collect. Empty → "/home";
	// "/" collects the whole host.
	FSScanPath string `json:"fs_scan_path,omitempty"`

	// DB fields — required when parent SourceGroup.Type == "db".
	DBType           string   `json:"db_type,omitempty"`
	DBName           string   `json:"db_name,omitempty"`
	DBAccessType     string   `json:"db_access_type,omitempty"`
	DBHost           string   `json:"db_host,omitempty"`
	DBPort           string   `json:"db_port,omitempty"`
	DBUsername       string   `json:"db_username,omitempty"`
	DBPassword       string   `json:"db_password,omitempty"`
	DBConnectTimeout int      `json:"db_connect_timeout,omitempty"`
	DBTLSMode        string   `json:"db_tls_mode,omitempty"`
	DBTLSCAPEM       string   `json:"db_tls_ca_pem,omitempty"`
	DBAuthSource     string   `json:"db_auth_source,omitempty"`
	DBPgSchema       []string `json:"db_pg_schema,omitempty"`

	// MinIO fields — required when parent SourceGroup.Type == "minio".
	OSAccessType      string `json:"os_access_type,omitempty"`
	OSEndpoint        string `json:"os_endpoint,omitempty"`
	OSAccessKeyId     string `json:"os_access_key_id,omitempty"`
	OSSecretAccessKey string `json:"os_secret_access_key,omitempty"`
	OSUseSSL          bool   `json:"os_use_ssl,omitempty"`
	OSScanBucket      string `json:"os_scan_bucket,omitempty"`
	OSScanPrefix      string `json:"os_scan_prefix,omitempty"`
}

type ListConnectionInfoRes struct {
	ConnectionInfo            []ConnectionInfo          `json:"connection_info"`
	ConnectionInfoStatusCount ConnectionInfoStatusCount `json:"connection_info_status_count"`
}

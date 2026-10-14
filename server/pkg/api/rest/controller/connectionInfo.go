package controller

import (
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	transxex "github.com/cloud-barista/cm-centipede/transx-ex"
	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/openbao"
	"github.com/cloud-barista/cm-honeybee/server/lib/rsautil"
	"github.com/cloud-barista/cm-honeybee/server/lib/ssh"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/google/uuid"
	"github.com/jollaman999/utils/iputil"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
	cryptossh "golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

// defaultProbeTimeout bounds the TCP reachability check of a direct db or
// minio connection when the connection sets no timeout of its own.
const defaultProbeTimeout = 5 * time.Second

// validateKubeconfig checks that s is a non-empty, structurally valid kubeconfig.
func validateKubeconfig(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("kubeconfig is empty")
	}
	var kc struct {
		Kind     string        `yaml:"kind"`
		Clusters []interface{} `yaml:"clusters"`
	}
	if err := yaml.Unmarshal([]byte(s), &kc); err != nil {
		return errors.New("kubeconfig is not valid YAML (" + err.Error() + ")")
	}
	if kc.Kind != "" && kc.Kind != "Config" {
		return errors.New("kubeconfig kind must be 'Config'")
	}
	if len(kc.Clusters) == 0 {
		return errors.New("kubeconfig has no clusters")
	}
	return nil
}

func checkIPAddress(ipAddress string) error {
	if ipAddress == "" {
		return errors.New("ip_address is empty")
	}

	if iputil.CheckValidIP(ipAddress) == nil {
		return errors.New("ip_address is invalid")
	}

	return nil
}

func checkPort(port string) error {
	portInt, err := strconv.Atoi(port)
	if err != nil || portInt < 1 || portInt > 65535 {
		return errors.New("port value is invalid")
	}

	return nil
}

// checkDBType rejects a db_type transx-ex cannot inspect, so a typo fails at
// registration instead of at the first import.
func checkDBType(dbType string) error {
	switch dbType {
	case transxex.DBMSTypeMySQL, transxex.DBMSTypeMariaDB, transxex.DBMSTypePostgreSQL, transxex.DBMSTypeMongoDB:
		return nil
	}
	return errors.New("db_type must be one of mysql | mariadb | postgresql | mongodb")
}

// checkAccessType rejects an access type other than direct or ssh-tunnel.
// field names the request field in the error.
func checkAccessType(field string, accessType string) error {
	switch accessType {
	case model.AccessTypeDirect, model.AccessTypeSSHTunnel:
		return nil
	}
	return errors.New(field + " must be one of direct | ssh-tunnel")
}

// checkTLSMode rejects an unknown db_tls_mode. transx-ex rejects one as well,
// but only when an import runs; this keeps a connection that can never connect
// from being stored. Empty means disable.
func checkTLSMode(mode string) error {
	switch mode {
	case "", transxex.TLSModeDisable, transxex.TLSModePrefer, transxex.TLSModeRequire,
		transxex.TLSModeVerifyCA, transxex.TLSModeVerifyFull:
		return nil
	}
	return errors.New("db_tls_mode must be one of disable | prefer | require | verify-ca | verify-full")
}

// checkPgSchema checks what honeybee can judge about a db_pg_schema selection.
// The rules on the names themselves (length, characters, pg_ system schemas)
// are transx-ex's and are not repeated here.
func checkPgSchema(dbType string, dbName string, schemas []string) error {
	if len(schemas) == 0 {
		return nil
	}
	// MySQL and MariaDB have no schema within a database, MongoDB none at all.
	if dbType != transxex.DBMSTypePostgreSQL {
		return errors.New("db_pg_schema is only supported for postgresql")
	}
	// Enumerating every database reuses one location for all of them, so a
	// schema selection would skip each database that lacks those schemas.
	if dbName == "" {
		return errors.New("db_pg_schema requires db_name")
	}
	seen := make(map[string]bool, len(schemas))
	for _, s := range schemas {
		if strings.TrimSpace(s) == "" {
			return errors.New("db_pg_schema has an empty name")
		}
		if seen[s] {
			return errors.New("db_pg_schema has a duplicate name: " + s)
		}
		seen[s] = true
	}
	return nil
}

// sameStringSet reports whether a and b hold the same strings, ignoring order
// and repeats.
func sameStringSet(a []string, b []string) bool {
	setA := make(map[string]bool, len(a))
	for _, s := range a {
		setA[s] = true
	}
	setB := make(map[string]bool, len(b))
	for _, s := range b {
		setB[s] = true
	}
	if len(setA) != len(setB) {
		return false
	}
	for s := range setA {
		if !setB[s] {
			return false
		}
	}
	return true
}

// checkFSScanPath requires an absolute path: the agent would resolve a relative
// one against its own working directory, which the caller cannot predict.
func checkFSScanPath(scanPath string) error {
	if !path.IsAbs(scanPath) {
		return errors.New("fs_scan_path must be an absolute path")
	}
	return nil
}

// checkOSScanBucket rejects a bucket name with a slash. The scan root is built
// as bucket + "/" + prefix, and a slash in the name makes a root that is
// neither, which the inspection reports as an empty result, not an error.
func checkOSScanBucket(bucket string) error {
	if strings.Contains(bucket, "/") {
		return errors.New("os_scan_bucket must not contain '/'")
	}
	return nil
}

// normalizeOSScanPrefix drops leading slashes: object keys do not start with
// one. It also makes "/" mean "no prefix" on update, where an empty value
// already means "no change".
func normalizeOSScanPrefix(prefix string) string {
	return strings.TrimLeft(prefix, "/")
}

// checkSSHKeyOnly checks the SSH credentials of fs, db and minio connections.
// cm-centipede reaches these hosts too when migrating, and it authenticates by
// private key only, so a password-only connection would collect here and fail
// there. The key is parsed so that one unusable for authentication is refused
// now rather than at the first SSH connection; "\n" escapes are expanded first,
// as lib/ssh does.
func checkSSHKeyOnly(password string, privateKey string) error {
	if password != "" {
		return errors.New("password is not allowed; use private_key")
	}
	if privateKey == "" || privateKey == "-" {
		return errors.New("private_key is empty")
	}
	if _, err := cryptossh.ParsePrivateKey([]byte(strings.ReplaceAll(privateKey, "\\n", "\n"))); err != nil {
		return errors.New("private_key is invalid (" + err.Error() + ")")
	}
	return nil
}

// invalidateSavedFSInfo marks the connection's file system result stale after
// its scan path changed. A connection never collected has no row; that is not
// an error.
func invalidateSavedFSInfo(connID string) error {
	saved, _ := dao.SavedFSInfoGet(connID)
	if saved == nil {
		return nil
	}
	saved.Status = model.SavedFSInfoStatusStale
	return dao.SavedFSInfoUpdate(saved)
}

// invalidateSavedObjectStorageInfo marks the connection's object storage result
// stale after its bucket or prefix changed.
func invalidateSavedObjectStorageInfo(connID string) error {
	saved, _ := dao.SavedObjectStorageInfoGet(connID)
	if saved == nil {
		return nil
	}
	saved.Status = model.SavedObjectStorageInfoStatusStale
	return dao.SavedObjectStorageInfoUpdate(saved)
}

// invalidateSavedDBInfo marks the connection's DBMS result stale after its
// db_type, db_name or db_pg_schema changed.
func invalidateSavedDBInfo(connID string) error {
	saved, _ := dao.SavedDBInfoGet(connID)
	if saved == nil {
		return nil
	}
	saved.Status = model.SavedDBInfoStatusStale
	return dao.SavedDBInfoUpdate(saved)
}

func encryptField(plaintext string, label string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	enc, err := rsautil.EncryptWithPublicKey([]byte(plaintext), serverCommon.PubKey)
	if err != nil {
		errMsg := "error occurred while encrypting the " + label + " (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)
		return "", errors.New(errMsg)
	}
	return base64.StdEncoding.EncodeToString(enc), nil
}

func encryptSecrets(connectionInfo *model.ConnectionInfo) (*model.ConnectionInfo, error) {
	user, err := encryptField(connectionInfo.User, "user")
	if err != nil {
		return nil, err
	}
	connectionInfo.User = user

	password, err := encryptField(connectionInfo.Password, "password")
	if err != nil {
		return nil, err
	}
	connectionInfo.Password = password

	privateKey, err := encryptField(connectionInfo.PrivateKey, "private key")
	if err != nil {
		return nil, err
	}
	connectionInfo.PrivateKey = privateKey

	kubeconfig, err := encryptField(connectionInfo.Kubeconfig, "kubeconfig")
	if err != nil {
		return nil, err
	}
	connectionInfo.Kubeconfig = kubeconfig

	dbUsername, err := encryptField(connectionInfo.DBUsername, "db username")
	if err != nil {
		return nil, err
	}
	connectionInfo.DBUsername = dbUsername

	dbPassword, err := encryptField(connectionInfo.DBPassword, "db password")
	if err != nil {
		return nil, err
	}
	connectionInfo.DBPassword = dbPassword

	osAccessKeyId, err := encryptField(connectionInfo.OSAccessKeyId, "object storage access key id")
	if err != nil {
		return nil, err
	}
	connectionInfo.OSAccessKeyId = osAccessKeyId

	osSecretAccessKey, err := encryptField(connectionInfo.OSSecretAccessKey, "object storage secret access key")
	if err != nil {
		return nil, err
	}
	connectionInfo.OSSecretAccessKey = osSecretAccessKey

	return connectionInfo, nil
}

// connectionSecretPath is the OpenBao KV path for a connection's secrets. The
// "ssh" segment is kept so entries written before kubeconfig support stay readable.
func connectionSecretPath(connID string) string { return "honeybee/ssh/" + connID }

// storeConnectionSecrets moves the connection secrets (SSH password/private key,
// kubeconfig, db password, object storage keys) into OpenBao when enabled,
// clearing them from ci so the DB row holds no secrets. Errors when there are
// secrets but OpenBao is off. Which secrets a connection has follows from its
// fields, not its source group type: checkCreateConnectionInfoReq fills only
// the fields of the group's type.
func storeConnectionSecrets(ci *model.ConnectionInfo) error {
	if ci.Password == "" && (ci.PrivateKey == "" || ci.PrivateKey == "-") && ci.Kubeconfig == "" &&
		ci.DBPassword == "" && ci.OSAccessKeyId == "" && ci.OSSecretAccessKey == "" {
		return nil // nothing sensitive to store (e.g. CSP connection without SSH)
	}
	if !openbao.Enabled() {
		return errors.New("OpenBao is required to store connection secrets (set cm-honeybee.openbao.address)")
	}
	data := map[string]string{
		"password":             ci.Password,
		"private_key":          ci.PrivateKey,
		"kubeconfig":           ci.Kubeconfig,
		"db_password":          ci.DBPassword,
		"os_access_key_id":     ci.OSAccessKeyId,
		"os_secret_access_key": ci.OSSecretAccessKey,
	}
	if err := openbao.Put(connectionSecretPath(ci.ID), data); err != nil {
		return err
	}
	ci.Password = ""
	ci.PrivateKey = ""
	ci.Kubeconfig = ""
	ci.DBPassword = ""
	ci.OSAccessKeyId = ""
	ci.OSSecretAccessKey = ""
	return nil
}

// hydrateConnectionSecrets loads the connection secrets from OpenBao into ci
// before they are used. Falls back to whatever is already on ci (DB values) when
// OpenBao is off or the connection predates OpenBao.
func hydrateConnectionSecrets(ci *model.ConnectionInfo) error {
	if !openbao.Enabled() {
		return nil
	}
	data, err := openbao.Get(connectionSecretPath(ci.ID))
	if err != nil {
		if errors.Is(err, openbao.ErrNotFound) {
			return nil
		}
		return err
	}
	ci.Password = data["password"]
	ci.PrivateKey = data["private_key"]
	if ci.PrivateKey == "" {
		ci.PrivateKey = "-"
	}
	ci.Kubeconfig = data["kubeconfig"]
	ci.DBPassword = data["db_password"]
	ci.OSAccessKeyId = data["os_access_key_id"]
	ci.OSSecretAccessKey = data["os_secret_access_key"]
	return nil
}

// deleteConnectionSecrets removes a connection's secrets from OpenBao.
func deleteConnectionSecrets(connID string) {
	if !openbao.Enabled() {
		return
	}
	if err := openbao.Delete(connectionSecretPath(connID)); err != nil {
		logger.Println(logger.WARN, true, "OpenBao: failed to delete connection secrets ("+connID+"): "+err.Error())
	}
}

func checkCreateConnectionInfoReq(sourceGroup *model.SourceGroup, createConnectionInfoReq *model.CreateConnectionInfoReq) (*model.ConnectionInfo, error) {
	if sourceGroup == nil || sourceGroup.ID == "" {
		return nil, errors.New("source group is missing")
	}

	connectionInfo := &model.ConnectionInfo{
		ID:            uuid.New().String(),
		Name:          createConnectionInfoReq.Name,
		Description:   createConnectionInfoReq.Description,
		SourceGroupID: sourceGroup.ID,
	}

	if connectionInfo.Name == "" {
		return nil, errors.New("name is empty")
	}

	switch sourceGroup.Type {
	case "", serverCommon.SourceGroupTypeSSH, serverCommon.SourceGroupTypeOnprem:
		connectionInfo.ResourceType = strings.ToLower(strings.TrimSpace(createConnectionInfoReq.ResourceType))
		switch connectionInfo.ResourceType {
		case "", serverCommon.ResourceTypeVM:
			connectionInfo.IPAddress = createConnectionInfoReq.IPAddress
			connectionInfo.SSHPort = createConnectionInfoReq.SSHPort
			connectionInfo.User = createConnectionInfoReq.User
			connectionInfo.Password = createConnectionInfoReq.Password
			connectionInfo.PrivateKey = createConnectionInfoReq.PrivateKey

			if err := checkIPAddress(connectionInfo.IPAddress); err != nil {
				return nil, err
			}
			if err := checkPort(connectionInfo.SSHPort); err != nil {
				return nil, err
			}
			if connectionInfo.User == "" {
				return nil, errors.New("user is empty")
			}
			if connectionInfo.Password == "" && connectionInfo.PrivateKey == "" {
				return nil, errors.New("password or private_key must be provided")
			}
			if connectionInfo.PrivateKey == "" {
				connectionInfo.PrivateKey = "-"
			}
		case serverCommon.ResourceTypeK8s:
			connectionInfo.Kubeconfig = createConnectionInfoReq.Kubeconfig
			if err := validateKubeconfig(connectionInfo.Kubeconfig); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("resource_type for on-prem must be one of vm | k8s")
		}
	case serverCommon.SourceGroupTypeCSP:
		connectionInfo.ResourceType = strings.ToLower(strings.TrimSpace(createConnectionInfoReq.ResourceType))
		connectionInfo.ResourceID = strings.TrimSpace(createConnectionInfoReq.ResourceID)
		connectionInfo.Zone = strings.TrimSpace(createConnectionInfoReq.Zone)
		if !serverCommon.IsValidCSPResourceType(connectionInfo.ResourceType) {
			return nil, errors.New(serverCommon.CSPResourceTypesMsg)
		}
		if connectionInfo.ResourceID == "" {
			return nil, errors.New("resource_id is empty")
		}

		// Optional SSH access. cb-spider gives us the CSP-level VM metadata, but
		// OS-level details (CPU/memory/kernel/software) need the in-guest agent.
		// When SSH credentials are provided, honeybee installs and queries that
		// agent just like an SSH source; without them, only CSP metadata is
		// available.
		connectionInfo.IPAddress = strings.TrimSpace(createConnectionInfoReq.IPAddress)
		connectionInfo.SSHPort = strings.TrimSpace(createConnectionInfoReq.SSHPort)
		connectionInfo.User = createConnectionInfoReq.User
		connectionInfo.Password = createConnectionInfoReq.Password
		connectionInfo.PrivateKey = createConnectionInfoReq.PrivateKey
		if connectionInfo.IPAddress != "" {
			if connectionInfo.SSHPort == "" {
				connectionInfo.SSHPort = "22"
			}
			if err := checkPort(connectionInfo.SSHPort); err != nil {
				return nil, err
			}
			if connectionInfo.User == "" {
				return nil, errors.New("user is empty (required when ip_address is provided)")
			}
			if connectionInfo.Password == "" && connectionInfo.PrivateKey == "" {
				return nil, errors.New("password or private_key must be provided when ip_address is provided")
			}
			if connectionInfo.PrivateKey == "" {
				connectionInfo.PrivateKey = "-"
			}
		}
	case serverCommon.SourceGroupTypeFS:
		// The host the agent walks, reached over SSH by key only (see
		// checkSSHKeyOnly). No resource_type: an fs group holds hosts only.
		connectionInfo.IPAddress = createConnectionInfoReq.IPAddress
		connectionInfo.SSHPort = createConnectionInfoReq.SSHPort
		connectionInfo.User = createConnectionInfoReq.User
		connectionInfo.PrivateKey = createConnectionInfoReq.PrivateKey

		if err := checkIPAddress(connectionInfo.IPAddress); err != nil {
			return nil, err
		}
		if err := checkPort(connectionInfo.SSHPort); err != nil {
			return nil, err
		}
		if connectionInfo.User == "" {
			return nil, errors.New("user is empty")
		}
		if err := checkSSHKeyOnly(createConnectionInfoReq.Password, connectionInfo.PrivateKey); err != nil {
			return nil, err
		}

		// The default is applied here so that a stored path is never empty and
		// the import has no default of its own.
		connectionInfo.FSScanPath = strings.TrimSpace(createConnectionInfoReq.FSScanPath)
		if connectionInfo.FSScanPath == "" {
			connectionInfo.FSScanPath = "/home"
		}
		if err := checkFSScanPath(connectionInfo.FSScanPath); err != nil {
			return nil, err
		}
	case serverCommon.SourceGroupTypeDB:
		connectionInfo.DBType = strings.ToLower(strings.TrimSpace(createConnectionInfoReq.DBType))
		connectionInfo.DBName = strings.TrimSpace(createConnectionInfoReq.DBName)
		connectionInfo.DBHost = strings.TrimSpace(createConnectionInfoReq.DBHost)
		connectionInfo.DBPort = strings.TrimSpace(createConnectionInfoReq.DBPort)
		connectionInfo.DBUsername = createConnectionInfoReq.DBUsername
		connectionInfo.DBPassword = createConnectionInfoReq.DBPassword
		connectionInfo.DBConnectTimeout = createConnectionInfoReq.DBConnectTimeout
		connectionInfo.DBTLSMode = strings.ToLower(strings.TrimSpace(createConnectionInfoReq.DBTLSMode))
		connectionInfo.DBTLSCAPEM = createConnectionInfoReq.DBTLSCAPEM
		connectionInfo.DBAuthSource = strings.TrimSpace(createConnectionInfoReq.DBAuthSource)
		connectionInfo.DBPgSchema = createConnectionInfoReq.DBPgSchema

		if err := checkDBType(connectionInfo.DBType); err != nil {
			return nil, err
		}
		if connectionInfo.DBHost == "" {
			return nil, errors.New("db_host is empty")
		}
		if err := checkPort(connectionInfo.DBPort); err != nil {
			return nil, errors.New("db_port value is invalid")
		}
		if connectionInfo.DBUsername == "" {
			return nil, errors.New("db_username is empty")
		}
		if connectionInfo.DBPassword == "" {
			return nil, errors.New("db_password is empty")
		}
		if connectionInfo.DBConnectTimeout < 0 {
			return nil, errors.New("db_connect_timeout must not be negative")
		}
		if err := checkTLSMode(connectionInfo.DBTLSMode); err != nil {
			return nil, err
		}
		if err := checkPgSchema(connectionInfo.DBType, connectionInfo.DBName, connectionInfo.DBPgSchema); err != nil {
			return nil, err
		}

		connectionInfo.DBAccessType = strings.ToLower(strings.TrimSpace(createConnectionInfoReq.DBAccessType))
		if connectionInfo.DBAccessType == "" {
			connectionInfo.DBAccessType = model.AccessTypeDirect
		}
		if err := checkAccessType("db_access_type", connectionInfo.DBAccessType); err != nil {
			return nil, err
		}
		// direct takes no SSH fields, even when the request carries them. No "-"
		// default for private_key either way: ssh-tunnel requires a key, and on
		// direct the "-" would be encrypted into every response.
		if connectionInfo.DBAccessType == model.AccessTypeSSHTunnel {
			connectionInfo.IPAddress = strings.TrimSpace(createConnectionInfoReq.IPAddress)
			connectionInfo.SSHPort = strings.TrimSpace(createConnectionInfoReq.SSHPort)
			connectionInfo.User = createConnectionInfoReq.User
			connectionInfo.PrivateKey = createConnectionInfoReq.PrivateKey

			if err := checkIPAddress(connectionInfo.IPAddress); err != nil {
				return nil, errors.New(err.Error() + " (required for a " + model.AccessTypeSSHTunnel + " connection)")
			}
			if connectionInfo.SSHPort == "" {
				connectionInfo.SSHPort = "22"
			}
			if err := checkPort(connectionInfo.SSHPort); err != nil {
				return nil, err
			}
			if connectionInfo.User == "" {
				return nil, errors.New("user is empty (required for a " + model.AccessTypeSSHTunnel + " connection)")
			}
			if err := checkSSHKeyOnly(createConnectionInfoReq.Password, connectionInfo.PrivateKey); err != nil {
				return nil, errors.New(err.Error() + " (" + model.AccessTypeSSHTunnel + " connection)")
			}
		}
	case serverCommon.SourceGroupTypeMinIO:
		connectionInfo.OSEndpoint = strings.TrimSpace(createConnectionInfoReq.OSEndpoint)
		connectionInfo.OSAccessKeyId = strings.TrimSpace(createConnectionInfoReq.OSAccessKeyId)
		connectionInfo.OSSecretAccessKey = createConnectionInfoReq.OSSecretAccessKey
		connectionInfo.OSUseSSL = createConnectionInfoReq.OSUseSSL
		connectionInfo.OSScanBucket = strings.TrimSpace(createConnectionInfoReq.OSScanBucket)
		connectionInfo.OSScanPrefix = normalizeOSScanPrefix(strings.TrimSpace(createConnectionInfoReq.OSScanPrefix))

		if connectionInfo.OSAccessKeyId == "" {
			return nil, errors.New("os_access_key_id is empty")
		}
		if connectionInfo.OSSecretAccessKey == "" {
			return nil, errors.New("os_secret_access_key is empty")
		}
		if connectionInfo.OSScanBucket == "" {
			return nil, errors.New("os_scan_bucket is empty")
		}
		if err := checkOSScanBucket(connectionInfo.OSScanBucket); err != nil {
			return nil, err
		}
		// The os_endpoint rules per provider live in resolveS3Endpoint only.
		if _, _, _, _, err := resolveS3Endpoint(sourceGroup, connectionInfo); err != nil {
			return nil, err
		}

		connectionInfo.OSAccessType = strings.ToLower(strings.TrimSpace(createConnectionInfoReq.OSAccessType))
		if connectionInfo.OSAccessType == "" {
			connectionInfo.OSAccessType = model.AccessTypeDirect
		}
		if err := checkAccessType("os_access_type", connectionInfo.OSAccessType); err != nil {
			return nil, err
		}
		// Same as db: SSH fields only for ssh-tunnel, and no "-" default.
		if connectionInfo.OSAccessType == model.AccessTypeSSHTunnel {
			connectionInfo.IPAddress = strings.TrimSpace(createConnectionInfoReq.IPAddress)
			connectionInfo.SSHPort = strings.TrimSpace(createConnectionInfoReq.SSHPort)
			connectionInfo.User = createConnectionInfoReq.User
			connectionInfo.PrivateKey = createConnectionInfoReq.PrivateKey

			if err := checkIPAddress(connectionInfo.IPAddress); err != nil {
				return nil, errors.New(err.Error() + " (required for a " + model.AccessTypeSSHTunnel + " connection)")
			}
			if connectionInfo.SSHPort == "" {
				connectionInfo.SSHPort = "22"
			}
			if err := checkPort(connectionInfo.SSHPort); err != nil {
				return nil, err
			}
			if connectionInfo.User == "" {
				return nil, errors.New("user is empty (required for a " + model.AccessTypeSSHTunnel + " connection)")
			}
			if err := checkSSHKeyOnly(createConnectionInfoReq.Password, connectionInfo.PrivateKey); err != nil {
				return nil, errors.New(err.Error() + " (" + model.AccessTypeSSHTunnel + " connection)")
			}
		}
	default:
		return nil, errors.New("unsupported source group type: " + sourceGroup.Type)
	}

	return connectionInfo, nil
}

func doGetConnectionInfo(connID string, refresh bool) (*model.ConnectionInfo, error) {
	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, err
	}

	oldConnectionInfo, err := dao.ConnectionInfoGet(connectionInfo.ID)
	if err != nil {
		return nil, err
	}

	if refresh {
		// Load SSH secrets from OpenBao (when enabled) before any SSH operation.
		if err := hydrateConnectionSecrets(connectionInfo); err != nil {
			return nil, err
		}

		sourceGroup, err := dao.SourceGroupGet(connectionInfo.SourceGroupID)
		if err != nil {
			return nil, err
		}

		switch sourceGroup.Type {
		case serverCommon.SourceGroupTypeCSP:
			// connection_status reflects CSP reachability: whether the CSP driver can
			// identify this resource. This is a status-only check - CSP data is
			// collected/persisted by import/infra, not here (refresh/registration).
			if err := checkCSPConnection(sourceGroup, connectionInfo); err != nil {
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.ConnectionFailedMessage = err.Error()
			} else {
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusSuccess
				oldConnectionInfo.ConnectionFailedMessage = ""
			}

			// agent_status reflects the REAL in-guest agent: it requires SSH
			// access, so it is only attempted when credentials were provided.
			// Previously this was faked to "success" from the cb-spider result
			// even though no agent was installed.
			if connectionInfo.IPAddress == "" {
				oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.AgentFailedMessage = "no SSH access configured for this CSP connection; " +
					"provide ip_address/user (and password or private_key) to install the agent"
			} else {
				c := &ssh.SSH{}
				if err := c.RunAgent(*connectionInfo); err != nil {
					oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusFailed
					oldConnectionInfo.AgentFailedMessage = err.Error()
				} else {
					c.Close()
					oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusSuccess
					oldConnectionInfo.AgentFailedMessage = ""
				}
			}
		case serverCommon.SourceGroupTypeDB:
			// connection_status is TCP reachability of the DBMS from where it is
			// inspected: this server for direct, the SSH host for ssh-tunnel, from
			// whose side db_host is resolved. Whether the credentials work is
			// decided by an import, not here.
			timeout := defaultProbeTimeout
			if connectionInfo.DBConnectTimeout > 0 {
				timeout = time.Duration(connectionInfo.DBConnectTimeout) * time.Second
			}
			addr := net.JoinHostPort(connectionInfo.DBHost, connectionInfo.DBPort)
			var probeErr error
			if connectionInfo.DBAccessType == model.AccessTypeSSHTunnel {
				// telnet:// is curl's raw-TCP scheme: no DB protocol is spoken,
				// which keeps this a reachability check like the dial below.
				c := &ssh.SSH{}
				probeErr = c.ProbeFromHost(*connectionInfo, "telnet://"+addr, timeout)
			} else {
				conn, dialErr := net.DialTimeout("tcp", addr, timeout)
				if dialErr == nil {
					_ = conn.Close()
				}
				probeErr = dialErr
			}
			if probeErr != nil {
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.ConnectionFailedMessage = probeErr.Error()
			} else {
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusSuccess
				oldConnectionInfo.ConnectionFailedMessage = ""
			}

			// ssh-tunnel collects through the agent on the SSH host. direct is
			// inspected by this server and installs no agent, marked the way
			// on-prem k8s is.
			if connectionInfo.DBAccessType == model.AccessTypeSSHTunnel {
				c := &ssh.SSH{}
				if err := c.RunAgent(*connectionInfo); err != nil {
					oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusFailed
					oldConnectionInfo.AgentFailedMessage = err.Error()
				} else {
					c.Close()
					oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusSuccess
					oldConnectionInfo.AgentFailedMessage = ""
				}
			} else {
				oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.AgentFailedMessage = "agent-based collection is not applicable to direct db connections"
			}
		case serverCommon.SourceGroupTypeMinIO:
			// Both access types need the endpoint, so it is resolved first. A
			// connection that cannot be resolved can be neither probed nor
			// collected, and both statuses say why.
			endpoint, _, useSSL, _, resolveErr := resolveS3Endpoint(sourceGroup, connectionInfo)
			if resolveErr != nil {
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.ConnectionFailedMessage = resolveErr.Error()
				oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.AgentFailedMessage = resolveErr.Error()
				break
			}
			// resolveS3Endpoint already returns host[:port]; a missing port is the
			// scheme's default.
			if _, _, splitErr := net.SplitHostPort(endpoint); splitErr != nil {
				if useSSL {
					endpoint = net.JoinHostPort(endpoint, "443")
				} else {
					endpoint = net.JoinHostPort(endpoint, "80")
				}
			}
			// A tunnelled endpoint is resolved from the SSH host's side, where the
			// agent inspects it, so that is where it is probed.
			var probeErr error
			if connectionInfo.OSAccessType == model.AccessTypeSSHTunnel {
				scheme := "http"
				if useSSL {
					scheme = "https"
				}
				c := &ssh.SSH{}
				probeErr = c.ProbeFromHost(*connectionInfo, scheme+"://"+endpoint+"/", defaultProbeTimeout)
			} else {
				conn, dialErr := net.DialTimeout("tcp", endpoint, defaultProbeTimeout)
				if dialErr == nil {
					_ = conn.Close()
				}
				probeErr = dialErr
			}
			if probeErr != nil {
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.ConnectionFailedMessage = probeErr.Error()
			} else {
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusSuccess
				oldConnectionInfo.ConnectionFailedMessage = ""
			}

			if connectionInfo.OSAccessType == model.AccessTypeSSHTunnel {
				c := &ssh.SSH{}
				if err := c.RunAgent(*connectionInfo); err != nil {
					oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusFailed
					oldConnectionInfo.AgentFailedMessage = err.Error()
				} else {
					c.Close()
					oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusSuccess
					oldConnectionInfo.AgentFailedMessage = ""
				}
			} else {
				oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.AgentFailedMessage = "agent-based collection is not applicable to direct object storage connections"
			}
		default:
			if connectionInfo.ResourceType == serverCommon.ResourceTypeK8s {
				// on-prem k8s has no SSH host. The kubeconfig (validated at
				// register) is consumed by downstream migration; agent-based
				// collection over SSH does not apply.
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusSuccess
				oldConnectionInfo.ConnectionFailedMessage = ""
				oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.AgentFailedMessage = "agent-based collection is not applicable to on-prem k8s connections"
				break
			}

			c := &ssh.SSH{}

			if err := c.NewClientConn(*connectionInfo); err != nil {
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.ConnectionFailedMessage = err.Error()
			} else {
				c.Close()
				oldConnectionInfo.ConnectionStatus = model.ConnectionInfoStatusSuccess
				oldConnectionInfo.ConnectionFailedMessage = ""
			}

			if err := c.RunAgent(*connectionInfo); err != nil {
				oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusFailed
				oldConnectionInfo.AgentFailedMessage = err.Error()
			} else {
				c.Close()
				oldConnectionInfo.AgentStatus = model.ConnectionInfoStatusSuccess
				oldConnectionInfo.AgentFailedMessage = ""
			}
		}

		err = dao.ConnectionInfoUpdateWithSelect(oldConnectionInfo, []string{
			"connection_status",
			"connection_failed_message",
			"agent_status",
			"agent_failed_message",
		})
		if err != nil {
			return nil, errors.New("Error occurred while updating the connection information. " +
				"(ID: " + oldConnectionInfo.ID + ", Error: " + err.Error() + ")")
		}
	}

	// Restore SSH secrets (password, private key) from OpenBao before encrypting
	// the response. storeConnectionSecrets moves them out of the DB row into
	// OpenBao, so the row we just read has them empty. Consumers such as
	// cm-grasshopper read these fields (encrypted here, decrypted on their side)
	// to SSH into the source host; without this they receive empty credentials
	// and fail with "failed to determine auth method". The DB is untouched - the
	// refresh status update above uses a field allowlist.
	if err := hydrateConnectionSecrets(oldConnectionInfo); err != nil {
		return nil, err
	}

	connectionInfo, err = encryptSecrets(oldConnectionInfo)
	if err != nil {
		return nil, err
	}

	return connectionInfo, nil
}

func doCreateConnectionInfo(connectionInfo *model.ConnectionInfo) (*model.ConnectionInfo, error) {
	_, err := dao.SourceGroupGet(connectionInfo.SourceGroupID)
	if err != nil {
		return nil, err
	}

	// Move SSH secrets into OpenBao (when enabled) before persisting the row.
	if err := storeConnectionSecrets(connectionInfo); err != nil {
		return nil, err
	}

	connectionInfo, err = dao.ConnectionInfoRegister(connectionInfo)
	if err != nil {
		return nil, err
	}

	connectionInfo, err = doGetConnectionInfo(connectionInfo.ID, true)
	if err != nil {
		return nil, err
	}

	return connectionInfo, nil
}

// CreateConnectionInfo godoc
//
//	@ID				create-connection-info
//	@Summary		Create ConnectionInfo
//	@Description	Create the connection information.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			ConnectionInfo body model.CreateConnectionInfoReq true "Connection information of the node."
//	@Success		200	{object}	model.ConnectionInfo	"Successfully register the connection information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to register the connection information"
//	@Router			/source_group/{sgId}/connection_info [post]
func CreateConnectionInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	createConnectionInfoReq := new(model.CreateConnectionInfoReq)
	err = c.Bind(createConnectionInfoReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfo, err := checkCreateConnectionInfoReq(sourceGroup, createConnectionInfoReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	listOption := &model.ConnectionInfo{
		SourceGroupID: sourceGroup.ID,
	}
	connectionInfos, err := dao.ConnectionInfoGetList(listOption, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if len(*connectionInfos) >= model.ConnectionInfoMaxLength {
		return common.ReturnErrorMsg(c, "Maximum number of connection info is exceeded."+
			" (Max: "+strconv.Itoa(model.ConnectionInfoMaxLength)+")")
	}

	connectionInfo, err = doCreateConnectionInfo(connectionInfo)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, connectionInfo, " ")
}

// GetConnectionInfo godoc
//
//	@ID				get-connection-info
//	@Summary		Get ConnectionInfo
//	@Description	Get the connection information.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			connId path string true "ID of the connectionInfo"
//	@Success		200	{object}	model.ConnectionInfo	"Successfully get the connection information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get the connection information"
//	@Router			/source_group/{sgId}/connection_info/{connId} [get]
func GetConnectionInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	connectionInfo, err := doGetConnectionInfo(connID, false)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, connectionInfo, " ")
}

// GetConnectionInfoDirectly godoc
//
//	@ID				get-connection-info-directly
//	@Summary		Get ConnectionInfo Directly
//	@Description	Get the connection information directly.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			connId path string true "ID of the connectionInfo"
//	@Success		200	{object}	model.ConnectionInfo	"Successfully get the connection information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get the connection information"
//	@Router			/connection_info/{connId} [get]
func GetConnectionInfoDirectly(c echo.Context) error {
	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	connectionInfo, err := doGetConnectionInfo(connID, false)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, connectionInfo, " ")
}

// ListConnectionInfo godoc
//
//	@ID				list-connection-info
//	@Summary		List ConnectionInfo
//	@Description	Get a list of connection information.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			page query string false "Page of the connection information list."
//	@Param			row query string false "Row of the connection information list."
//	@Param			name query string false "Name of the connection information."
//	@Param			description query string false "Description of the connection information."
//	@Param			ip_address query string false "IP address of the connection information."
//	@Param			ssh_port query string false "SSH port of the connection information."
//	@Param			user query string false "User of the connection information."
//	@Success		200	{object}	[]model.ListConnectionInfoRes	"Successfully get a list of connection information."
//	@Failure		400	{object}	common.ErrorResponse			"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse			"Failed to get a list of connection information."
//	@Router			/source_group/{sgId}/connection_info [get]
func ListConnectionInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	page, row, err := common.CheckPageRow(c)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfo := &model.ConnectionInfo{
		Name:          c.QueryParam("name"),
		Description:   c.QueryParam("description"),
		SourceGroupID: sourceGroup.ID,
		IPAddress:     c.QueryParam("ip_address"),
		SSHPort:       c.QueryParam("ssh_port"),
		User:          c.QueryParam("user"),
	}

	connectionInfos, err := dao.ConnectionInfoGetList(connectionInfo, page, row)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var listConnectionInfoRes model.ListConnectionInfoRes
	var encryptedConnectionInfos []model.ConnectionInfo

	for _, ci := range *connectionInfos {
		listConnectionInfoRes.ConnectionInfoStatusCount.ConnectionInfoTotal++
		if ci.ConnectionStatus == model.ConnectionInfoStatusSuccess {
			listConnectionInfoRes.ConnectionInfoStatusCount.CountConnectionSuccess++
		} else {
			listConnectionInfoRes.ConnectionInfoStatusCount.CountConnectionFailed++
		}
		if ci.AgentStatus == model.ConnectionInfoStatusSuccess {
			listConnectionInfoRes.ConnectionInfoStatusCount.CountAgentSuccess++
		} else {
			listConnectionInfoRes.ConnectionInfoStatusCount.CountAgentFailed++
		}

		// Restore SSH secrets from OpenBao so the list carries them encrypted,
		// same as the single-connection GET (see doGetConnectionInfo).
		if err := hydrateConnectionSecrets(&ci); err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}

		encryptedConnectionInfo, err := encryptSecrets(&ci)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}

		encryptedConnectionInfos = append(encryptedConnectionInfos, *encryptedConnectionInfo)
	}

	sort.Slice(encryptedConnectionInfos, func(i, j int) bool {
		return strings.Compare(encryptedConnectionInfos[i].Name, encryptedConnectionInfos[j].Name) < 0
	})

	listConnectionInfoRes.ConnectionInfo = encryptedConnectionInfos

	return c.JSONPretty(http.StatusOK, &listConnectionInfoRes, " ")
}

// UpdateConnectionInfo godoc
//
//	@ID				update-connection-info
//	@Summary		Update ConnectionInfo
//	@Description	Update the connection information.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			connId path string true "ID of the connectionInfo"
//	@Param			ConnectionInfo body model.CreateConnectionInfoReq true "Connection information to modify."
//	@Success		200	{object}	model.ConnectionInfo	"Successfully update the connection information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to update the connection information"
//	@Router			/source_group/{sgId}/connection_info/{connId} [put]
func UpdateConnectionInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	oldConnectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	// Load existing SSH secrets (OpenBao) so fields not present in the update
	// request keep their current values when re-stored below.
	if err := hydrateConnectionSecrets(oldConnectionInfo); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	updateConnectionInfoReq := new(model.CreateConnectionInfoReq)
	err = c.Bind(updateConnectionInfoReq)
	if err != nil {
		return err
	}

	if updateConnectionInfoReq.Name != "" {
		oldConnectionInfo.Name = updateConnectionInfoReq.Name
	}
	if updateConnectionInfoReq.Description != "" {
		oldConnectionInfo.Description = updateConnectionInfoReq.Description
	}

	// ConnectionInfoUpdate skips zero values; the arms list the columns that may
	// be cleared on purpose, written by name after the save.
	var explicitColumns []string

	switch sourceGroup.Type {
	case serverCommon.SourceGroupTypeCSP:
		if updateConnectionInfoReq.ResourceType != "" {
			rt := strings.ToLower(strings.TrimSpace(updateConnectionInfoReq.ResourceType))
			if !serverCommon.IsValidCSPResourceType(rt) {
				return common.ReturnErrorMsg(c, serverCommon.CSPResourceTypesMsg)
			}
			oldConnectionInfo.ResourceType = rt
		}
		if updateConnectionInfoReq.ResourceID != "" {
			oldConnectionInfo.ResourceID = strings.TrimSpace(updateConnectionInfoReq.ResourceID)
		}
	case serverCommon.SourceGroupTypeFS:
		// scanTargetChanged records that the update changes what the connection
		// collects (not where the source is), so the saved result no longer matches.
		scanTargetChanged := false

		// password is not taken: fs authenticates by key only (checked below).
		err = checkIPAddress(updateConnectionInfoReq.IPAddress)
		if err == nil {
			oldConnectionInfo.IPAddress = updateConnectionInfoReq.IPAddress
		}
		err = checkPort(updateConnectionInfoReq.SSHPort)
		if err == nil {
			oldConnectionInfo.SSHPort = updateConnectionInfoReq.SSHPort
		}
		if updateConnectionInfoReq.User != "" {
			oldConnectionInfo.User = updateConnectionInfoReq.User
		}
		if updateConnectionInfoReq.PrivateKey != "" {
			oldConnectionInfo.PrivateKey = updateConnectionInfoReq.PrivateKey
		}
		if scanPath := strings.TrimSpace(updateConnectionInfoReq.FSScanPath); scanPath != "" {
			if err := checkFSScanPath(scanPath); err != nil {
				return common.ReturnErrorMsg(c, err.Error())
			}
			if scanPath != oldConnectionInfo.FSScanPath {
				scanTargetChanged = true
			}
			oldConnectionInfo.FSScanPath = scanPath
		}

		// fs always reaches its host over SSH, by key only. Checked on the merged
		// values so that keeping the stored key is covered too, and before the
		// invalidation so that a refused request marks nothing. A password stored
		// before the rule is dropped here, and from OpenBao with the store below.
		if err := checkSSHKeyOnly(updateConnectionInfoReq.Password, oldConnectionInfo.PrivateKey); err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		oldConnectionInfo.Password = ""

		// Mark the saved result stale before the connection is saved: should the
		// save then fail, the worst left behind is an unchanged target marked
		// stale, which the next import clears, never a new target next to a success.
		if scanTargetChanged {
			if err := invalidateSavedFSInfo(oldConnectionInfo.ID); err != nil {
				return common.ReturnErrorMsg(c, "failed to mark the saved result stale: "+err.Error())
			}
		}
	case serverCommon.SourceGroupTypeDB:
		// Same as fs: what the connection collects, not where the server is.
		scanTargetChanged := false

		if dbType := strings.ToLower(strings.TrimSpace(updateConnectionInfoReq.DBType)); dbType != "" {
			if err := checkDBType(dbType); err != nil {
				return common.ReturnErrorMsg(c, err.Error())
			}
			if dbType != oldConnectionInfo.DBType {
				scanTargetChanged = true
			}
			oldConnectionInfo.DBType = dbType
		}
		if dbName := strings.TrimSpace(updateConnectionInfoReq.DBName); dbName != "" {
			if dbName != oldConnectionInfo.DBName {
				scanTargetChanged = true
			}
			oldConnectionInfo.DBName = dbName
		}
		if accessType := strings.ToLower(strings.TrimSpace(updateConnectionInfoReq.DBAccessType)); accessType != "" {
			if err := checkAccessType("db_access_type", accessType); err != nil {
				return common.ReturnErrorMsg(c, err.Error())
			}
			oldConnectionInfo.DBAccessType = accessType
		}
		if dbHost := strings.TrimSpace(updateConnectionInfoReq.DBHost); dbHost != "" {
			oldConnectionInfo.DBHost = dbHost
		}
		if dbPort := strings.TrimSpace(updateConnectionInfoReq.DBPort); dbPort != "" {
			if err := checkPort(dbPort); err != nil {
				return common.ReturnErrorMsg(c, "db_port value is invalid")
			}
			oldConnectionInfo.DBPort = dbPort
		}
		if updateConnectionInfoReq.DBUsername != "" {
			oldConnectionInfo.DBUsername = updateConnectionInfoReq.DBUsername
		}
		if updateConnectionInfoReq.DBPassword != "" {
			oldConnectionInfo.DBPassword = updateConnectionInfoReq.DBPassword
		}
		if updateConnectionInfoReq.DBConnectTimeout < 0 {
			return common.ReturnErrorMsg(c, "db_connect_timeout must not be negative")
		}
		if updateConnectionInfoReq.DBConnectTimeout > 0 {
			oldConnectionInfo.DBConnectTimeout = updateConnectionInfoReq.DBConnectTimeout
		}
		if updateConnectionInfoReq.DBAuthSource != "" {
			oldConnectionInfo.DBAuthSource = strings.TrimSpace(updateConnectionInfoReq.DBAuthSource)
		}
		// Taken as sent, unlike the fields above: an empty mode is the real
		// value "disable", not "no change", or a connection once set to require
		// could never be turned back. The CA follows the mode.
		tlsMode := strings.ToLower(strings.TrimSpace(updateConnectionInfoReq.DBTLSMode))
		if err := checkTLSMode(tlsMode); err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		oldConnectionInfo.DBTLSMode = tlsMode
		oldConnectionInfo.DBTLSCAPEM = updateConnectionInfoReq.DBTLSCAPEM
		// Also as sent: a list cannot say "no change" with an empty value, so
		// omitting it means every user schema again.
		if !sameStringSet(updateConnectionInfoReq.DBPgSchema, oldConnectionInfo.DBPgSchema) {
			scanTargetChanged = true
		}
		oldConnectionInfo.DBPgSchema = updateConnectionInfoReq.DBPgSchema
		// db_type, db_name and db_pg_schema may change together, so the check
		// runs on the merged values.
		if err := checkPgSchema(oldConnectionInfo.DBType, oldConnectionInfo.DBName, oldConnectionInfo.DBPgSchema); err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		if oldConnectionInfo.DBAccessType == model.AccessTypeSSHTunnel {
			err = checkIPAddress(updateConnectionInfoReq.IPAddress)
			if err == nil {
				oldConnectionInfo.IPAddress = updateConnectionInfoReq.IPAddress
			}
			err = checkPort(updateConnectionInfoReq.SSHPort)
			if err == nil {
				oldConnectionInfo.SSHPort = updateConnectionInfoReq.SSHPort
			}
			if updateConnectionInfoReq.User != "" {
				oldConnectionInfo.User = updateConnectionInfoReq.User
			}
			if updateConnectionInfoReq.PrivateKey != "" {
				oldConnectionInfo.PrivateKey = updateConnectionInfoReq.PrivateKey
			}

			// Key-only SSH, as for fs. Placed after the merge so that switching
			// db_access_type to ssh-tunnel is checked as well.
			if err := checkSSHKeyOnly(updateConnectionInfoReq.Password, oldConnectionInfo.PrivateKey); err != nil {
				return common.ReturnErrorMsg(c, err.Error())
			}
			oldConnectionInfo.Password = ""
		}

		// Before the save, for the reason given in the fs arm.
		if scanTargetChanged {
			if err := invalidateSavedDBInfo(oldConnectionInfo.ID); err != nil {
				return common.ReturnErrorMsg(c, "failed to mark the saved result stale: "+err.Error())
			}
		}

		explicitColumns = []string{"db_tls_mode", "db_tls_ca_pem", "db_pg_schema"}
	case serverCommon.SourceGroupTypeMinIO:
		// Same as fs: which bucket and prefix are collected, not where the
		// endpoint is.
		scanTargetChanged := false

		if accessType := strings.ToLower(strings.TrimSpace(updateConnectionInfoReq.OSAccessType)); accessType != "" {
			if err := checkAccessType("os_access_type", accessType); err != nil {
				return common.ReturnErrorMsg(c, err.Error())
			}
			oldConnectionInfo.OSAccessType = accessType
		}
		if endpoint := strings.TrimSpace(updateConnectionInfoReq.OSEndpoint); endpoint != "" {
			oldConnectionInfo.OSEndpoint = endpoint
		}
		if accessKeyId := strings.TrimSpace(updateConnectionInfoReq.OSAccessKeyId); accessKeyId != "" {
			oldConnectionInfo.OSAccessKeyId = accessKeyId
		}
		if updateConnectionInfoReq.OSSecretAccessKey != "" {
			oldConnectionInfo.OSSecretAccessKey = updateConnectionInfoReq.OSSecretAccessKey
		}
		// A bool cannot say "no change": omitting it turns SSL off.
		oldConnectionInfo.OSUseSSL = updateConnectionInfoReq.OSUseSSL
		if bucket := strings.TrimSpace(updateConnectionInfoReq.OSScanBucket); bucket != "" {
			if err := checkOSScanBucket(bucket); err != nil {
				return common.ReturnErrorMsg(c, err.Error())
			}
			if bucket != oldConnectionInfo.OSScanBucket {
				scanTargetChanged = true
			}
			oldConnectionInfo.OSScanBucket = bucket
		}
		// Empty means "no change", so clearing the prefix is sent as "/", which
		// normalizes to "" (the whole bucket).
		if prefix := strings.TrimSpace(updateConnectionInfoReq.OSScanPrefix); prefix != "" {
			prefix = normalizeOSScanPrefix(prefix)
			if prefix != oldConnectionInfo.OSScanPrefix {
				scanTargetChanged = true
			}
			oldConnectionInfo.OSScanPrefix = prefix
		}
		if _, _, _, _, err := resolveS3Endpoint(sourceGroup, oldConnectionInfo); err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		if oldConnectionInfo.OSAccessType == model.AccessTypeSSHTunnel {
			err = checkIPAddress(updateConnectionInfoReq.IPAddress)
			if err == nil {
				oldConnectionInfo.IPAddress = updateConnectionInfoReq.IPAddress
			}
			err = checkPort(updateConnectionInfoReq.SSHPort)
			if err == nil {
				oldConnectionInfo.SSHPort = updateConnectionInfoReq.SSHPort
			}
			if updateConnectionInfoReq.User != "" {
				oldConnectionInfo.User = updateConnectionInfoReq.User
			}
			if updateConnectionInfoReq.PrivateKey != "" {
				oldConnectionInfo.PrivateKey = updateConnectionInfoReq.PrivateKey
			}

			// Key-only SSH, as for db.
			if err := checkSSHKeyOnly(updateConnectionInfoReq.Password, oldConnectionInfo.PrivateKey); err != nil {
				return common.ReturnErrorMsg(c, err.Error())
			}
			oldConnectionInfo.Password = ""
		}

		// Before the save, for the reason given in the fs arm.
		if scanTargetChanged {
			if err := invalidateSavedObjectStorageInfo(oldConnectionInfo.ID); err != nil {
				return common.ReturnErrorMsg(c, "failed to mark the saved result stale: "+err.Error())
			}
		}

		explicitColumns = []string{"os_use_ssl", "os_scan_prefix"}
	default:
		if oldConnectionInfo.ResourceType == serverCommon.ResourceTypeK8s {
			if updateConnectionInfoReq.Kubeconfig != "" {
				if err := validateKubeconfig(updateConnectionInfoReq.Kubeconfig); err != nil {
					return common.ReturnErrorMsg(c, err.Error())
				}
				oldConnectionInfo.Kubeconfig = updateConnectionInfoReq.Kubeconfig
			}
			break
		}
		err = checkIPAddress(updateConnectionInfoReq.IPAddress)
		if err == nil {
			oldConnectionInfo.IPAddress = updateConnectionInfoReq.IPAddress
		}
		err = checkPort(updateConnectionInfoReq.SSHPort)
		if err == nil {
			oldConnectionInfo.SSHPort = updateConnectionInfoReq.SSHPort
		}
		if updateConnectionInfoReq.User != "" {
			oldConnectionInfo.User = updateConnectionInfoReq.User
		}
		if updateConnectionInfoReq.Password != "" {
			oldConnectionInfo.Password = updateConnectionInfoReq.Password
		}
		if updateConnectionInfoReq.PrivateKey != "" {
			oldConnectionInfo.PrivateKey = updateConnectionInfoReq.PrivateKey
		}
	}

	// Persist SSH secrets to OpenBao (when enabled), clearing them from the row.
	if err := storeConnectionSecrets(oldConnectionInfo); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	err = dao.ConnectionInfoUpdate(oldConnectionInfo)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	// After the save, never before: written ahead of it, a failing save would
	// leave these columns changed and the rest of the row not.
	if len(explicitColumns) > 0 {
		if err := dao.ConnectionInfoUpdateWithSelect(oldConnectionInfo, explicitColumns); err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
	}

	connectionInfo, err := doGetConnectionInfo(oldConnectionInfo.ID, true)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, connectionInfo, " ")
}

// DeleteConnectionInfo godoc
//
//	@ID				delete-connection-info
//	@Summary		Delete ConnectionInfo
//	@Description	Delete the connection information.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			connId path string true "ID of the connectionInfo"
//	@Success		200	{object}	model.SimpleMsg			"Successfully delete the connection information"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to delete the connection information"
//	@Router			/source_group/{sgId}/connection_info/{connId} [delete]
func DeleteConnectionInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	deleteConnectionSecrets(connectionInfo.ID)

	err = dao.ConnectionInfoDelete(connectionInfo)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "success"}, " ")
}

// countConnectionsOnHost returns how many connections other than excludeID
// target the same host (ip_address). The agent is a host-wide systemd service,
// so this is used to avoid removing an agent still shared by another connection.
func countConnectionsOnHost(ipAddress, excludeID string) (int, error) {
	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{}, 0, 0)
	if err != nil {
		return 0, err
	}
	ip := strings.TrimSpace(ipAddress)
	count := 0
	for _, conn := range *list {
		if conn.ID != excludeID && strings.TrimSpace(conn.IPAddress) == ip {
			count++
		}
	}
	return count, nil
}

// UninstallAgent godoc
//
//	@ID				uninstall-agent
//	@Summary		Uninstall Agent
//	@Description	Uninstall cm-honeybee-agent from the connection's host over SSH. It is kept (not removed) when another connection on the same host (same ip_address) still uses it. This does not delete the connection record.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			connId path string true "ID of the connectionInfo"
//	@Success		200	{object}	model.SimpleMsg			"Agent uninstalled, or kept because it is still shared."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to uninstall the agent."
//	@Router			/source_group/{sgId}/connection_info/{connId}/agent [delete]
func UninstallAgent(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}
	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	if _, err := dao.SourceGroupGet(sgID); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if strings.TrimSpace(connectionInfo.IPAddress) == "" {
		return common.ReturnErrorMsg(c, "connection has no ip_address; the agent is installed only on SSH-reachable hosts")
	}

	// Keep the agent if another connection (in any source group) targets the same
	// host - it is a host-wide service shared across connections.
	shared, err := countConnectionsOnHost(connectionInfo.IPAddress, connectionInfo.ID)
	if err != nil {
		return common.ReturnInternalError(c, err, "failed to check other connections on the host")
	}
	if shared > 0 {
		return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "agent kept: host " +
			connectionInfo.IPAddress + " is still used by " + strconv.Itoa(shared) + " other connection(s)"}, " ")
	}

	// Load SSH secrets (OpenBao) before connecting.
	if err := hydrateConnectionSecrets(connectionInfo); err != nil {
		return common.ReturnInternalError(c, err, "failed to load SSH secrets")
	}

	s := &ssh.SSH{}
	if err := s.UninstallAgent(*connectionInfo); err != nil {
		return common.ReturnInternalError(c, err, "failed to uninstall the agent")
	}

	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "agent uninstalled from " + connectionInfo.IPAddress}, " ")
}

// countConnectionsOnHostOutsideGroup counts connections targeting the same host
// (ip_address) that belong to a source group other than sgID.
func countConnectionsOnHostOutsideGroup(ipAddress, sgID string) (int, error) {
	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{}, 0, 0)
	if err != nil {
		return 0, err
	}
	ip := strings.TrimSpace(ipAddress)
	count := 0
	for _, conn := range *list {
		if conn.SourceGroupID != sgID && strings.TrimSpace(conn.IPAddress) == ip {
			count++
		}
	}
	return count, nil
}

// AgentUninstallResult is one host's outcome in a source-group agent uninstall.
type AgentUninstallResult struct {
	Host    string `json:"host"`
	Message string `json:"message"`
}

// UninstallAgentSourceGroup godoc
//
//	@ID				uninstall-agent-source-group
//	@Summary		Uninstall Agent (Source Group)
//	@Description	Uninstall cm-honeybee-agent from every host in the source group over SSH. Each host is processed once; a host is kept (not removed) when a connection in another source group still uses it. This does not delete the connection records.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Success		200	{object}	model.SimpleMsg			"Per-host uninstall results."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to uninstall the agents."
//	@Router			/source_group/{sgId}/agent [delete]
func UninstallAgentSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}
	if _, err := dao.SourceGroupGet(sgID); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnInternalError(c, err, "failed to list connections")
	}

	results := make([]AgentUninstallResult, 0)
	done := make(map[string]bool) // process each host once

	for i := range *list {
		conn := (*list)[i]
		ip := strings.TrimSpace(conn.IPAddress)
		if ip == "" || done[ip] {
			continue
		}
		done[ip] = true

		shared, err := countConnectionsOnHostOutsideGroup(ip, sgID)
		if err != nil {
			results = append(results, AgentUninstallResult{ip, "error checking other connections: " + err.Error()})
			continue
		}
		if shared > 0 {
			results = append(results, AgentUninstallResult{ip, "kept: still used by " +
				strconv.Itoa(shared) + " connection(s) in other source group(s)"})
			continue
		}

		if err := hydrateConnectionSecrets(&conn); err != nil {
			results = append(results, AgentUninstallResult{ip, "failed to load SSH secrets: " + err.Error()})
			continue
		}
		s := &ssh.SSH{}
		if err := s.UninstallAgent(conn); err != nil {
			results = append(results, AgentUninstallResult{ip, "failed to uninstall: " + err.Error()})
			continue
		}
		results = append(results, AgentUninstallResult{ip, "uninstalled"})
	}

	return c.JSONPretty(http.StatusOK, map[string]interface{}{"results": results}, " ")
}

// RefreshConnectionInfoStatus godoc
//
//	@ID				refresh-connection-info-status
//	@Summary		Refresh Connection Info Status
//	@Description	Refresh the connection info status.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Param			connId path string true "ID of the connectionInfo"
//	@Success		200	{object}	model.SimpleMsg			"Successfully refresh the source group"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to refresh the source group"
//	@Router			/source_group/{sgId}/connection_info/{connId}/refresh [put]
func RefreshConnectionInfoStatus(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, err = doGetConnectionInfo(connID, true)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "success"}, " ")
}

// RefreshConnectionInfoStatusDirectly godoc
//
//	@ID				refresh-connection-info-status-directly
//	@Summary		Refresh Connection Info Status Directly
//	@Description	Refresh the connection info status directly.
//	@Tags			[On-premise] ConnectionInfo
//	@Accept			json
//	@Produce		json
//	@Param			connId path string true "ID of the connectionInfo"
//	@Success		200	{object}	model.SimpleMsg			"Successfully refresh the source group"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to refresh the source group"
//	@Router			/connection_info/{connId}/refresh [put]
func RefreshConnectionInfoStatusDirectly(c echo.Context) error {
	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, err := doGetConnectionInfo(connID, true)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "success"}, " ")
}

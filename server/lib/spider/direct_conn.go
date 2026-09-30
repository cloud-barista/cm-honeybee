package spider

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	alibabadrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/alibaba"
	awsdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws"
	azuredrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure"
	gcpdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/gcp"
	ibmdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibm"
	ktvpcdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/kt"
	ktdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ktclassic"
	ncpdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncp"
	nhndrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/nhn"
	openstackdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack"
	oracledrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/oracle"
	tencentdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"

	_ "github.com/cloud-barista/cm-honeybee/server/lib/spider/spiderroot"
)

// directConnectionName is the ConnectionName handed to drivers. Nothing is
// registered under it; drivers only use it for their call logs.
const directConnectionName = "honeybee"

// Conn describes a CSP connection for the driver-direct API. Credential keys
// are the CSP-side names honeybee stores (e.g. aws_access_key_id); they are
// mapped to cb-spider's names before the driver sees them.
type Conn struct {
	Provider   string
	Region     string
	Zone       string
	Credential []KeyValue
}

func normalizeDirectProvider(provider string) (string, error) {
	p := strings.ToUpper(strings.TrimSpace(provider))
	if p == "" {
		return "", errors.New("ProviderName is empty")
	}
	return p, nil
}

// cbspiderFile fails when a file cb-spider reads from $CBSPIDER_ROOT is
// missing, because cb-spider itself exits or panics in that case.
func cbspiderFile(rel string) error {
	root := os.Getenv("CBSPIDER_ROOT")
	if root == "" {
		return errors.New("$CBSPIDER_ROOT is not set")
	}
	if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
		return fmt.Errorf("cb-spider file is not readable: %w", err)
	}
	return nil
}

func directMetaInfo(provider string) (cim.CloudOSMetaInfo, error) {
	if err := cbspiderFile(filepath.Join("cloud-driver-libs", "cloudos_meta.yaml")); err != nil {
		return cim.CloudOSMetaInfo{}, err
	}
	return cim.GetCloudOSMetaInfo(provider)
}

// mapCredentialKeys renames CSP-side keys to cb-spider keys by the position of
// CredentialCSP[i] -> Credential[i]; unknown keys are kept as they are.
// Same as cb-spider's mapCredentialsCSPKeyToSpiderKeys().
func mapCredentialKeys(meta cim.CloudOSMetaInfo, kvList []KeyValue) []KeyValue {
	cspToSpider := make(map[string]string)
	for i, cspKey := range meta.CredentialCSP {
		if i < len(meta.Credential) {
			cspToSpider[cspKey] = meta.Credential[i]
		}
	}

	mapped := make([]KeyValue, 0, len(kvList))
	for _, kv := range kvList {
		if spiderKey, found := cspToSpider[kv.Key]; found {
			mapped = append(mapped, KeyValue{Key: spiderKey, Value: kv.Value})
		} else {
			mapped = append(mapped, kv)
		}
	}
	return mapped
}

// kvValue mirrors cb-spider's KeyValueListGetValue(), including the "Not set"
// value some drivers compare against.
func kvValue(kvList []KeyValue, key string) string {
	for _, kv := range kvList {
		if strings.EqualFold(kv.Key, key) {
			return kv.Value
		}
	}
	return "Not set"
}

// credentialInfo fills idrv.CredentialInfo the way cb-spider's
// createConnectionInfo() does. kvList must already carry cb-spider keys.
func credentialInfo(kvList []KeyValue) idrv.CredentialInfo {
	return idrv.CredentialInfo{
		ClientId:           kvValue(kvList, "ClientId"),
		ClientSecret:       kvValue(kvList, "ClientSecret"),
		StsToken:           kvValue(kvList, "StsToken"),
		TenantId:           kvValue(kvList, "TenantId"),
		SubscriptionId:     kvValue(kvList, "SubscriptionId"),
		IdentityEndpoint:   kvValue(kvList, "IdentityEndpoint"),
		Username:           kvValue(kvList, "Username"),
		Password:           kvValue(kvList, "Password"),
		DomainName:         kvValue(kvList, "DomainName"),
		ProjectID:          kvValue(kvList, "ProjectID"),
		AuthToken:          kvValue(kvList, "AuthToken"),
		ClientEmail:        kvValue(kvList, "ClientEmail"),
		PrivateKey:         kvValue(kvList, "PrivateKey"),
		Host:               kvValue(kvList, "Host"),
		APIVersion:         kvValue(kvList, "APIVersion"),
		MockName:           kvValue(kvList, "MockName"),
		ApiKey:             kvValue(kvList, "ApiKey"),
		ClusterId:          kvValue(kvList, "ClusterId"),
		RDSUserAccessKey:   kvValue(kvList, "User Access Key"),
		RDSSecretAccessKey: kvValue(kvList, "Secret Access Key"),
		RDSMySQLAppKey:     kvValue(kvList, "mysqlAppKey"),
		RDSMariaDBAppKey:   kvValue(kvList, "mariadbAppKey"),
		ConnectionName:     directConnectionName,
	}
}

// connectionInfo builds the driver input for c from memory; nothing is
// registered with cb-spider's store.
func connectionInfo(c Conn) (idrv.ConnectionInfo, error) {
	provider, err := normalizeDirectProvider(c.Provider)
	if err != nil {
		return idrv.ConnectionInfo{}, err
	}
	meta, err := directMetaInfo(provider)
	if err != nil {
		return idrv.ConnectionInfo{}, err
	}
	return idrv.ConnectionInfo{
		CredentialInfo: credentialInfo(mapCredentialKeys(meta, c.Credential)),
		RegionInfo: idrv.RegionInfo{
			Region: c.Region,
			Zone:   c.Zone,
		},
	}, nil
}

// cloudDriver selects a static driver the same way as cb-spider's
// getCloudDriver() in CloudDriverHandler_static.go, without MOCK.
func cloudDriver(provider string) (idrv.CloudDriver, error) {
	p, err := normalizeDirectProvider(provider)
	if err != nil {
		return nil, err
	}
	switch p {
	case "AWS":
		return new(awsdrv.AwsDriver), nil
	case "AZURE":
		return new(azuredrv.AzureDriver), nil
	case "GCP":
		return new(gcpdrv.GCPDriver), nil
	case "ALIBABA":
		return new(alibabadrv.AlibabaDriver), nil
	case "OPENSTACK":
		return new(openstackdrv.OpenStackDriver), nil
	case "TENCENT":
		return new(tencentdrv.TencentDriver), nil
	case "IBM":
		return new(ibmdrv.IbmCloudDriver), nil
	case "ORACLE":
		return new(oracledrv.OracleDriver), nil
	case "NCP":
		return new(ncpdrv.NcpVpcDriver), nil
	case "NHN":
		return new(nhndrv.NhnCloudDriver), nil
	case "KTCLASSIC":
		return new(ktdrv.KtCloudDriver), nil
	case "KT":
		return new(ktvpcdrv.KTCloudVpcDriver), nil
	default:
		return nil, fmt.Errorf("unsupported CSP driver: %q", provider)
	}
}

func connectInfo(provider string, info idrv.ConnectionInfo) (icon.CloudConnection, error) {
	if err := cbspiderFile(filepath.Join("conf", "calllog_conf.yaml")); err != nil {
		return nil, err
	}
	drv, err := cloudDriver(provider)
	if err != nil {
		return nil, err
	}
	conn, err := drv.ConnectCloud(info)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", provider, err)
	}
	return conn, nil
}

func connect(c Conn) (icon.CloudConnection, error) {
	if _, err := cloudDriver(c.Provider); err != nil {
		return nil, err
	}
	info, err := connectionInfo(c)
	if err != nil {
		return nil, err
	}
	return connectInfo(c.Provider, info)
}

// convertJSON copies a driver struct into a honeybee type through JSON, the
// same encoding cb-spider's REST API returns.
func convertJSON[T any](in any) (T, error) {
	var out T
	b, err := json.Marshal(in)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, fmt.Errorf("failed to decode driver output: %w", err)
	}
	return out, nil
}

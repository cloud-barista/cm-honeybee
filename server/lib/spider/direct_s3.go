package spider

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

// s3ConnInfo mirrors cb-spider's S3ConnectionInfo (S3Manager.go).
type s3ConnInfo struct {
	Endpoint       string
	AccessKey      string
	SecretKey      string
	UseSSL         bool
	RegionRequired bool
	Region         string
	ProviderName   string
	AppId          string
}

// s3AccessKey mirrors cb-spider's getAccessKey(): the first key that matches
// exactly wins, and a missing key yields "" (not "Not set").
func s3AccessKey(kvList []KeyValue, keys ...string) string {
	for _, key := range keys {
		for _, kv := range kvList {
			if kv.Key == key {
				return kv.Value
			}
		}
	}
	return ""
}

// openStackS3Endpoint mirrors getOpenStackS3Endpoint() in cb-spider's
// GetS3ConnectionInfo(): the IdentityEndpoint host with its port swapped for 8080.
func openStackS3Endpoint(kvList []KeyValue) (string, error) {
	identityEndpoint := kvValue(kvList, "IdentityEndpoint")
	if identityEndpoint == "" {
		return "", fmt.Errorf("IdentityEndpoint is required for OpenStack S3 connection")
	}

	parsedURL, err := url.Parse(identityEndpoint)
	if err != nil {
		return "", fmt.Errorf("failed to parse IdentityEndpoint URL: %v", err)
	}

	host := parsedURL.Host
	if parsedURL.Port() != "" {
		host = strings.Replace(host, ":"+parsedURL.Port(), ":8080", 1)
	} else {
		host = host + ":8080"
	}

	return host, nil
}

// s3ConnRule applies the per-provider endpoint and credential rules of
// cb-spider's GetS3ConnectionInfo() (S3Manager.go). kvList must already carry
// cb-spider keys. The Tencent AppId lookup is left to s3ConnectionInfo.
func s3ConnRule(providerName, regionID string, kvList []KeyValue) (*s3ConnInfo, error) {
	var accessKey, secretKey string
	var endpoint string
	var useSSL, regionRequired bool

	switch providerName {
	case "AWS":
		accessKey = kvValue(kvList, "ClientId")
		secretKey = kvValue(kvList, "ClientSecret")
		endpoint = fmt.Sprintf("s3.%s.amazonaws.com", regionID)
		useSSL = true
		regionRequired = true

	case "IBM":
		accessKey = s3AccessKey(kvList, "S3AccessKey", "access_key_id")
		secretKey = s3AccessKey(kvList, "S3SecretKey", "secret_access_key")
		endpoint = fmt.Sprintf("s3.%s.cloud-object-storage.appdomain.cloud", regionID)
		useSSL = true
		regionRequired = false

	case "OPENSTACK":
		accessKey = s3AccessKey(kvList, "S3AccessKey", "access")
		secretKey = s3AccessKey(kvList, "S3SecretKey", "secret")

		var err error
		endpoint, err = openStackS3Endpoint(kvList)
		if err != nil {
			return nil, err
		}

		useSSL = false
		regionRequired = false

	case "KT":
		accessKey = s3AccessKey(kvList, "S3AccessKey", "Access Key")
		secretKey = s3AccessKey(kvList, "S3SecretKey", "Secret Key")
		endpoint = "obj-e-1.ktcloud.com"
		useSSL = true
		regionRequired = false

	case "GCP":
		accessKey = s3AccessKey(kvList, "S3AccessKey", "Access Key")
		secretKey = s3AccessKey(kvList, "S3SecretKey", "Secret")
		endpoint = "storage.googleapis.com"
		useSSL = true
		regionRequired = true

	case "ALIBABA":
		accessKey = kvValue(kvList, "ClientId")
		secretKey = kvValue(kvList, "ClientSecret")
		endpoint = fmt.Sprintf("oss-%s.aliyuncs.com", regionID)
		useSSL = true
		regionRequired = false

	case "TENCENT":
		accessKey = kvValue(kvList, "ClientId")
		secretKey = kvValue(kvList, "ClientSecret")
		endpoint = fmt.Sprintf("cos.%s.myqcloud.com", regionID)
		useSSL = true
		regionRequired = true

	case "NCP":
		accessKey = kvValue(kvList, "ClientId")
		secretKey = kvValue(kvList, "ClientSecret")
		endpoint = fmt.Sprintf("%s.object.ncloudstorage.com", regionID)
		useSSL = true
		regionRequired = false

	case "NHN":
		accessKey = s3AccessKey(kvList, "S3AccessKey", "Access Key")
		secretKey = s3AccessKey(kvList, "S3SecretKey", "Secret Key")
		endpoint = fmt.Sprintf("%s-api-object-storage.nhncloudservice.com", regionID)
		useSSL = true
		regionRequired = true

	case "AZURE":
		accessKey = s3AccessKey(kvList, "S3AccessKey", "StorageAccountName")
		secretKey = s3AccessKey(kvList, "S3SecretKey", "StorageAccountKey")
		if accessKey != "" {
			endpoint = fmt.Sprintf("%s.blob.core.windows.net", accessKey)
		}
		useSSL = true
		regionRequired = false

	default:
		return nil, fmt.Errorf("provider '%s' does not support Object Storage service or is not configured for S3 access", providerName)
	}

	if accessKey == "" {
		return nil, fmt.Errorf("accessKey is empty for provider '%s'", providerName)
	}
	if secretKey == "" {
		return nil, fmt.Errorf("secretKey is empty for provider '%s'", providerName)
	}

	return &s3ConnInfo{
		Endpoint:       endpoint,
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		UseSSL:         useSSL,
		RegionRequired: regionRequired,
		Region:         regionID,
		ProviderName:   providerName,
	}, nil
}

// tencentAppId is cb-spider's getTencentAppId() (S3Manager.go). It is a
// variable so tests can run without the CAM API.
var tencentAppId = func(accessKey, secretKey string) (string, error) {
	credential := common.NewCredential(accessKey, secretKey)

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cam.tencentcloudapi.com"

	client, err := cam.NewClient(credential, "", cpf)
	if err != nil {
		return "", fmt.Errorf("failed to create CAM client: %v", err)
	}

	request := cam.NewGetUserAppIdRequest()

	response, err := client.GetUserAppId(request)
	if err != nil {
		return "", fmt.Errorf("failed to get AppId from CAM API: %v", err)
	}

	if response.Response == nil || response.Response.AppId == nil {
		return "", fmt.Errorf("AppId not found in CAM API response")
	}

	return fmt.Sprintf("%d", *response.Response.AppId), nil
}

// s3ConnRuleFor runs s3ConnRule on c after mapping its credential keys.
func s3ConnRuleFor(c Conn) (*s3ConnInfo, error) {
	providerName, err := upperProvider(c.Provider)
	if err != nil {
		return nil, err
	}
	meta, err := loadMetaInfo(providerName)
	if err != nil {
		return nil, err
	}
	return s3ConnRule(providerName, c.Region, mapCredentialKeys(meta, c.Credential))
}

// s3ConnectionInfo is cb-spider's GetS3ConnectionInfo() with the credential
// and region taken from c instead of spider's meta-DB.
func s3ConnectionInfo(c Conn) (*s3ConnInfo, error) {
	info, err := s3ConnRuleFor(c)
	if err != nil {
		return nil, err
	}

	if info.ProviderName == "TENCENT" {
		appId, err := tencentAppId(info.AccessKey, info.SecretKey)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve Tencent AppId: %v", err)
		}
		info.AppId = appId
	}
	return info, nil
}

// s3ClientOptions is the option half of cb-spider's NewS3Client() (S3Manager.go).
func s3ClientOptions(connInfo *s3ConnInfo) *minio.Options {
	options := &minio.Options{
		Creds:  credentials.NewStaticV4(connInfo.AccessKey, connInfo.SecretKey, ""),
		Secure: connInfo.UseSSL,
	}

	if connInfo.RegionRequired {
		options.Region = connInfo.Region
	}

	if connInfo.ProviderName == "TENCENT" {
		options.BucketLookup = minio.BucketLookupDNS
		options.Region = connInfo.Region
	}

	return options
}

// newS3Client is cb-spider's NewS3Client() (S3Manager.go).
func newS3Client(connInfo *s3ConnInfo) (*minio.Client, error) {
	return minio.New(connInfo.Endpoint, s3ClientOptions(connInfo))
}

// s3BucketEntry is the cspBucketEntry of cb-spider's ListAllS3BucketInfo().
type s3BucketEntry struct {
	Name         string
	CreationDate time.Time
}

// listS3BucketEntries lists the CSP buckets the way ListAllS3BucketInfo()
// (S3Manager.go) does: Azure containers through azblob, the rest through minio.
func listS3BucketEntries(connInfo *s3ConnInfo) ([]s3BucketEntry, error) {
	if connInfo.ProviderName == "AZURE" {
		return listAzureContainers(connInfo)
	}

	client, err := newS3Client(connInfo)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}
	var entries []s3BucketEntry
	for _, b := range buckets {
		entries = append(entries, s3BucketEntry{Name: b.Name, CreationDate: b.CreationDate})
	}
	return entries, nil
}

// s3BucketInfos turns CSP buckets into what honeybee used to read from
// cb-spider's GET /alls3info. With nothing registered in spider every bucket falls into
// OnlyCSPInfoList with NameId = SystemId = the CSP name, and CreationDate is
// the RFC3339Nano string time.Time marshals to.
func s3BucketInfos(entries []s3BucketEntry) []S3BucketInfo {
	out := make([]S3BucketInfo, 0, len(entries))
	for _, e := range entries {
		out = append(out, S3BucketInfo{
			Name:         e.Name,
			CreationDate: e.CreationDate.Format(time.RFC3339Nano),
		})
	}
	return out
}

// ListS3Buckets returns every bucket the CSP has, as cb-spider's
// GET /alls3info does through ListAllS3BucketInfo(). SystemId, the real bucket
// name on the CSP, is what Name carries.
func ListS3Buckets(c Conn) ([]S3BucketInfo, error) {
	connInfo, err := s3ConnectionInfo(c)
	if err != nil {
		return nil, err
	}
	entries, err := listS3BucketEntries(connInfo)
	if err != nil {
		return nil, err
	}
	return s3BucketInfos(entries), nil
}

// GetS3BucketLocation returns what cb-spider's GET /s3/{Name}?location
// answers through getBucketLocation() (rest-runtime/S3Rest.go). That handler asks
// no CSP: it returns the Region stored in spider's meta-DB when the bucket was
// created or registered there, and "" otherwise. honeybee registers nothing,
// so LocationConstraint is always "". Only the provider and credential rules
// are checked, without the Tencent CAM call, since the handler contacts nothing.
func GetS3BucketLocation(c Conn, bucketName string) (*S3BucketInfo, error) {
	if err := mustNonEmpty("BucketName", bucketName); err != nil {
		return nil, err
	}
	if _, err := s3ConnRuleFor(c); err != nil {
		return nil, err
	}
	return &S3BucketInfo{
		Name:   bucketName,
		Region: "",
	}, nil
}

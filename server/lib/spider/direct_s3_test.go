package spider

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/minio/minio-go/v7"
)

func TestS3ConnRulePerProvider(t *testing.T) {
	tests := []struct {
		provider       string
		region         string
		cred           []KeyValue
		endpoint       string
		useSSL         bool
		regionRequired bool
		access, secret string
	}{
		{"AWS", "ap-northeast-2",
			[]KeyValue{{Key: "aws_access_key_id", Value: "ak"}, {Key: "aws_secret_access_key", Value: "sk"}},
			"s3.ap-northeast-2.amazonaws.com", true, true, "ak", "sk"},
		{"IBM", "us-south",
			[]KeyValue{{Key: "ApiKey", Value: "api"}, {Key: "access_key_id", Value: "ak"}, {Key: "secret_access_key", Value: "sk"}},
			"s3.us-south.cloud-object-storage.appdomain.cloud", true, false, "ak", "sk"},
		{"OPENSTACK", "RegionOne",
			[]KeyValue{{Key: "IdentityEndpoint", Value: "http://192.0.2.10:5000/v3"}, {Key: "access", Value: "ak"}, {Key: "secret", Value: "sk"}},
			"192.0.2.10:8080", false, false, "ak", "sk"},
		{"KT", "KR1",
			[]KeyValue{{Key: "Access Key", Value: "ak"}, {Key: "Secret Key", Value: "sk"}},
			"obj-e-1.ktcloud.com", true, false, "ak", "sk"},
		{"GCP", "asia-northeast3",
			[]KeyValue{{Key: "project_id", Value: "p"}, {Key: "Access Key", Value: "ak"}, {Key: "Secret", Value: "sk"}},
			"storage.googleapis.com", true, true, "ak", "sk"},
		{"ALIBABA", "ap-northeast-2",
			[]KeyValue{{Key: "AccessKeyId", Value: "ak"}, {Key: "AccessKeySecret", Value: "sk"}},
			"oss-ap-northeast-2.aliyuncs.com", true, false, "ak", "sk"},
		{"TENCENT", "ap-seoul",
			[]KeyValue{{Key: "SecretId", Value: "ak"}, {Key: "SecretKey", Value: "sk"}},
			"cos.ap-seoul.myqcloud.com", true, true, "ak", "sk"},
		{"NCP", "kr",
			[]KeyValue{{Key: "ncloud_access_key", Value: "ak"}, {Key: "ncloud_secret_key", Value: "sk"}},
			"kr.object.ncloudstorage.com", true, false, "ak", "sk"},
		{"NHN", "kr1",
			[]KeyValue{{Key: "Access Key", Value: "ak"}, {Key: "Secret Key", Value: "sk"}},
			"kr1-api-object-storage.nhncloudservice.com", true, true, "ak", "sk"},
		{"AZURE", "koreacentral",
			[]KeyValue{{Key: "clientId", Value: "c"}, {Key: "StorageAccountName", Value: "acct"}, {Key: "StorageAccountKey", Value: "sk"}},
			"acct.blob.core.windows.net", true, false, "acct", "sk"},
	}
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got, err := s3ConnRuleFor(Conn{Provider: strings.ToLower(tt.provider), Region: tt.region, Credential: tt.cred})
			if err != nil {
				t.Fatal(err)
			}
			if got.Endpoint != tt.endpoint || got.UseSSL != tt.useSSL || got.RegionRequired != tt.regionRequired {
				t.Fatalf("got endpoint=%q ssl=%v regionRequired=%v, want %q %v %v",
					got.Endpoint, got.UseSSL, got.RegionRequired, tt.endpoint, tt.useSSL, tt.regionRequired)
			}
			if got.AccessKey != tt.access || got.SecretKey != tt.secret {
				t.Fatalf("got keys %q/%q, want %q/%q", got.AccessKey, got.SecretKey, tt.access, tt.secret)
			}
			if got.Region != tt.region || got.ProviderName != tt.provider {
				t.Fatalf("got region=%q provider=%q", got.Region, got.ProviderName)
			}
		})
	}
}

func TestS3ConnRuleS3KeysWin(t *testing.T) {
	for _, provider := range []string{"IBM", "OPENSTACK", "KT", "GCP", "NHN", "AZURE"} {
		cred := []KeyValue{
			{Key: "IdentityEndpoint", Value: "https://keystone.example"},
			{Key: "S3AccessKey", Value: "s3ak"},
			{Key: "S3SecretKey", Value: "s3sk"},
			{Key: "access_key_id", Value: "x"}, {Key: "secret_access_key", Value: "x"},
			{Key: "access", Value: "x"}, {Key: "secret", Value: "x"},
			{Key: "Access Key", Value: "x"}, {Key: "Secret Key", Value: "x"}, {Key: "Secret", Value: "x"},
			{Key: "StorageAccountName", Value: "x"}, {Key: "StorageAccountKey", Value: "x"},
		}
		got, err := s3ConnRule(provider, "r", cred)
		if err != nil {
			t.Fatalf("%s: %v", provider, err)
		}
		if got.AccessKey != "s3ak" || got.SecretKey != "s3sk" {
			t.Fatalf("%s: got %q/%q, want S3AccessKey/S3SecretKey", provider, got.AccessKey, got.SecretKey)
		}
	}
}

func TestS3ConnRuleKeyMatchIsExact(t *testing.T) {
	_, err := s3ConnRule("KT", "KR1", []KeyValue{{Key: "access key", Value: "ak"}, {Key: "secret key", Value: "sk"}})
	if err == nil || !strings.Contains(err.Error(), "accessKey is empty") {
		t.Fatalf("got %v, want accessKey is empty", err)
	}
}

func TestS3ConnRuleMissingKeys(t *testing.T) {
	_, err := s3ConnRule("NHN", "kr1", []KeyValue{{Key: "Access Key", Value: "ak"}})
	if err == nil || !strings.Contains(err.Error(), "secretKey is empty for provider 'NHN'") {
		t.Fatalf("got %v", err)
	}
	_, err = s3ConnRule("AZURE", "koreacentral", []KeyValue{{Key: "ClientId", Value: "c"}})
	if err == nil || !strings.Contains(err.Error(), "accessKey is empty for provider 'AZURE'") {
		t.Fatalf("got %v", err)
	}
	// Like cb-spider, AWS reads through KeyValueListGetValue, which yields
	// "Not set" instead of "" for a missing key.
	got, err := s3ConnRule("AWS", "us-east-1", nil)
	if err != nil || got.AccessKey != "Not set" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestOpenStackS3Endpoint(t *testing.T) {
	tests := map[string]string{
		"http://192.0.2.10:5000/v3":     "192.0.2.10:8080",
		"https://keystone.example/v3": "keystone.example:8080",
	}
	for in, want := range tests {
		got, err := openStackS3Endpoint([]KeyValue{{Key: "IdentityEndpoint", Value: in}})
		if err != nil || got != want {
			t.Fatalf("%s: got %q, %v, want %q", in, got, err, want)
		}
	}
	if _, err := openStackS3Endpoint([]KeyValue{{Key: "IdentityEndpoint", Value: ""}}); err == nil {
		t.Fatal("empty IdentityEndpoint must fail")
	}
}

func TestS3ConnRuleUnsupportedProvider(t *testing.T) {
	for _, provider := range []string{"ORACLE", "KTCLASSIC", "NOPE"} {
		_, err := s3ConnRule(provider, "r", []KeyValue{{Key: "ClientId", Value: "a"}, {Key: "ClientSecret", Value: "b"}})
		if err == nil || !strings.Contains(err.Error(), "does not support Object Storage") {
			t.Fatalf("%s: got %v", provider, err)
		}
	}
	if _, err := DirectListS3Buckets(Conn{Provider: "ORACLE", Region: "r"}); err == nil {
		t.Fatal("ORACLE must fail")
	}
	if _, err := DirectGetS3BucketLocation(Conn{Provider: "", Region: "r"}, "b"); err == nil {
		t.Fatal("empty provider must fail")
	}
}

func TestS3ClientOptions(t *testing.T) {
	tests := []struct {
		info   s3ConnInfo
		region string
		lookup minio.BucketLookupType
	}{
		{s3ConnInfo{ProviderName: "AWS", Region: "ap-northeast-2", RegionRequired: true, UseSSL: true}, "ap-northeast-2", minio.BucketLookupAuto},
		{s3ConnInfo{ProviderName: "NCP", Region: "kr", RegionRequired: false, UseSSL: true}, "", minio.BucketLookupAuto},
		{s3ConnInfo{ProviderName: "TENCENT", Region: "ap-seoul", RegionRequired: false, UseSSL: true}, "ap-seoul", minio.BucketLookupDNS},
		{s3ConnInfo{ProviderName: "OPENSTACK", Region: "RegionOne", UseSSL: false}, "", minio.BucketLookupAuto},
	}
	for _, tt := range tests {
		opt := s3ClientOptions(&tt.info)
		if opt.Region != tt.region || opt.BucketLookup != tt.lookup || opt.Secure != tt.info.UseSSL {
			t.Fatalf("%s: got region=%q lookup=%v secure=%v", tt.info.ProviderName, opt.Region, opt.BucketLookup, opt.Secure)
		}
	}

	client, err := newS3Client(&s3ConnInfo{ProviderName: "OPENSTACK", Endpoint: "192.0.2.10:8080", AccessKey: "a", SecretKey: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if u := client.EndpointURL(); u.Scheme != "http" || u.Host != "192.0.2.10:8080" {
		t.Fatalf("got %s", u)
	}
}

func TestS3ConnectionInfoTencentAppId(t *testing.T) {
	saved := tencentAppId
	defer func() { tencentAppId = saved }()

	var gotAK, gotSK string
	tencentAppId = func(ak, sk string) (string, error) {
		gotAK, gotSK = ak, sk
		return "1250000000", nil
	}
	info, err := s3ConnectionInfo(Conn{Provider: "TENCENT", Region: "ap-seoul",
		Credential: []KeyValue{{Key: "SecretId", Value: "ak"}, {Key: "SecretKey", Value: "sk"}}})
	if err != nil {
		t.Fatal(err)
	}
	if info.AppId != "1250000000" || gotAK != "ak" || gotSK != "sk" {
		t.Fatalf("got AppId=%q keys=%q/%q", info.AppId, gotAK, gotSK)
	}

	tencentAppId = func(string, string) (string, error) { return "", errors.New("cam down") }
	if _, err := s3ConnectionInfo(Conn{Provider: "TENCENT", Region: "ap-seoul",
		Credential: []KeyValue{{Key: "SecretId", Value: "ak"}, {Key: "SecretKey", Value: "sk"}}}); err == nil ||
		!strings.Contains(err.Error(), "failed to retrieve Tencent AppId") {
		t.Fatalf("got %v", err)
	}

	tencentAppId = func(string, string) (string, error) {
		t.Fatal("location must not call the CAM API")
		return "", nil
	}
	loc, err := DirectGetS3BucketLocation(Conn{Provider: "TENCENT", Region: "ap-seoul",
		Credential: []KeyValue{{Key: "SecretId", Value: "ak"}, {Key: "SecretKey", Value: "sk"}}}, "b-1250000000")
	if err != nil || loc.Name != "b-1250000000" || loc.Region != "" {
		t.Fatalf("got %+v, %v", loc, err)
	}
}

func TestListS3BucketEntriesMinio(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/" {
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<ListAllMyBucketsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
<Owner><ID>o</ID></Owner>
<Buckets>
<Bucket><Name>alpha</Name><CreationDate>2025-01-02T03:04:05.000Z</CreationDate></Bucket>
<Bucket><Name>beta-1250000000</Name><CreationDate>2025-06-07T08:09:10.123Z</CreationDate></Bucket>
</Buckets>
</ListAllMyBucketsResult>`))
	}))
	defer srv.Close()

	entries, err := listS3BucketEntries(&s3ConnInfo{
		ProviderName: "OPENSTACK",
		Endpoint:     strings.TrimPrefix(srv.URL, "http://"),
		AccessKey:    "a",
		SecretKey:    "b",
		Region:       "RegionOne",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := s3BucketInfos(entries)
	want := []S3BucketInfo{
		{Name: "alpha", CreationDate: "2025-01-02T03:04:05Z"},
		{Name: "beta-1250000000", CreationDate: "2025-06-07T08:09:10.123Z"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %+v, want %+v", got[i], want[i])
		}
	}
}

func TestS3BucketInfosCreationDateMatchesJSON(t *testing.T) {
	type spiderEntry struct {
		CreationDate time.Time
	}
	for _, ts := range []time.Time{{}, time.Date(2025, 1, 2, 3, 4, 5, 600, time.UTC)} {
		want, err := convertJSON[struct{ CreationDate string }](spiderEntry{CreationDate: ts})
		if err != nil {
			t.Fatal(err)
		}
		got := s3BucketInfos([]s3BucketEntry{{Name: "b", CreationDate: ts}})
		if got[0].CreationDate != want.CreationDate {
			t.Fatalf("got %q, want %q", got[0].CreationDate, want.CreationDate)
		}
	}
}

func TestAzureBlobClient(t *testing.T) {
	if got := azureBlobServiceURL("acct"); got != "https://acct.blob.core.windows.net/" {
		t.Fatalf("got %q", got)
	}
	client, err := newAzureBlobClient(&s3ConnInfo{ProviderName: "AZURE", AccessKey: "acct", SecretKey: "c2VjcmV0"})
	if err != nil {
		t.Fatal(err)
	}
	if got := client.URL(); got != "https://acct.blob.core.windows.net/" {
		t.Fatalf("got %q", got)
	}
	if _, err := newAzureBlobClient(&s3ConnInfo{ProviderName: "AZURE", AccessKey: "acct", SecretKey: "not base64!"}); err == nil ||
		!strings.Contains(err.Error(), "SharedKeyCredential") {
		t.Fatalf("got %v", err)
	}
}

func TestAzureContainerEntries(t *testing.T) {
	name1, name2 := "c1", "c2"
	lm := time.Date(2025, 3, 4, 5, 6, 7, 0, time.UTC)
	got := azureContainerEntries([]*service.ContainerItem{
		{Name: &name1, Properties: &service.ContainerProperties{LastModified: &lm}},
		{Name: &name2},
		{Name: nil},
		nil,
	})
	if len(got) != 2 || got[0].Name != "c1" || !got[0].CreationDate.Equal(lm) ||
		got[1].Name != "c2" || !got[1].CreationDate.IsZero() {
		t.Fatalf("got %+v", got)
	}
}

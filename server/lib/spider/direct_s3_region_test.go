package spider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/minio/minio-go/v7"
)

// s3RegionServer stands in for the S3 endpoint of ap-northeast-3. Each HEAD on
// a bucket gets the status and x-amz-bucket-region header listed for it; an
// unlisted bucket gets 404. maxInFlight records the most requests seen at once.
func s3RegionServer(t *testing.T, answers map[string]struct {
	status int
	region string
}, maxInFlight *int32) *minio.Client {
	t.Helper()

	var inFlight int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&inFlight, 1)
		defer atomic.AddInt32(&inFlight, -1)
		for {
			old := atomic.LoadInt32(maxInFlight)
			if n <= old || atomic.CompareAndSwapInt32(maxInFlight, old, n) {
				break
			}
		}

		if r.Method != http.MethodHead {
			http.Error(w, "unexpected "+r.Method, http.StatusBadRequest)
			return
		}
		a, has := answers[strings.Trim(r.URL.Path, "/")]
		if !has {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if a.region != "" {
			w.Header().Set("x-amz-bucket-region", a.region)
		}
		w.WriteHeader(a.status)
	}))
	t.Cleanup(srv.Close)

	client, err := newS3Client(&s3ConnInfo{
		ProviderName:   "AWS",
		Endpoint:       strings.TrimPrefix(srv.URL, "http://"),
		AccessKey:      "a",
		SecretKey:      "b",
		RegionRequired: true,
		Region:         "ap-northeast-3",
	})
	if err != nil {
		t.Fatal(err)
	}

	return client
}

// The region comes from the redirect or denial header when S3 sends one, and
// is the endpoint's own region when the endpoint serves the bucket. A bucket
// S3 says nothing about is an error of its own, not a region.
func TestS3BucketRegions(t *testing.T) {
	var maxInFlight int32
	client := s3RegionServer(t, map[string]struct {
		status int
		region string
	}{
		"near":        {http.StatusOK, ""},
		"near-header": {http.StatusOK, "ap-northeast-3"},
		"far":         {http.StatusMovedPermanently, "ap-northeast-2"},
		"far-denied":  {http.StatusForbidden, "us-east-1"},
		"denied":      {http.StatusForbidden, ""},
	}, &maxInFlight)

	names := []string{"near", "near-header", "far", "far-denied", "denied", "gone"}
	got := s3BucketRegions(context.Background(), client, "ap-northeast-3", names, 2)

	want := map[string]string{
		"near":        "ap-northeast-3",
		"near-header": "ap-northeast-3",
		"far":         "ap-northeast-2",
		"far-denied":  "us-east-1",
	}
	for name, region := range want {
		if r := got[name]; r.Err != nil || r.Region != region {
			t.Errorf("%s: got %q, %v; want %q", name, r.Region, r.Err, region)
		}
	}
	for _, name := range []string{"denied", "gone"} {
		if r := got[name]; r.Err == nil || r.Region != "" || !strings.Contains(r.Err.Error(), "'"+name+"'") {
			t.Errorf("%s: got %q, %v; want an error naming the bucket", name, r.Region, r.Err)
		}
	}
	if len(got) != len(names) {
		t.Errorf("got %d entries, want %d", len(got), len(names))
	}
	if maxInFlight > 2 {
		t.Errorf("%d lookups ran at once, want at most 2", maxInFlight)
	}
}

// A provider whose bucket region cannot be read answers nil without contacting
// the CSP, and GetS3BucketLocation reports its region as unknown.
func TestGetS3BucketRegionsUnknownProvider(t *testing.T) {
	saved := tencentAppId
	defer func() { tencentAppId = saved }()
	tencentAppId = func(string, string) (string, error) {
		t.Fatal("an unknown-region provider must not contact the CSP")
		return "", nil
	}

	for _, c := range []Conn{
		{Provider: "tencent", Region: "ap-seoul", Credential: []KeyValue{{Key: "SecretId", Value: "ak"}, {Key: "SecretKey", Value: "sk"}}},
		{Provider: "kt", Region: "KR1", Credential: []KeyValue{{Key: "Access Key", Value: "ak"}, {Key: "Secret Key", Value: "sk"}}},
		{Provider: "azure", Region: "koreacentral", Credential: []KeyValue{{Key: "StorageAccountName", Value: "acct"}, {Key: "StorageAccountKey", Value: "sk"}}},
	} {
		got, err := GetS3BucketRegions(c, []string{"b"})
		if err != nil || got != nil {
			t.Errorf("%s: got %v, %v; want nil, nil", c.Provider, got, err)
		}
		loc, err := GetS3BucketLocation(c, "b")
		if err != nil || loc.Name != "b" || loc.Region != "" {
			t.Errorf("%s: location %+v, %v; want an unknown region", c.Provider, loc, err)
		}
	}

	if _, err := GetS3BucketRegions(Conn{Provider: "ORACLE", Region: "r"}, []string{"b"}); err == nil {
		t.Error("a provider without object storage must fail")
	}
}

func TestBucketRegionProviders(t *testing.T) {
	for _, p := range []string{"AWS"} {
		if !bucketRegionProviders[p] {
			t.Errorf("%s must read the bucket region", p)
		}
	}
	for _, p := range []string{"AZURE", "GCP", "IBM", "ALIBABA", "TENCENT", "NCP", "NHN", "KT", "OPENSTACK"} {
		if bucketRegionProviders[p] {
			t.Errorf("%s must leave the bucket region unknown", p)
		}
	}
}

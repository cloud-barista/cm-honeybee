package spider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
)

func TestPolicyAllowsAnyone(t *testing.T) {
	cases := []struct {
		name   string
		policy string
		want   bool
	}{
		{"no policy", "", false},
		{"principal star", `{"Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject"}]}`, true},
		{"aws star", `{"Statement":[{"Effect":"Allow","Principal":{"AWS":"*"}}]}`, true},
		{"aws star list", `{"Statement":[{"Effect":"Allow","Principal":{"AWS":["arn:aws:iam::1:root","*"]}}]}`, true},
		{"single statement object", `{"Statement":{"Effect":"Allow","Principal":"*"}}`, true},
		{"deny star", `{"Statement":[{"Effect":"Deny","Principal":"*"}]}`, false},
		{"named account", `{"Statement":[{"Effect":"Allow","Principal":{"AWS":"arn:aws:iam::1:root"}}]}`, false},
		{"broken json", `{"Statement":`, false},
	}
	for _, c := range cases {
		if got := policyAllowsAnyone(c.policy); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestSplitCSV(t *testing.T) {
	s := "GET, PUT,,POST "
	got := splitCSV(&s)
	if len(got) != 3 || got[0] != "GET" || got[1] != "PUT" || got[2] != "POST" {
		t.Errorf("got %q", got)
	}
	if splitCSV(nil) != nil {
		t.Error("nil must give nil")
	}
}

// s3DetailServer answers every bucket sub-resource with 403 AccessDenied, except
// the ones listed in ok, which get the given body with 200, and the ones listed
// in codes, which get that S3 error code with 404.
func s3DetailServer(t *testing.T, ok map[string]string, codes map[string]string) *minio.Client {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		item := "objects"
		for _, sub := range []string{"versioning", "cors", "encryption", "policy", "tagging"} {
			if _, has := r.URL.Query()[sub]; has {
				item = sub
			}
		}
		w.Header().Set("Content-Type", "application/xml")
		if body, has := ok[item]; has {
			_, _ = w.Write([]byte(body))
			return
		}
		code, status := "AccessDenied", http.StatusForbidden
		if c, has := codes[item]; has {
			code, status = c, http.StatusNotFound
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>` + code +
			`</Code><Message>denied</Message><BucketName>b</BucketName></Error>`))
	}))
	t.Cleanup(srv.Close)

	client, err := newS3Client(&s3ConnInfo{
		ProviderName:   "AWS",
		Endpoint:       strings.TrimPrefix(srv.URL, "http://"),
		AccessKey:      "a",
		SecretKey:      "b",
		RegionRequired: true,
		Region:         "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	return client
}

// A bucket the CSP answers nothing about is reported as nothing read, with one
// error per item asked.
func TestS3BucketDetail_NothingRead(t *testing.T) {
	client := s3DetailServer(t, nil, nil)

	d := s3BucketDetail(context.Background(), client, "bucket-a", true)
	if !d.NothingRead() {
		t.Errorf("NothingRead() = false, want true (tried %d, read %d, errors %q)", d.ItemsTried, d.ItemsRead, d.Errors)
	}
	if d.ItemsTried != 6 || d.ItemsRead != 0 || len(d.Errors) != 6 {
		t.Errorf("tried %d, read %d, %d errors; want 6, 0, 6", d.ItemsTried, d.ItemsRead, len(d.Errors))
	}
	for _, e := range d.Errors {
		if !strings.Contains(e, "denied") {
			t.Errorf("error %q is not the server's answer", e)
		}
	}
}

// One item answered is enough for the bucket to count as collected; the
// failures stay in Errors.
func TestS3BucketDetail_SomethingRead(t *testing.T) {
	client := s3DetailServer(t, map[string]string{
		"versioning": `<?xml version="1.0" encoding="UTF-8"?><VersioningConfiguration><Status>Enabled</Status></VersioningConfiguration>`,
	}, nil)

	d := s3BucketDetail(context.Background(), client, "bucket-a", false)
	if d.NothingRead() {
		t.Errorf("NothingRead() = true, want false (errors %q)", d.Errors)
	}
	if !d.VersioningEnabled {
		t.Error("versioning was answered as enabled")
	}
	if d.ItemsTried != 5 || d.ItemsRead != 1 || len(d.Errors) != 4 {
		t.Errorf("tried %d, read %d, %d errors; want 5, 1, 4", d.ItemsTried, d.ItemsRead, len(d.Errors))
	}
}

// "Not configured" is an answer: encryption and tags read that way count as
// read, not as failures.
func TestS3BucketDetail_NotConfiguredIsRead(t *testing.T) {
	client := s3DetailServer(t, nil, map[string]string{
		"encryption": codeNoEncryption,
		"tagging":    codeNoTagSet,
	})

	d := s3BucketDetail(context.Background(), client, "bucket-a", false)
	if d.NothingRead() {
		t.Errorf("NothingRead() = true, want false (errors %q)", d.Errors)
	}
	if d.ItemsRead != 2 || len(d.Errors) != 3 {
		t.Errorf("read %d, %d errors; want 2, 3 (errors %q)", d.ItemsRead, len(d.Errors), d.Errors)
	}
}

// A zero detail was never collected, so it does not count as a failed one.
func TestS3BucketDetail_ZeroIsNotNothingRead(t *testing.T) {
	if (&S3BucketDetail{}).NothingRead() {
		t.Error("a detail with nothing tried must not read as nothing read")
	}
}

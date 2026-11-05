package spider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

// bucketDetailTimeout bounds one bucket's detail collection. Counting objects
// walks the whole bucket, so it gets more room than a single driver call.
var bucketDetailTimeout = 10 * time.Minute

// S3CORSRule is one CORS rule of a bucket.
type S3CORSRule struct {
	AllowedOrigin []string
	AllowedMethod []string
	AllowedHeader []string
	ExposeHeader  []string
	MaxAgeSeconds int
}

// S3BucketDetail is the configuration and usage of one bucket, as the CSP
// reports it through its S3 (or Azure Blob) API.
type S3BucketDetail struct {
	Name              string
	CreationDate      string
	VersioningEnabled bool
	CORSRules         []S3CORSRule
	EncryptionEnabled bool
	IsPublic          bool
	Tags              map[string]string

	// ObjectsCounted is false when counting was not asked for or failed; the
	// two totals are then zero and mean nothing.
	ObjectsCounted bool
	TotalSizeBytes int64
	ObjectCount    int64

	// Errors lists the items that could not be read, one "<item>: <reason>"
	// each. The matching fields keep their zero value, so a reader must check
	// here before taking a false or empty value as observed.
	Errors []string
}

// GetS3BucketDetail reads versioning, CORS, default encryption, public access,
// tags and, when countObjects is set, the total size and object count of one
// bucket. A failure on one item is recorded in Errors and does not stop the
// others; only a connection-level failure returns an error.
func GetS3BucketDetail(c Conn, bucketName string, countObjects bool) (*S3BucketDetail, error) {
	return withTimeout(bucketDetailTimeout, func() (*S3BucketDetail, error) {
		return getS3BucketDetail(c, bucketName, countObjects)
	})
}

func getS3BucketDetail(c Conn, bucketName string, countObjects bool) (*S3BucketDetail, error) {
	if err := mustNonEmpty("BucketName", bucketName); err != nil {
		return nil, err
	}
	connInfo, err := s3ConnectionInfo(c)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), bucketDetailTimeout)
	defer cancel()

	if connInfo.ProviderName == "AZURE" {
		return azureContainerDetail(ctx, connInfo, bucketName, countObjects)
	}
	client, err := newS3Client(connInfo)
	if err != nil {
		return nil, err
	}
	return s3BucketDetail(ctx, client, bucketName, countObjects), nil
}

// S3 error codes that mean "not configured" rather than a failure.
const (
	codeNoEncryption = "ServerSideEncryptionConfigurationNotFoundError"
	codeNoTagSet     = "NoSuchTagSet"
)

func s3BucketDetail(ctx context.Context, client *minio.Client, bucketName string, countObjects bool) *S3BucketDetail {
	d := &S3BucketDetail{Name: bucketName}
	fail := func(item string, err error) {
		d.Errors = append(d.Errors, item+": "+err.Error())
	}

	if v, err := client.GetBucketVersioning(ctx, bucketName); err != nil {
		fail("versioning", err)
	} else {
		d.VersioningEnabled = v.Enabled()
	}

	// GetBucketCors answers nil, nil when no CORS configuration is set.
	if cfg, err := client.GetBucketCors(ctx, bucketName); err != nil {
		fail("cors", err)
	} else if cfg != nil {
		for _, r := range cfg.CORSRules {
			d.CORSRules = append(d.CORSRules, S3CORSRule{
				AllowedOrigin: r.AllowedOrigin,
				AllowedMethod: r.AllowedMethod,
				AllowedHeader: r.AllowedHeader,
				ExposeHeader:  r.ExposeHeader,
				MaxAgeSeconds: r.MaxAgeSeconds,
			})
		}
	}

	if cfg, err := client.GetBucketEncryption(ctx, bucketName); err != nil {
		if minio.ToErrorResponse(err).Code != codeNoEncryption {
			fail("encryption", err)
		}
	} else {
		d.EncryptionEnabled = cfg != nil && len(cfg.Rules) > 0
	}

	// GetBucketPolicy answers "", nil when the bucket has no policy. Only the
	// policy is visible through the S3 API; account-level blocks and ACLs are not.
	if policy, err := client.GetBucketPolicy(ctx, bucketName); err != nil {
		fail("policy", err)
	} else {
		d.IsPublic = policyAllowsAnyone(policy)
	}

	if t, err := client.GetBucketTagging(ctx, bucketName); err != nil {
		if minio.ToErrorResponse(err).Code != codeNoTagSet {
			fail("tagging", err)
		}
	} else if t != nil {
		if m := t.ToMap(); len(m) > 0 {
			d.Tags = m
		}
	}

	if countObjects {
		var size, count int64
		var listErr error
		for obj := range client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Recursive: true}) {
			if obj.Err != nil {
				listErr = obj.Err
				break
			}
			size += obj.Size
			count++
		}
		if listErr != nil {
			fail("objects", listErr)
		} else {
			d.ObjectsCounted, d.TotalSizeBytes, d.ObjectCount = true, size, count
		}
	}

	return d
}

// policyAllowsAnyone reports whether a bucket policy has an Allow statement
// whose principal is everyone ("*" or {"AWS": "*"}). Conditions are not
// evaluated, so a statement limited by, say, source IP still counts as public.
func policyAllowsAnyone(policy string) bool {
	if strings.TrimSpace(policy) == "" {
		return false
	}
	var doc struct {
		Statement json.RawMessage `json:"Statement"`
	}
	if json.Unmarshal([]byte(policy), &doc) != nil {
		return false
	}
	type statement struct {
		Effect    string          `json:"Effect"`
		Principal json.RawMessage `json:"Principal"`
	}
	var stmts []statement
	if json.Unmarshal(doc.Statement, &stmts) != nil {
		var one statement
		if json.Unmarshal(doc.Statement, &one) != nil {
			return false
		}
		stmts = []statement{one}
	}
	for _, s := range stmts {
		if strings.EqualFold(s.Effect, "Allow") && principalIsAnyone(s.Principal) {
			return true
		}
	}
	return false
}

func principalIsAnyone(raw json.RawMessage) bool {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s == "*"
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return false
	}
	for _, v := range m {
		if json.Unmarshal(v, &s) == nil && s == "*" {
			return true
		}
		var list []string
		if json.Unmarshal(v, &list) == nil {
			for _, p := range list {
				if p == "*" {
					return true
				}
			}
		}
	}
	return false
}

// errAzureVersioning explains the one item the Blob data plane cannot answer.
var errAzureVersioning = errors.New("not readable through the Blob API (it is a storage account setting in Azure Resource Manager)")

// azureContainerDetail reads a container. CORS is a storage account setting in
// Azure, so the account's rules are reported for every container; encryption
// at rest cannot be turned off in Azure Storage, so it is always true.
func azureContainerDetail(ctx context.Context, connInfo *s3ConnInfo, name string, countObjects bool) (*S3BucketDetail, error) {
	client, err := newAzureBlobClient(connInfo)
	if err != nil {
		return nil, err
	}
	d := &S3BucketDetail{Name: name, EncryptionEnabled: true}
	fail := func(item string, err error) {
		d.Errors = append(d.Errors, item+": "+err.Error())
	}
	fail("versioning", errAzureVersioning)

	if props, err := client.ServiceClient().GetProperties(ctx, nil); err != nil {
		fail("cors", err)
	} else {
		for _, r := range props.CORS {
			if r == nil {
				continue
			}
			rule := S3CORSRule{
				AllowedOrigin: splitCSV(r.AllowedOrigins),
				AllowedMethod: splitCSV(r.AllowedMethods),
				AllowedHeader: splitCSV(r.AllowedHeaders),
				ExposeHeader:  splitCSV(r.ExposedHeaders),
			}
			if r.MaxAgeInSeconds != nil {
				rule.MaxAgeSeconds = int(*r.MaxAgeInSeconds)
			}
			d.CORSRules = append(d.CORSRules, rule)
		}
	}

	container := client.ServiceClient().NewContainerClient(name)
	if props, err := container.GetProperties(ctx, nil); err != nil {
		fail("properties", err)
	} else {
		d.IsPublic = props.BlobPublicAccess != nil && *props.BlobPublicAccess != ""
		if props.LastModified != nil {
			d.CreationDate = props.LastModified.Format(time.RFC3339Nano)
		}
		for k, v := range props.Metadata {
			if v == nil {
				continue
			}
			if d.Tags == nil {
				d.Tags = map[string]string{}
			}
			d.Tags[k] = *v
		}
	}

	if countObjects {
		var size, count int64
		pager := container.NewListBlobsFlatPager(nil)
		var listErr error
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				listErr = fmt.Errorf("failed to list blobs: %w", err)
				break
			}
			for _, b := range page.Segment.BlobItems {
				count++
				if b.Properties != nil && b.Properties.ContentLength != nil {
					size += *b.Properties.ContentLength
				}
			}
		}
		if listErr != nil {
			fail("objects", listErr)
		} else {
			d.ObjectsCounted, d.TotalSizeBytes, d.ObjectCount = true, size, count
		}
	}

	return d, nil
}

func splitCSV(s *string) []string {
	if s == nil || *s == "" {
		return nil
	}
	parts := strings.Split(*s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

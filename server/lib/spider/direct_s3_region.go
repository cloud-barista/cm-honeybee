package spider

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
)

// Where a bucket lives, per provider. A source group is bound to one region,
// and the bucket listing of most CSPs covers the whole account, so the region
// of each bucket has to be asked separately. It is asked only where the CSP
// answers with a region of the same kind as the one the source group carries;
// elsewhere the region stays unknown ("") and callers keep every bucket.
//
//	Provider        Region read   How, or why not
//	AWS             yes           HEAD on the bucket at the group region's endpoint. S3 serves
//	                              it there only for a bucket of that region and names the
//	                              bucket's region in x-amz-bucket-region otherwise (301, and
//	                              also 403 when the caller may not list the bucket).
//	                              GetBucketLocation is not used: minio-go answers it with the
//	                              client's own region without asking whenever one is set, and
//	                              the AWS client always has one (RegionRequired).
//	AZURE           no            A container has no region of its own; its storage account's
//	                              location is an Azure Resource Manager property the Blob API
//	                              does not expose to the account key.
//	GCP             no            "?location" answers a GCS location, which is a multi- or
//	                              dual-region (US, ASIA, NAM4) as often as a region, in another
//	                              spelling than the group's region. Not verified on an account.
//	IBM             no            The location constraint carries the storage class
//	                              (us-south-standard), not the region alone. Not verified.
//	ALIBABA,        no            Each has its own location naming through its S3-compatible
//	TENCENT, NCP,                 API, not verified against the region names cb-spider uses for
//	NHN                           the group.
//	KT, OPENSTACK   no            One endpoint for the whole account (s3ConnRule): there is no
//	                              region to compare with.
var bucketRegionProviders = map[string]bool{
	"AWS": true,
}

// bucketRegionWorkers bounds the lookups run at once for one listing, so an
// account with many buckets does not open a connection per bucket.
const bucketRegionWorkers = 8

// bucketRegionTimeout bounds all the lookups of one call. It stays under
// driverCallTimeout so that lookups which run out of time come back as errors
// of their own buckets instead of failing the whole call.
var bucketRegionTimeout = 45 * time.Second

// S3BucketRegion is what the CSP answered about where one bucket lives.
type S3BucketRegion struct {
	Region string
	Err    error
}

// GetS3BucketRegions reads the region of each named bucket. It returns nil,
// without asking the CSP, for a provider whose bucket region cannot be read;
// otherwise every name has an entry, with Err set when its lookup failed.
func GetS3BucketRegions(c Conn, bucketNames []string) (map[string]S3BucketRegion, error) {
	return withTimeout(driverCallTimeout, func() (map[string]S3BucketRegion, error) {
		return getS3BucketRegions(c, bucketNames)
	})
}

func getS3BucketRegions(c Conn, bucketNames []string) (map[string]S3BucketRegion, error) {
	info, err := s3ConnRuleFor(c)
	if err != nil {
		return nil, err
	}
	if !bucketRegionProviders[info.ProviderName] {
		return nil, nil
	}
	if len(bucketNames) == 0 {
		return map[string]S3BucketRegion{}, nil
	}

	client, err := newS3Client(info)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), bucketRegionTimeout)
	defer cancel()

	return s3BucketRegions(ctx, client, info.Region, bucketNames, bucketRegionWorkers), nil
}

// s3BucketRegions runs s3BucketRegion for every name on at most workers
// goroutines.
func s3BucketRegions(ctx context.Context, client *minio.Client, endpointRegion string, bucketNames []string, workers int) map[string]S3BucketRegion {
	if workers < 1 {
		workers = 1
	}
	if workers > len(bucketNames) {
		workers = len(bucketNames)
	}

	out := make(map[string]S3BucketRegion, len(bucketNames))
	var mu sync.Mutex
	names := make(chan string)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for name := range names {
				region, err := s3BucketRegion(ctx, client, endpointRegion, name)
				mu.Lock()
				out[name] = S3BucketRegion{Region: region, Err: err}
				mu.Unlock()
			}
		}()
	}
	for _, name := range bucketNames {
		names <- name
	}
	close(names)
	wg.Wait()

	return out
}

// s3BucketRegion asks the regional endpoint of client for the bucket. An
// answer that names the bucket's region wins; a plain success means the
// endpoint serves the bucket, so it lives in endpointRegion.
func s3BucketRegion(ctx context.Context, client *minio.Client, endpointRegion string, bucketName string) (string, error) {
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		if region := minio.ToErrorResponse(err).Region; region != "" {
			return region, nil
		}
		return "", fmt.Errorf("failed to read the region of bucket '%s': %w", bucketName, err)
	}
	if !exists {
		return "", errors.New("failed to read the region of bucket '" + bucketName + "': the bucket does not exist")
	}

	return endpointRegion, nil
}

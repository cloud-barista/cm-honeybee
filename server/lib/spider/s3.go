package spider

type bucketJSON struct {
	IId          BucketIID `json:"IId"`
	CreationDate string    `json:"CreationDate"`
}

type bucketsJSON struct {
	Bucket []bucketJSON `json:"Bucket"`
}

type listAllMyBucketsResultJSON struct {
	Buckets bucketsJSON `json:"Buckets"`
}

type bucketLocationJSON struct {
	LocationConstraint string `json:"LocationConstraint"`
}

// ListS3Buckets returns all S3 buckets reachable through the given connection.
func ListS3Buckets(connectionName string) ([]S3BucketInfo, error) {
	if err := mustNonEmpty("ConnectionName", connectionName); err != nil {
		return nil, err
	}
	var raw listAllMyBucketsResultJSON
	if err := do("GET", "/s3?ConnectionName="+encodePath(connectionName), nil, &raw); err != nil {
		return nil, err
	}
	out := make([]S3BucketInfo, 0, len(raw.Buckets.Bucket))
	for _, b := range raw.Buckets.Bucket {
		name := b.IId.NameId
		if name == "" {
			name = b.IId.SystemId
		}
		out = append(out, S3BucketInfo{
			Name:         name,
			CreationDate: b.CreationDate,
		})
	}
	return out, nil
}

// GetS3BucketLocation returns the bucket region (LocationConstraint) — used as
// a lightweight existence check + region lookup.
func GetS3BucketLocation(connectionName, bucketName string) (*S3BucketInfo, error) {
	if err := mustNonEmpty("ConnectionName", connectionName); err != nil {
		return nil, err
	}
	if err := mustNonEmpty("BucketName", bucketName); err != nil {
		return nil, err
	}
	var loc bucketLocationJSON
	path := "/s3/" + encodePath(bucketName) + "?location&ConnectionName=" + encodePath(connectionName)
	if err := do("GET", path, nil, &loc); err != nil {
		return nil, err
	}
	return &S3BucketInfo{
		Name:   bucketName,
		Region: loc.LocationConstraint,
	}, nil
}

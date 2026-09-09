package spider

// /alls3info response. Unlike /s3 — which only reads cb-spider's own
// s3bucket_iid_infos table — this endpoint queries the CSP directly and reports
// the CSP-side buckets alongside spider's registered metadata.
type s3BucketWithIID struct {
	NameId       string `json:"NameId"`
	SystemId     string `json:"SystemId"`
	CreationDate string `json:"CreationDate"`
}

type allS3ListInfo struct {
	MappedInfoList  []s3BucketWithIID `json:"MappedInfoList"`
	OnlySpiderList  []s3BucketWithIID `json:"OnlySpiderList"`
	OnlyCSPInfoList []s3BucketWithIID `json:"OnlyCSPInfoList"`
}

type allS3BucketInfoJSON struct {
	AllListInfo allS3ListInfo `json:"AllListInfo"`
}

type bucketLocationJSON struct {
	LocationConstraint string `json:"LocationConstraint"`
}

// ListS3Buckets returns all S3 buckets reachable through the given connection.
//
// It uses /alls3info rather than /s3: /s3 lists only buckets registered in
// cb-spider's own metadata table, which is always empty for the per-call
// temporary connections honeybee creates (see withSpiderConnection), so it
// would never return anything. /alls3info contacts the CSP itself, matching
// how VPCs and security groups are already collected (/allvpcinfo,
// /allsecuritygroupinfo).
func ListS3Buckets(connectionName string) ([]S3BucketInfo, error) {
	if err := mustNonEmpty("ConnectionName", connectionName); err != nil {
		return nil, err
	}
	var raw allS3BucketInfoJSON
	if err := do("GET", "/alls3info?ConnectionName="+encodePath(connectionName), nil, &raw); err != nil {
		return nil, err
	}

	// Buckets that exist on the CSP = Mapped + OnlyCSP. OnlySpiderList holds
	// stale metadata for buckets already deleted on the CSP, so it is dropped.
	// honeybee registers nothing with spider, so in practice everything arrives
	// in OnlyCSPInfoList.
	all := make([]s3BucketWithIID, 0,
		len(raw.AllListInfo.MappedInfoList)+len(raw.AllListInfo.OnlyCSPInfoList))
	all = append(all, raw.AllListInfo.MappedInfoList...)
	all = append(all, raw.AllListInfo.OnlyCSPInfoList...)

	out := make([]S3BucketInfo, 0, len(all))
	for _, b := range all {
		// SystemId is the real bucket name on the CSP; NameId is spider's alias
		// and differs on Tencent, where an AppId suffix is appended.
		name := b.SystemId
		if name == "" {
			name = b.NameId
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

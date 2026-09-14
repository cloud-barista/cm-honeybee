package spider

type allClusterInfoResp struct {
	AllListInfo struct {
		MappedInfoList  []ClusterInfo `json:"MappedInfoList"`
		OnlyCSPInfoList []ClusterInfo `json:"OnlyCSPInfoList"`
	} `json:"AllListInfo"`
}

// ListAllClusterInfo returns every Kubernetes cluster the CSP actually has -
// both the ones cb-spider manages and the ones it does not.
//
// GET /cluster walks cb-spider's own meta-DB and returns an empty list without
// ever asking the CSP when that DB holds nothing for the connection. Every
// honeybee request registers a fresh connection, so its meta-DB is always
// empty and a migration source's clusters never appear. Measured against a
// live cb-spider: through a newly registered connection /cluster answered
// {"cluster":[]} while /allclusterinfo listed the cluster that was running.
//
// A cluster cb-spider did not create arrives with an empty NameId, so SystemId
// is the only identifier it carries.
func ListAllClusterInfo(connectionName string) ([]ClusterInfo, error) {
	if err := mustNonEmpty("ConnectionName", connectionName); err != nil {
		return nil, err
	}
	var r allClusterInfoResp
	if err := do("GET", "/allclusterinfo?ConnectionName="+encodePath(connectionName), nil, &r); err != nil {
		return nil, err
	}
	return append(append([]ClusterInfo{}, r.AllListInfo.MappedInfoList...),
		r.AllListInfo.OnlyCSPInfoList...), nil
}

// GetCluster fetches a single Kubernetes cluster by name.
func GetCluster(connectionName, clusterName string) (*ClusterInfo, error) {
	if err := mustNonEmpty("ConnectionName", connectionName); err != nil {
		return nil, err
	}
	if err := mustNonEmpty("Name", clusterName); err != nil {
		return nil, err
	}
	var out ClusterInfo
	if err := do("GET", "/cluster/"+encodePath(clusterName)+"?ConnectionName="+encodePath(connectionName), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

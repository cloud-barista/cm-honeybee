package spider

type clusterListResp struct {
	Cluster []ClusterInfo `json:"cluster"`
}

// ListCluster returns all Kubernetes clusters reachable through the given connection.
func ListCluster(connectionName string) ([]ClusterInfo, error) {
	if err := mustNonEmpty("ConnectionName", connectionName); err != nil {
		return nil, err
	}
	var out clusterListResp
	if err := do("GET", "/cluster?ConnectionName="+encodePath(connectionName), nil, &out); err != nil {
		return nil, err
	}
	return out.Cluster, nil
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

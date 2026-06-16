package kubernetes

import (
	"encoding/json"
	"testing"
)

// TestManualCollect runs the live K8s collection against a real cluster so you
// can see the result with your own eyes. Run it explicitly with KUBECONFIG
// pointing at the cluster:
//
//	KUBECONFIG=$HOME/.kube/config go test ./driver/kubernetes/ -run TestManualCollect -v
func TestManualCollect(t *testing.T) {
	KubeConfigPath = getKubeConfigPath()
	t.Logf("KubeConfigPath = %s", KubeConfigPath)

	cluster, err := GetClusterInfo()
	if err != nil {
		t.Fatalf("GetClusterInfo: %v", err)
	}
	clusterJSON, _ := json.MarshalIndent(cluster, "", "  ")
	t.Logf("Cluster:\n%s", clusterJSON)

	nodeCount, nodes, err := GetNodeInfo()
	if err != nil {
		t.Fatalf("GetNodeInfo: %v", err)
	}
	t.Logf("NodeCount: total=%d control-plane=%d worker=%d",
		nodeCount.Total, nodeCount.ControlPlane, nodeCount.Worker)
	for _, n := range nodes {
		t.Logf("Node name=%v type=%s", n.Name, n.Type)
	}
}

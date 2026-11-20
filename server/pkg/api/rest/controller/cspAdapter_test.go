package controller

import (
	"errors"
	"strings"
	"testing"

	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// TestMatchCluster covers the identifier shapes cb-spider reports. The
// CSP-only case is the one that matters: a cluster cb-spider did not create
// arrives with an empty NameId, measured against a live cb-spider through a
// freshly registered connection.
func TestMatchCluster(t *testing.T) {
	cspOnly := spider.ClusterInfo{IId: spider.IID{NameId: "", SystemId: "tbho74o84d55a49rjo4i"}}
	managed := spider.ClusterInfo{IId: spider.IID{NameId: "my-cluster", SystemId: "eks-abc123"}}
	listing := []spider.ClusterInfo{cspOnly, managed}

	tests := []struct {
		name       string
		resourceID string
		wantSystem string
	}{
		{"CSP-only cluster by system id", "tbho74o84d55a49rjo4i", "tbho74o84d55a49rjo4i"},
		{"managed cluster by system id", "eks-abc123", "eks-abc123"},
		{"managed cluster by name id", "my-cluster", "eks-abc123"},
		{"unknown id", "nope", ""},
		// An empty id must not match the empty NameId every CSP-only cluster has.
		{"empty id", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchCluster(listing, tc.resourceID)

			if tc.wantSystem == "" {
				if got != nil {
					t.Fatalf("matched %+v, want no match", got.IId)
				}

				return
			}

			if got == nil {
				t.Fatalf("no match, want system id %q", tc.wantSystem)
			}

			if got.IId.SystemId != tc.wantSystem {
				t.Errorf("system id = %q, want %q", got.IId.SystemId, tc.wantSystem)
			}
		})
	}
}

var ncpConn = spider.Conn{Provider: "NCP", Region: "KR", Zone: "KR-1"}

const ncpClusterID = "4d7c2c9e-1b2a-4f3e-9a8b-0c1d2e3f4a5b"

// stubClusterCSP stands in for the cluster calls of the CSP. The listing
// answers listed, or fails with listErr when set; GetCluster answers got, or
// fails with getErr when set. The calls made are counted.
func stubClusterCSP(t *testing.T, listed []spider.ClusterInfo, listErr error, got *spider.ClusterInfo, getErr error) map[string]int {
	t.Helper()

	calls := map[string]int{}
	origGet, origList := getCSPCluster, listCSPCluster
	t.Cleanup(func() { getCSPCluster, listCSPCluster = origGet, origList })

	listCSPCluster = func(spider.Conn) ([]spider.ClusterInfo, error) {
		calls["list"]++
		if listErr != nil {
			return nil, listErr
		}
		return listed, nil
	}
	getCSPCluster = func(_ spider.Conn, id string) (*spider.ClusterInfo, error) {
		calls["get"]++
		calls["get:"+id]++
		if getErr != nil {
			return nil, getErr
		}
		return got, nil
	}

	return calls
}

// A cluster the listing does not hold is asked for by GetCluster, since the
// ncp driver lists no cluster at all, and is found only when GetCluster
// answers with that very cluster.
func TestFindClusterByID(t *testing.T) {
	ncpCluster := &spider.ClusterInfo{IId: spider.IID{NameId: "nks-1", SystemId: ncpClusterID}, Version: "1.30"}

	t.Run("listed", func(t *testing.T) {
		calls := stubClusterCSP(t, []spider.ClusterInfo{*ncpCluster}, nil, nil, errors.New("must not be called"))
		cl, err := findClusterByID(ncpConn, ncpClusterID)
		if err != nil || cl.IId.SystemId != ncpClusterID {
			t.Fatalf("got %+v, %v; want %s", cl, err, ncpClusterID)
		}
		if calls["get"] != 0 {
			t.Errorf("get calls = %d, want 0", calls["get"])
		}
	})

	t.Run("empty listing, found by GetCluster", func(t *testing.T) {
		for _, id := range []string{ncpClusterID, "nks-1"} {
			calls := stubClusterCSP(t, nil, nil, ncpCluster, nil)
			cl, err := findClusterByID(ncpConn, id)
			if err != nil || cl.IId.SystemId != ncpClusterID || cl.Version != "1.30" {
				t.Fatalf("%s: got %+v, %v; want %s", id, cl, err, ncpClusterID)
			}
			if calls["get:"+id] != 1 {
				t.Errorf("%s: calls = %v, want one GetCluster of the id", id, calls)
			}
		}
	})

	t.Run("GetCluster answers another cluster", func(t *testing.T) {
		other := &spider.ClusterInfo{IId: spider.IID{NameId: "nks-2", SystemId: "other-cluster"}}
		stubClusterCSP(t, nil, nil, other, nil)
		cl, err := findClusterByID(ncpConn, ncpClusterID)
		if !errors.Is(err, ErrCSPResourceNotFound) {
			t.Fatalf("got %+v, %v; want ErrCSPResourceNotFound", cl, err)
		}
		want := "cluster '" + ncpClusterID + "' was not found in ncp region 'KR' zone 'KR-1'"
		if err.Error() != want {
			t.Errorf("err = %q, want %q", err, want)
		}
	})

	t.Run("GetCluster answers nil", func(t *testing.T) {
		stubClusterCSP(t, nil, nil, nil, nil)
		if _, err := findClusterByID(ncpConn, ncpClusterID); !errors.Is(err, ErrCSPResourceNotFound) {
			t.Fatalf("err = %v, want ErrCSPResourceNotFound", err)
		}
	})

	t.Run("GetCluster fails", func(t *testing.T) {
		stubClusterCSP(t, nil, nil, nil, errors.New("cluster does not exist"))
		_, err := findClusterByID(ncpConn, ncpClusterID)
		if !errors.Is(err, ErrCSPResourceNotFound) {
			t.Fatalf("err = %v, want ErrCSPResourceNotFound", err)
		}
		want := "cluster '" + ncpClusterID + "' was not found in ncp region 'KR' zone 'KR-1' (cluster does not exist)"
		if err.Error() != want {
			t.Errorf("err = %q, want %q", err, want)
		}
	})

	t.Run("listing fails", func(t *testing.T) {
		calls := stubClusterCSP(t, nil, errors.New("RequestTimeout"), ncpCluster, nil)
		_, err := findClusterByID(ncpConn, ncpClusterID)
		if err == nil || errors.Is(err, ErrCSPResourceNotFound) || err.Error() != "RequestTimeout" {
			t.Fatalf("err = %v, want the listing error unmarked", err)
		}
		if calls["get"] != 0 {
			t.Errorf("get calls = %d, want 0", calls["get"])
		}
	})
}

// The k8s collection reads the cluster as the registration check does, so an
// ncp cluster that only GetCluster can read is stored.
func TestCollectCSPResource_K8sByGetCluster(t *testing.T) {
	openTestDB(t)
	stubClusterCSP(t, nil, nil, &spider.ClusterInfo{IId: spider.IID{SystemId: ncpClusterID}, Version: "1.30"}, nil)

	ci := &model.ConnectionInfo{ID: "ncp-k8s", ResourceType: "k8s", ResourceID: ncpClusterID}
	if err := collectCSPResource(ncpConn, nil, ci, importInfraOptions{}); err != nil {
		t.Fatalf("collect: %v", err)
	}

	k, err := dao.SavedKubernetesInfoGet("ncp-k8s")
	if err != nil {
		t.Fatal(err)
	}
	if k.Status != model.ConnectionInfoStatusSuccess || !strings.Contains(k.KubernetesData, ncpClusterID) ||
		!strings.Contains(k.KubernetesData, "1.30") {
		t.Errorf("k8s record = %#v, want the cluster stored", k)
	}
}

func TestMatchBucket(t *testing.T) {
	listing := []spider.S3BucketInfo{
		{Name: "logs-archive"},
		{Name: "app-assets"},
	}

	tests := []struct {
		name   string
		bucket string
		want   string
	}{
		{"first", "logs-archive", "logs-archive"},
		{"second", "app-assets", "app-assets"},
		{"unknown", "nope", ""},
		{"empty", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchBucket(listing, tc.bucket)

			if tc.want == "" {
				if got != nil {
					t.Fatalf("matched %q, want no match", got.Name)
				}

				return
			}

			if got == nil || got.Name != tc.want {
				t.Fatalf("got %v, want %q", got, tc.want)
			}
		})
	}
}

func TestMatchRDBMS(t *testing.T) {
	cspOnly := spider.RDBMSInfo{IId: spider.IID{NameId: "", SystemId: "projects/p/instances/sql-1"}}
	managed := spider.RDBMSInfo{IId: spider.IID{NameId: "billing-db", SystemId: "db-abc123"}}
	listing := []spider.RDBMSInfo{cspOnly, managed}

	tests := []struct {
		name       string
		resourceID string
		wantSystem string
	}{
		{"CSP-only instance by system id", "projects/p/instances/sql-1", "projects/p/instances/sql-1"},
		{"managed instance by system id", "db-abc123", "db-abc123"},
		{"managed instance by name id", "billing-db", "db-abc123"},
		{"unknown id", "nope", ""},
		// An empty id must not match the empty NameId of a CSP-only instance.
		{"empty id", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchRDBMS(listing, tc.resourceID)

			if tc.wantSystem == "" {
				if got != nil {
					t.Fatalf("matched %+v, want no match", got.IId)
				}

				return
			}

			if got == nil {
				t.Fatalf("no match, want system id %q", tc.wantSystem)
			}

			if got.IId.SystemId != tc.wantSystem {
				t.Errorf("system id = %q, want %q", got.IId.SystemId, tc.wantSystem)
			}
		})
	}
}

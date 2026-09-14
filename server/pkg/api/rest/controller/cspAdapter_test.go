package controller

import (
	"testing"

	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
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

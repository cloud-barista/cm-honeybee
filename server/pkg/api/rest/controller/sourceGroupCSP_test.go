package controller

import "testing"

func TestSplitRegionZone(t *testing.T) {
	tests := []struct {
		name         string
		regionName   string
		zoneOverride string
		wantRegion   string
		wantZone     string
		wantErr      bool
	}{
		{"region only", "koreacentral", "", "koreacentral", "", false},
		{"zone in region_name", "koreacentral/1", "", "koreacentral", "1", false},
		{"override wins", "koreacentral/1", "2", "koreacentral", "2", false},
		{"override without embedded zone", " ap-northeast-2 ", " ap-northeast-2a ", "ap-northeast-2", "ap-northeast-2a", false},
		{"empty", "", "", "", "", true},
		{"zone without region", "/1", "", "", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			region, zone, err := splitRegionZone(tc.regionName, tc.zoneOverride)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("got %q/%q, want an error", region, zone)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if region != tc.wantRegion || zone != tc.wantZone {
				t.Errorf("got %q/%q, want %q/%q", region, zone, tc.wantRegion, tc.wantZone)
			}
		})
	}
}

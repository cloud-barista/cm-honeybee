package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// ncpMainKV is an ncp credential with the required keys only.
func ncpMainKV() []model.KeyValue {
	return []model.KeyValue{
		{Key: "ncloud_access_key", Value: fakeMainValue},
		{Key: "ncloud_secret_key", Value: fakeMainValue},
	}
}

// ncpRegions is the shape of ncp's region list: upper-case names, which its
// SDK puts in the API path as they are.
var ncpRegions = []spider.RegionZoneInfo{
	{Name: "KR", ZoneList: []spider.ZoneInfo{{Name: "KR-1"}, {Name: "KR-2"}}},
	{Name: "JPN", ZoneList: []spider.ZoneInfo{{Name: "JPN-1"}}},
}

// Registering a csp group stores region_name as the CSP's region list spells
// it, the zone of "<region>/<zone>" too. What the list does not name, or a
// list that cannot be read, leaves that part as given (trimmed): the group is
// not refused for it.
func TestValidateAndCanonicalizeCSP_Region(t *testing.T) {
	useSpiderRoot(t)

	tests := []struct {
		name    string
		in      string
		listErr error
		want    string
	}{
		{"region", "kr", nil, "KR"},
		{"already as listed", "KR", nil, "KR"},
		{"spaces", "  kr  ", nil, "KR"},
		{"region and zone", "kr/kr-1", nil, "KR/KR-1"},
		{"spaces around the zone", " kr / kr-2 ", nil, "KR/KR-2"},
		{"zone not listed", "kr/kr-9", nil, "KR/kr-9"},
		{"zone of another region", "kr/jpn-1", nil, "KR/jpn-1"},
		{"region not listed", "us", nil, "us"},
		{"region not listed with a zone", " us / us-1 ", nil, "us/us-1"},
		{"list fails", " kr / kr-1 ", errors.New("AuthFailure: key " + fakeMainValue + " was not accepted"), "kr/kr-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stubRegionZone(t, ncpRegions, tt.listErr)
			sg := &model.SourceGroup{ProviderName: "ncp", RegionName: tt.in}
			if err := validateAndCanonicalizeCSP(sg, ncpMainKV()); err != nil {
				t.Fatal(err)
			}
			if sg.RegionName != tt.want {
				t.Errorf("region_name = %q, want %q", sg.RegionName, tt.want)
			}
			creds := map[string]string{}
			for _, kv := range *got {
				creds[kv.Key] = kv.Value
			}
			want := map[string]string{"ncloud_access_key": fakeMainValue, "ncloud_secret_key": fakeMainValue}
			if !reflect.DeepEqual(creds, want) {
				t.Errorf("credential handed to the region query = %v, want %v", creds, want)
			}
		})
	}
}

// A blank region_name is refused before the CSP is asked.
func TestValidateAndCanonicalizeCSP_EmptyRegion(t *testing.T) {
	useSpiderRoot(t)
	called := stubRegionZone(t, ncpRegions, nil)

	sg := &model.SourceGroup{ProviderName: "ncp", RegionName: "  "}
	err := validateAndCanonicalizeCSP(sg, ncpMainKV())
	if err == nil || err.Error() != "region_name is empty" {
		t.Errorf("err = %v, want region_name is empty", err)
	}
	if *called != nil {
		t.Errorf("the CSP was queried")
	}
}

// Changing only the region of a csp group matches it to the CSP's region list
// with the group's stored credential, as registering does. When the stored
// credential cannot be read the region is kept as given.
func TestUpdateSourceGroup_RegionOnly(t *testing.T) {
	useSpiderRoot(t)
	openTestDB(t)

	stored := toSpiderKV(ncpMainKV())
	putRegion := func(t *testing.T, sgID, region string) model.SourceGroup {
		t.Helper()
		rec := jsonConnRequest(t, http.MethodPut, sgID, "", model.UpdateSourceGroupReq{RegionName: region}, UpdateSourceGroup)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var res model.SourceGroup
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatal(err)
		}
		saved, err := dao.SourceGroupGet(sgID)
		if err != nil {
			t.Fatal(err)
		}
		if saved.RegionName != res.RegionName || saved.ProviderName != res.ProviderName {
			t.Errorf("saved %q %q, answered %q %q", saved.ProviderName, saved.RegionName, res.ProviderName, res.RegionName)
		}
		return res
	}

	t.Run("matched with the stored credential", func(t *testing.T) {
		group := &model.SourceGroup{ID: "sg-ncp-region", Name: "sg-ncp-region", Type: common.SourceGroupTypeCSP,
			ProviderName: "ncp", RegionName: "kr"}
		createGroup(t, group, nil)
		orig := groupCredential
		t.Cleanup(func() { groupCredential = orig })
		var askedFor string
		groupCredential = func(sg *model.SourceGroup) (string, []spider.KeyValue, error) {
			askedFor = sg.ID
			return "NCP", stored, nil
		}
		got := stubRegionZone(t, ncpRegions, nil)

		res := putRegion(t, group.ID, " kr / kr-2 ")
		if res.RegionName != "KR/KR-2" || res.ProviderName != "ncp" {
			t.Errorf("provider_name, region_name = %q, %q, want ncp, KR/KR-2", res.ProviderName, res.RegionName)
		}
		if askedFor != group.ID {
			t.Errorf("credential read for %q, want %q", askedFor, group.ID)
		}
		if !reflect.DeepEqual(*got, stored) {
			t.Errorf("credential handed to the region query = %v, want the stored one", *got)
		}
	})

	t.Run("stored credential unreadable", func(t *testing.T) {
		group := &model.SourceGroup{ID: "sg-ncp-nocred", Name: "sg-ncp-nocred", Type: common.SourceGroupTypeCSP,
			ProviderName: "ncp", RegionName: "KR"}
		createGroup(t, group, nil)
		called := stubRegionZone(t, ncpRegions, nil)

		res := putRegion(t, group.ID, " kr / kr-2 ")
		if res.RegionName != "kr/kr-2" {
			t.Errorf("region_name = %q, want kr/kr-2", res.RegionName)
		}
		if *called != nil {
			t.Errorf("the CSP was queried")
		}
	})

	t.Run("blank region", func(t *testing.T) {
		group := &model.SourceGroup{ID: "sg-ncp-blank", Name: "sg-ncp-blank", Type: common.SourceGroupTypeCSP,
			ProviderName: "ncp", RegionName: "KR"}
		createGroup(t, group, nil)
		called := stubRegionZone(t, ncpRegions, nil)

		rec := jsonConnRequest(t, http.MethodPut, group.ID, "", model.UpdateSourceGroupReq{RegionName: "   "}, UpdateSourceGroup)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "region_name is empty") {
			t.Errorf("status = %d, body = %s, want 400 region_name is empty", rec.Code, rec.Body.String())
		}
		if *called != nil {
			t.Errorf("the CSP was queried")
		}
	})
}

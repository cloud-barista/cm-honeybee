package controller

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// Changing only the region of a csp group keeps provider_name in the lower
// case registration stores it in, rather than cb-spider's upper-case name.
func TestUpdateSourceGroup_RegionOnlyKeepsProviderLowerCase(t *testing.T) {
	useSpiderRoot(t)
	openTestDB(t)
	stubRegionZone(t, nil, nil)

	group := &model.SourceGroup{ID: "sg-ncp-provider", Name: "sg-ncp-provider", Type: common.SourceGroupTypeCSP,
		ProviderName: "ncp", RegionName: "KR"}
	createGroup(t, group, nil)

	rec := jsonConnRequest(t, http.MethodPut, group.ID, "", model.UpdateSourceGroupReq{RegionName: "KR"}, UpdateSourceGroup)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var res model.SourceGroup
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.ProviderName != "ncp" {
		t.Errorf("answered provider_name = %q, want ncp", res.ProviderName)
	}
	saved, err := dao.SourceGroupGet(group.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.ProviderName != "ncp" {
		t.Errorf("saved provider_name = %q, want ncp", saved.ProviderName)
	}
}

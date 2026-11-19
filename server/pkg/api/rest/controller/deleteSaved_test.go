package controller

import (
	"net/http"
	"testing"

	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// savedImportTables are the tables an import stores a connection's result in,
// keyed by the connection ID.
var savedImportTables = map[string]any{
	"infra":      &model.SavedInfraInfo{},
	"software":   &model.SavedSoftwareInfo{},
	"kubernetes": &model.SavedKubernetesInfo{},
	"helm":       &model.SavedHelmInfo{},
	"data":       &model.SavedDataInfo{},
	"rdbms":      &model.SavedRDBMSInfo{},
	"nlb":        &model.SavedNLBInfo{},
}

func saveEveryImport(t *testing.T, connID string) {
	t.Helper()

	for _, rec := range []any{
		&model.SavedInfraInfo{ConnectionID: connID, Status: "success"},
		&model.SavedSoftwareInfo{ConnectionID: connID, Status: "success"},
		&model.SavedKubernetesInfo{ConnectionID: connID, Status: "success"},
		&model.SavedHelmInfo{ConnectionID: connID, Status: "success"},
		&model.SavedDataInfo{ConnectionID: connID, Status: "success"},
		&model.SavedRDBMSInfo{ConnectionID: connID, Status: "success"},
		&model.SavedNLBInfo{ConnectionID: connID, NLBData: "{}", Status: "success"},
	} {
		if err := db.DB.Create(rec).Error; err != nil {
			t.Fatal(err)
		}
	}
}

// savedImportCounts counts the rows each import table holds for connID.
func savedImportCounts(t *testing.T, connID string) map[string]int64 {
	t.Helper()

	counts := map[string]int64{}
	for name, table := range savedImportTables {
		var n int64
		if err := db.DB.Model(table).Where("connection_id = ?", connID).Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		counts[name] = n
	}
	return counts
}

func assertSavedImports(t *testing.T, connID string, want int64) {
	t.Helper()

	for name, n := range savedImportCounts(t, connID) {
		if n != want {
			t.Errorf("%s: %s rows = %d, want %d", connID, name, n, want)
		}
	}
}

// Deleting a connection removes what its imports stored, and leaves the other
// connections' records alone.
func TestDeleteConnectionInfo_RemovesSavedImports(t *testing.T) {
	openTestDB(t)
	createGroup(t, nlbGroup, nlbGroupConns())
	saveEveryImport(t, "nlb")
	saveEveryImport(t, "vm-web")

	rec := connRequest(t, http.MethodDelete, nlbGroup.ID, "nlb", DeleteConnectionInfo)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	assertSavedImports(t, "nlb", 0)
	assertSavedImports(t, "vm-web", 1)
}

// Deleting a source group removes what the imports stored for every one of
// its connections, and leaves another group's records alone.
func TestDoDeleteSourceGroup_RemovesSavedImports(t *testing.T) {
	openTestDB(t)
	createGroup(t, nlbGroup, nlbGroupConns())
	createGroup(t, onpremGroup, []model.ConnectionInfo{testConn("onprem-1", onpremGroup, "", "192.0.2.10")})
	for _, id := range []string{"vm-web", "nlb", "bucket", "onprem-1"} {
		saveEveryImport(t, id)
	}

	if err := doDeleteSourceGroup(nlbGroup.ID); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"vm-web", "nlb", "bucket"} {
		assertSavedImports(t, id, 0)
	}
	assertSavedImports(t, "onprem-1", 1)
}

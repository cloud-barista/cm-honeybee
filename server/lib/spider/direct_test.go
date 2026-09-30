package spider

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
)

// TestMain points $CBSPIDER_ROOT at a temp dir holding cb-spider's yaml files
// from the module cache. info-store init() has already run by now against the
// spiderroot default; everything else reads $CBSPIDER_ROOT lazily.
func TestMain(m *testing.M) {
	root, err := setupSpiderRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "setup CBSPIDER_ROOT:", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(root)
	os.Exit(code)
}

func setupSpiderRoot() (string, error) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/cloud-barista/cb-spider").Output()
	if err != nil {
		return "", fmt.Errorf("go list cb-spider: %w", err)
	}
	src := strings.TrimSpace(string(out))

	root, err := os.MkdirTemp("", "honeybee-spider-test-")
	if err != nil {
		return "", err
	}
	for _, rel := range []string{
		"cloud-driver-libs/cloudos.yaml",
		"cloud-driver-libs/cloudos_meta.yaml",
		"cloud-driver-libs/region/aws_region_meta.yaml",
		"conf/calllog_conf.yaml",
		"conf/log_conf.yaml",
	} {
		b, err := os.ReadFile(filepath.Join(src, rel))
		if err != nil {
			return root, err
		}
		dst := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return root, err
		}
		if err := os.WriteFile(dst, b, 0644); err != nil {
			return root, err
		}
	}
	return root, os.Setenv("CBSPIDER_ROOT", root)
}

func TestMapCredentialKeysAWS(t *testing.T) {
	meta, err := cim.GetCloudOSMetaInfo("AWS")
	if err != nil {
		t.Fatal(err)
	}
	got := mapCredentialKeys(meta, []KeyValue{
		{Key: "aws_access_key_id", Value: "id"},
		{Key: "aws_secret_access_key", Value: "secret"},
		{Key: "unknown_key", Value: "kept"},
	})
	want := []KeyValue{
		{Key: "ClientId", Value: "id"},
		{Key: "ClientSecret", Value: "secret"},
		{Key: "unknown_key", Value: "kept"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestConnectionInfoAWS(t *testing.T) {
	info, err := connectionInfo(Conn{
		Provider: "aws",
		Region:   "ap-northeast-2",
		Zone:     "ap-northeast-2a",
		Credential: []KeyValue{
			{Key: "aws_access_key_id", Value: "id"},
			{Key: "aws_secret_access_key", Value: "secret"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ci := info.CredentialInfo
	if ci.ClientId != "id" || ci.ClientSecret != "secret" {
		t.Errorf("ClientId/ClientSecret = %q/%q", ci.ClientId, ci.ClientSecret)
	}
	if ci.StsToken != "Not set" {
		t.Errorf("StsToken = %q, want \"Not set\"", ci.StsToken)
	}
	if ci.ConnectionName != driverConnectionName {
		t.Errorf("ConnectionName = %q", ci.ConnectionName)
	}
	if info.RegionInfo.Region != "ap-northeast-2" || info.RegionInfo.Zone != "ap-northeast-2a" {
		t.Errorf("RegionInfo = %+v", info.RegionInfo)
	}
}

func TestCredentialInfoRDSKeys(t *testing.T) {
	ci := credentialInfo([]KeyValue{
		{Key: "User Access Key", Value: "ua"},
		{Key: "Secret Access Key", Value: "sa"},
		{Key: "mysqlAppKey", Value: "my"},
		{Key: "mariadbAppKey", Value: "ma"},
	})
	if ci.RDSUserAccessKey != "ua" || ci.RDSSecretAccessKey != "sa" ||
		ci.RDSMySQLAppKey != "my" || ci.RDSMariaDBAppKey != "ma" {
		t.Errorf("RDS keys = %q %q %q %q", ci.RDSUserAccessKey, ci.RDSSecretAccessKey,
			ci.RDSMySQLAppKey, ci.RDSMariaDBAppKey)
	}
}

func TestQueryConnectionInfoDefaultRegion(t *testing.T) {
	meta, err := cim.GetCloudOSMetaInfo("AWS")
	if err != nil {
		t.Fatal(err)
	}
	info := queryConnectionInfo(meta, []KeyValue{{Key: "aws_access_key_id", Value: "id"}})
	if info.RegionInfo.Region != "ap-northeast-2" || info.RegionInfo.Zone != "ap-northeast-2a" {
		t.Errorf("RegionInfo = %+v", info.RegionInfo)
	}
	if info.CredentialInfo.ClientId != "id" {
		t.Errorf("ClientId = %q", info.CredentialInfo.ClientId)
	}
}

func TestCloudDriver(t *testing.T) {
	for _, p := range []string{"AWS", "azure", "GCP", "ALIBABA", "OPENSTACK", "TENCENT",
		"IBM", "ORACLE", "NCP", "NHN", "KTCLASSIC", "kt"} {
		if drv, err := cloudDriver(p); err != nil || drv == nil {
			t.Errorf("%s: driver %v, err %v", p, drv, err)
		}
	}
	for _, p := range []string{"MOCK", "NOPE", ""} {
		if _, err := cloudDriver(p); err == nil {
			t.Errorf("%q: expected an error", p)
		}
	}
}

func TestUnknownProvider(t *testing.T) {
	if _, err := ListVM(Conn{Provider: "NOPE"}); err == nil {
		t.Error("ListVM: expected an error")
	}
	if _, err := ListRegionZone("NOPE", nil); err == nil {
		t.Error("ListRegionZone: expected an error")
	}
	if _, err := GetDriverCapability("NOPE"); err == nil {
		t.Error("GetDriverCapability: expected an error")
	}
}

func TestGetDriverCapabilityAWS(t *testing.T) {
	c, err := GetDriverCapability("aws")
	if err != nil {
		t.Fatal(err)
	}
	if !c.VMHandler || !c.NLBHandler || !c.ClusterHandler {
		t.Errorf("capability = %+v", *c)
	}
}

func TestCloudOS(t *testing.T) {
	list, err := ListCloudOS()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range list {
		if n == "AWS" {
			found = true
		}
	}
	if !found {
		t.Errorf("AWS not in %v", list)
	}

	meta, err := GetCloudOSMetaInfo("aws")
	if err != nil {
		t.Fatal(err)
	}
	if len(meta.CredentialCSP) != 2 || meta.CredentialCSP[0] != "aws_access_key_id" {
		t.Errorf("CredentialCSP = %v", meta.CredentialCSP)
	}
}

func TestUpdateRegionZoneDisplayNames(t *testing.T) {
	list := []*cres.RegionZoneInfo{{
		Name:     "ap-northeast-2",
		ZoneList: []cres.ZoneInfo{{Name: "ap-northeast-2a"}},
	}}
	got, err := updateRegionZoneDisplayNames("aws", list)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].DisplayName == "" || got[0].ZoneList[0].DisplayName == "" {
		t.Errorf("display names not filled: %+v", *got[0])
	}

	missing := []*cres.RegionZoneInfo{{Name: "x"}}
	if _, err := updateRegionZoneDisplayNames("no-such-csp", missing); err != nil {
		t.Errorf("missing meta file: %v", err)
	}
}

func TestConvertListEmpty(t *testing.T) {
	var in []*cres.VMInfo
	out, err := convertList[VMInfo](in)
	if err != nil {
		t.Fatal(err)
	}
	if out == nil || len(out) != 0 {
		t.Errorf("got %#v, want empty non-nil slice", out)
	}
}

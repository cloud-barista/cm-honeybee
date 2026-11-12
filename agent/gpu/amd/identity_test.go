package amd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
)

// Captured from "amd-smi list --json" on a Radeon Pro V710 MxGPU, ROCm 10.0.
func TestParseAMDSMIList(t *testing.T) {
	uuids, err := parseAMDSMIList([]byte(`[
    {
        "gpu": 0,
        "bdf": "0002:00:00.0",
        "uuid": "ea007461-0000-1000-801a-76e15b35f12d",
        "kfd_id": 39285,
        "node_id": 1,
        "partition_id": 0
    },
    {"gpu": 1, "bdf": "0003:00:00.0", "uuid": "N/A"}
]`))
	if err != nil {
		t.Fatalf("parseAMDSMIList: %v", err)
	}

	if len(uuids) != 1 {
		t.Fatalf("got %d entries, want 1: %v", len(uuids), uuids)
	}
	if got := uuids["0002:00:00.0"]; got != "ea007461-0000-1000-801a-76e15b35f12d" {
		t.Errorf("uuid = %q", got)
	}
}

func TestParseAMDSMIListWrapped(t *testing.T) {
	uuids, err := parseAMDSMIList([]byte(`{"gpu_data":[{"bdf":"0000:c1:00.0","uuid":"u-1"}]}`))
	if err != nil {
		t.Fatalf("parseAMDSMIList: %v", err)
	}
	if uuids["0000:c1:00.0"] != "u-1" {
		t.Errorf("uuids = %v", uuids)
	}
}

func TestParseAMDSMIListRejectsNonJSON(t *testing.T) {
	if _, err := parseAMDSMIList([]byte("ERROR")); err == nil {
		t.Fatal("want an error for output that is not JSON")
	}
}

func TestApplyIdentity(t *testing.T) {
	dir := t.TempDir()
	device := filepath.Join(dir, "0002:00:00.0")
	if err := os.MkdirAll(device, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(device, "vendor"), []byte("0x1002\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	saved := sysPCIDevicesDir
	sysPCIDevicesDir = dir
	t.Cleanup(func() { sysPCIDevicesDir = saved })

	gpus := []infra.AMD{
		// rocm-smi form: no vendor id, address as rocm-smi prints it.
		{DeviceAttribute: infra.AMDDeviceAttribute{Card: "card0", PCIBusID: "0002:00:00.0"}},
		// A card sysfs and amd-smi do not know keeps both fields empty.
		{DeviceAttribute: infra.AMDDeviceAttribute{Card: "card1", PCIBusID: "0003:00:00.0"}},
		// A card without an address is left alone.
		{DeviceAttribute: infra.AMDDeviceAttribute{Card: "card2"}},
		// A vendor id the reader already took is kept.
		{DeviceAttribute: infra.AMDDeviceAttribute{Card: "card3", PCIBusID: "00000002:00:00.0", VendorID: "0xabcd"}},
	}

	applyIdentity(gpus, map[string]string{"0002:00:00.0": "ea007461-0000-1000-801a-76e15b35f12d"})

	first := gpus[0].DeviceAttribute
	if first.VendorID != "0x1002" {
		t.Errorf("card0 vendor_id = %q, want 0x1002", first.VendorID)
	}
	if first.GPUUUID != "ea007461-0000-1000-801a-76e15b35f12d" {
		t.Errorf("card0 gpu_uuid = %q", first.GPUUUID)
	}

	for _, i := range []int{1, 2} {
		a := gpus[i].DeviceAttribute
		if a.VendorID != "" || a.GPUUUID != "" {
			t.Errorf("%s vendor_id = %q, gpu_uuid = %q, want both empty", a.Card, a.VendorID, a.GPUUUID)
		}
	}

	last := gpus[3].DeviceAttribute
	if last.VendorID != "0xabcd" {
		t.Errorf("card3 vendor_id = %q, want 0xabcd kept", last.VendorID)
	}
	if last.GPUUUID != "ea007461-0000-1000-801a-76e15b35f12d" {
		t.Errorf("card3 gpu_uuid = %q, want matched through the eight-digit domain", last.GPUUUID)
	}
}

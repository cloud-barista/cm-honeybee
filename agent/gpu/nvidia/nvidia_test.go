package nvidia

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
)

func load(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}

	return data
}

func parseFile(t *testing.T, name string) ([]infra.NVIDIA, string) {
	t.Helper()

	gpus, schema, err := parse(load(t, name))
	if err != nil {
		t.Fatalf("parsing %s: %v", name, err)
	}

	return gpus, schema
}

func TestDetectSchema(t *testing.T) {
	cases := map[string]string{
		"v13-single.xml":        "v13",
		"vgpu-guest-v13.xml":    "v13",
		"v12-mig.xml":           "v12",
		"v11.xml":               "v11",
		"legacy-no-doctype.xml": "v11",
	}

	for name, want := range cases {
		if got := detectSchema(load(t, name)); got != want {
			t.Errorf("%s: schema = %q, want %q", name, got, want)
		}
	}
}

// An unknown schema version must not be dropped: it is newer than anything we
// know, so it is read with the newest parser rather than rejected.
func TestUnknownSchemaFallsBackToLatest(t *testing.T) {
	data := strings.Replace(string(load(t, "v13-single.xml")),
		"nvsmi_device_v13.dtd", "nvsmi_device_v99.dtd", 1)

	gpus, schema, err := parse([]byte(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if schema != latestSchema {
		t.Errorf("schema = %q, want %q", schema, latestSchema)
	}

	if len(gpus) != 1 {
		t.Fatalf("got %d GPUs, want 1", len(gpus))
	}
}

func TestParseV13(t *testing.T) {
	gpus, schema := parseFile(t, "v13-single.xml")

	if schema != "v13" {
		t.Fatalf("schema = %q, want v13", schema)
	}

	if len(gpus) != 1 {
		t.Fatalf("got %d GPUs, want 1", len(gpus))
	}

	g := gpus[0]

	wantAttr := map[string]string{
		"gpu_uuid":       "GPU-00000000-1111-2222-3333-444444444444",
		"driver_version": "610.57.04",
		"cuda_version":   "13.3",
		"product_name":   "NVIDIA L40S",
		"pci_bus_id":     "00000000:01:00.0",
		"minor_number":   "0",
		"serial":         "1234567890123",
		"vbios_version":  "95.02.6C.00.01",
		"compute_mode":   "Default",
	}
	gotAttr := map[string]string{
		"gpu_uuid":       g.DeviceAttribute.GPUUUID,
		"driver_version": g.DeviceAttribute.DriverVersion,
		"cuda_version":   g.DeviceAttribute.CUDAVersion,
		"product_name":   g.DeviceAttribute.ProductName,
		"pci_bus_id":     g.DeviceAttribute.PCIBusID,
		"minor_number":   g.DeviceAttribute.MinorNumber,
		"serial":         g.DeviceAttribute.Serial,
		"vbios_version":  g.DeviceAttribute.VBIOSVersion,
		"compute_mode":   g.DeviceAttribute.ComputeMode,
	}

	for key, want := range wantAttr {
		if gotAttr[key] != want {
			t.Errorf("%s = %q, want %q", key, gotAttr[key], want)
		}
	}

	// mig_mode reads N/A on a card that has no MIG support, which must not be
	// carried through as if it were a mode.
	if g.DeviceAttribute.MIGMode != "" {
		t.Errorf("mig_mode = %q, want empty", g.DeviceAttribute.MIGMode)
	}

	if got := deref32(t, g.Performance.GPUUsage); got != 25 {
		t.Errorf("gpu_usage = %d, want 25", got)
	}

	if got := deref64(t, g.Performance.FBMemoryTotal); got != 46068 {
		t.Errorf("fb_memory_total = %d, want 46068", got)
	}

	// instant_power_draw wins over average_power_draw on v13.
	if g.Performance.PowerDraw == nil || *g.Performance.PowerDraw != 39.02 {
		t.Errorf("power_draw = %v, want 39.02", g.Performance.PowerDraw)
	}

	if g.Performance.PowerLimit == nil || *g.Performance.PowerLimit != 350 {
		t.Errorf("power_limit = %v, want 350", g.Performance.PowerLimit)
	}

	// fan_speed reads N/A on a passively cooled card: absent, not zero.
	if g.Performance.FanSpeed != nil {
		t.Errorf("fan_speed = %v, want nil", *g.Performance.FanSpeed)
	}

	if got := deref32(t, g.Performance.PCIeLinkWidth); got != 16 {
		t.Errorf("pcie_link_width_current = %d, want 16", got)
	}

	// Only gpu_idle is active, which is bit 0.
	if got := deref64(t, g.Performance.ClocksEventReasons); got != 0x1 {
		t.Errorf("clocks_event_reasons = %#x, want 0x1", got)
	}

	if g.ECC == nil {
		t.Fatal("ecc is nil, want a block")
	}

	if g.ECC.Mode != "Enabled" {
		t.Errorf("ecc mode = %q, want Enabled", g.ECC.Mode)
	}

	// v13 splits uncorrectable SRAM into parity and SECDED; both are
	// uncorrectable SRAM errors, so the reported counter is their sum.
	if got := deref64(t, g.ECC.Volatile.SRAMUncorrectable); got != 5 {
		t.Errorf("volatile sram_uncorrectable = %d, want 5", got)
	}

	if len(g.Processes) != 1 || g.Processes[0].PID != 4242 {
		t.Fatalf("processes = %+v, want one entry with pid 4242", g.Processes)
	}
}

func TestParseV12MultiGPUAndMIG(t *testing.T) {
	gpus, schema := parseFile(t, "v12-mig.xml")

	if schema != "v12" {
		t.Fatalf("schema = %q, want v12", schema)
	}

	if len(gpus) != 2 {
		t.Fatalf("got %d GPUs, want 2", len(gpus))
	}

	first, second := gpus[0], gpus[1]

	if first.DeviceAttribute.Index != 0 || second.DeviceAttribute.Index != 1 {
		t.Errorf("index = %d,%d, want 0,1",
			first.DeviceAttribute.Index, second.DeviceAttribute.Index)
	}

	if first.DeviceAttribute.GPUUUID == second.DeviceAttribute.GPUUUID {
		t.Error("both GPUs carry the same uuid")
	}

	if first.DeviceAttribute.MIGMode != "Enabled" {
		t.Errorf("mig_mode = %q, want Enabled", first.DeviceAttribute.MIGMode)
	}

	if len(first.MIGDevices) != 1 {
		t.Fatalf("got %d MIG devices, want 1", len(first.MIGDevices))
	}

	mig := first.MIGDevices[0]
	if mig.GPUInstanceID != "1" || mig.UUID != "MIG-aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Errorf("mig device = %+v", mig)
	}

	if got := deref64(t, mig.FBMemoryTotal); got != 19968 {
		t.Errorf("mig fb_memory_total = %d, want 19968", got)
	}

	// v12 keeps power_draw inside the renamed gpu_power_readings block.
	if first.Performance.PowerDraw == nil || *first.Performance.PowerDraw != 310.55 {
		t.Errorf("power_draw = %v, want 310.55", first.Performance.PowerDraw)
	}

	// sw_power_cap is bit 2.
	if got := deref64(t, first.Performance.ClocksEventReasons); got != 0x4 {
		t.Errorf("clocks_event_reasons = %#x, want 0x4", got)
	}

	// The second GPU reports nothing readable. None of it may surface as zero,
	// and no percentage may be computed from a missing total.
	if second.Performance.GPUUsage != nil {
		t.Errorf("gpu_usage = %v, want nil", *second.Performance.GPUUsage)
	}

	if second.Performance.MemoryUsage != nil {
		t.Errorf("memory_usage = %v, want nil", *second.Performance.MemoryUsage)
	}

	if second.Performance.FBMemoryTotal != nil {
		t.Errorf("fb_memory_total = %v, want nil", *second.Performance.FBMemoryTotal)
	}

	if second.Performance.FBMemoryUsage != nil {
		t.Errorf("fb_memory_usage = %v, want nil", *second.Performance.FBMemoryUsage)
	}

	if second.Performance.TemperatureGPU != nil {
		t.Errorf("temperature_gpu = %v, want nil", *second.Performance.TemperatureGPU)
	}

	if second.Performance.PowerDraw != nil {
		t.Errorf("power_draw = %v, want nil", *second.Performance.PowerDraw)
	}

	// No reason was reported at all, which is different from every reason
	// being inactive.
	if second.Performance.ClocksEventReasons != nil {
		t.Errorf("clocks_event_reasons = %v, want nil", *second.Performance.ClocksEventReasons)
	}

	// A GPU that reports no ECC mode and no counter gets no ECC block.
	if second.ECC != nil {
		t.Errorf("ecc = %+v, want nil", second.ECC)
	}
}

func TestParseV11(t *testing.T) {
	gpus, schema := parseFile(t, "v11.xml")

	if schema != "v11" {
		t.Fatalf("schema = %q, want v11", schema)
	}

	g := gpus[0]

	// v11 names the block power_readings, not gpu_power_readings.
	if g.Performance.PowerDraw == nil || *g.Performance.PowerDraw != 26.78 {
		t.Errorf("power_draw = %v, want 26.78", g.Performance.PowerDraw)
	}

	if g.Performance.PowerLimit == nil || *g.Performance.PowerLimit != 70 {
		t.Errorf("power_limit = %v, want 70", g.Performance.PowerLimit)
	}

	// v11 names the block clocks_throttle_reasons; hw_slowdown is bit 3.
	if got := deref64(t, g.Performance.ClocksEventReasons); got != 0x8 {
		t.Errorf("clocks_event_reasons = %#x, want 0x8", got)
	}

	if got := deref32(t, g.Performance.FBMemoryUsage); got != 25 {
		t.Errorf("fb_memory_usage = %d, want 25", got)
	}

	if got := deref32(t, g.Performance.PCIeLinkWidth); got != 8 {
		t.Errorf("pcie_link_width_current = %d, want 8", got)
	}
}

func TestParseLegacyWithoutDoctype(t *testing.T) {
	gpus, schema := parseFile(t, "legacy-no-doctype.xml")

	if schema != "v11" {
		t.Fatalf("schema = %q, want v11", schema)
	}

	g := gpus[0]

	if g.DeviceAttribute.ProductName != "Quadro P400" {
		t.Errorf("product_name = %q", g.DeviceAttribute.ProductName)
	}

	// This driver reports no architecture at all.
	if g.DeviceAttribute.ProductArchitecture != "" {
		t.Errorf("product_architecture = %q, want empty", g.DeviceAttribute.ProductArchitecture)
	}

	// A pre-Volta card counts ECC errors by bit width instead of by location.
	if g.ECC == nil {
		t.Fatal("ecc is nil, want a block")
	}

	if got := deref64(t, g.ECC.Aggregate.SingleBit); got != 4 {
		t.Errorf("aggregate single_bit = %d, want 4", got)
	}

	if g.ECC.Volatile.SingleBit != nil {
		t.Errorf("volatile single_bit = %v, want nil", *g.ECC.Volatile.SingleBit)
	}

	if g.ECC.Aggregate.SRAMCorrectable != nil {
		t.Errorf("aggregate sram_correctable = %v, want nil", *g.ECC.Aggregate.SRAMCorrectable)
	}

	// used is a real 0 here, so the percentage is a real 0 too.
	if got := deref32(t, g.Performance.FBMemoryUsage); got != 0 {
		t.Errorf("fb_memory_usage = %d, want 0", got)
	}

	if g.Performance.PowerDraw != nil {
		t.Errorf("power_draw = %v, want nil", *g.Performance.PowerDraw)
	}

	if got := deref32(t, g.Performance.FanSpeed); got != 39 {
		t.Errorf("fan_speed = %d, want 39", got)
	}
}

func TestParseVGPUGuest(t *testing.T) {
	gpus, _ := parseFile(t, "vgpu-guest-v13.xml")

	g := gpus[0]

	if g.DeviceAttribute.VirtualizationMode != "VGPU" {
		t.Errorf("virtualization_mode = %q, want VGPU", g.DeviceAttribute.VirtualizationMode)
	}

	if !strings.HasPrefix(g.DeviceAttribute.VGPULicenseStatus, "Licensed") {
		t.Errorf("vgpu_license_status = %q", g.DeviceAttribute.VGPULicenseStatus)
	}

	// A vGPU guest sees framebuffer but no temperature or power.
	if got := deref64(t, g.Performance.FBMemoryTotal); got != 8192 {
		t.Errorf("fb_memory_total = %d, want 8192", got)
	}

	if g.Performance.TemperatureGPU != nil {
		t.Errorf("temperature_gpu = %v, want nil", *g.Performance.TemperatureGPU)
	}

	// bar1 total is a real 0 on a vGPU guest, so no percentage can be taken
	// from it and none may be invented.
	if g.Performance.Bar1MemoryUsage != nil {
		t.Errorf("bar1_memory_usage = %v, want nil", *g.Performance.Bar1MemoryUsage)
	}
}

func TestParseRejectsNonXML(t *testing.T) {
	if _, _, err := parse([]byte("nvidia-smi: command failed\n")); err == nil {
		t.Fatal("want an error for output that is not XML")
	}
}

func TestParseNVMLVersion(t *testing.T) {
	output := "NVIDIA-SMI version  : 610.57.04\n" +
		"NVML version        : 610.57\n" +
		"DRIVER version      : Deprecated, see \"KMD version\" instead\n"

	if got := parseNVMLVersion(output); got != "610.57" {
		t.Errorf("nvml version = %q, want 610.57", got)
	}

	if got := parseNVMLVersion("nvidia-smi: unrecognised option '--version'"); got != "" {
		t.Errorf("nvml version = %q, want empty", got)
	}
}

func deref32(t *testing.T, v *uint32) uint32 {
	t.Helper()

	if v == nil {
		t.Fatal("value is nil, want a number")
	}

	return *v
}

func deref64(t *testing.T, v *uint64) uint64 {
	t.Helper()

	if v == nil {
		t.Fatal("value is nil, want a number")
	}

	return *v
}

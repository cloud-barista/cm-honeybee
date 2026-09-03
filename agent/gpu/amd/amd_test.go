package amd

import (
	"os"
	"path/filepath"
	"testing"
)

func load(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}

	return data
}

func TestParseOlderROCm(t *testing.T) {
	gpus, err := parse(load(t, "rocm-old.json"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(gpus) != 2 {
		t.Fatalf("got %d GPUs, want 2", len(gpus))
	}

	first := gpus[0]

	if first.DeviceAttribute.Card != "card0" {
		t.Errorf("card = %q, want card0", first.DeviceAttribute.Card)
	}

	// This release spells the product key "Card series" in lower case.
	if first.DeviceAttribute.ProductName != "Instinct MI100" {
		t.Errorf("product_name = %q", first.DeviceAttribute.ProductName)
	}

	if first.DeviceAttribute.DriverVersion != "6.2.4" {
		t.Errorf("driver_version = %q, want 6.2.4", first.DeviceAttribute.DriverVersion)
	}

	if first.DeviceAttribute.PCIBusID != "0000:1E:00.0" {
		t.Errorf("pci_bus_id = %q", first.DeviceAttribute.PCIBusID)
	}

	// "Unique ID" reads N/A here and must not be carried through.
	if first.DeviceAttribute.UniqueID != "" {
		t.Errorf("unique_id = %q, want empty", first.DeviceAttribute.UniqueID)
	}

	// rocm-smi counts VRAM in bytes; the model carries MiB.
	if got := deref64(t, first.Performance.VRAMMemoryTotal); got != 32752 {
		t.Errorf("vram_memory_total = %d MiB, want 32752", got)
	}

	if got := deref64(t, first.Performance.VRAMMemoryUsed); got != 2294 {
		t.Errorf("vram_memory_used = %d MiB, want 2294", got)
	}

	// Clock readings arrive wrapped in parentheses with their unit attached.
	if got := deref32(t, first.Performance.ClockSM); got != 300 {
		t.Errorf("clock_sm = %d, want 300", got)
	}

	if got := deref32(t, first.Performance.ClockMemory); got != 1200 {
		t.Errorf("clock_memory = %d, want 1200", got)
	}

	if first.Performance.TemperatureGPU == nil || *first.Performance.TemperatureGPU != 31 {
		t.Errorf("temperature_gpu = %v, want 31", first.Performance.TemperatureGPU)
	}

	if first.Performance.PowerCap == nil || *first.Performance.PowerCap != 290 {
		t.Errorf("power_cap = %v, want 290", first.Performance.PowerCap)
	}

	// The second card answers N/A to everything. None of that may surface as
	// zero, and its fan speed of 0 on the first card must stay a real zero.
	second := gpus[1]

	if second.Performance.GPUUsage != nil {
		t.Errorf("gpu_usage = %v, want nil", *second.Performance.GPUUsage)
	}

	if second.Performance.VRAMMemoryTotal != nil {
		t.Errorf("vram_memory_total = %v, want nil", *second.Performance.VRAMMemoryTotal)
	}

	if second.Performance.TemperatureGPU != nil {
		t.Errorf("temperature_gpu = %v, want nil", *second.Performance.TemperatureGPU)
	}

	if second.Performance.PerformanceLevel != "" {
		t.Errorf("performance_level = %q, want empty", second.Performance.PerformanceLevel)
	}

	if got := deref32(t, first.Performance.FanSpeed); got != 0 {
		t.Errorf("fan_speed = %d, want 0", got)
	}
}

func TestParseNewerROCm(t *testing.T) {
	gpus, err := parse(load(t, "rocm-new.json"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(gpus) != 2 {
		t.Fatalf("got %d GPUs, want 2", len(gpus))
	}

	// Cards are ordered by their numeric index, so card2 comes before card10.
	if gpus[0].DeviceAttribute.Card != "card2" || gpus[1].DeviceAttribute.Card != "card10" {
		t.Fatalf("card order = %q,%q, want card2,card10",
			gpus[0].DeviceAttribute.Card, gpus[1].DeviceAttribute.Card)
	}

	g := gpus[1]

	// This release renames the keys, and "Device Name" wins over "Card Series".
	if g.DeviceAttribute.ProductName != "Instinct MI300X" {
		t.Errorf("product_name = %q", g.DeviceAttribute.ProductName)
	}

	if g.DeviceAttribute.DeviceID != "0x74a1" {
		t.Errorf("device_id = %q", g.DeviceAttribute.DeviceID)
	}

	if g.DeviceAttribute.UniqueID != "0x79ccd55167a2124a" {
		t.Errorf("unique_id = %q", g.DeviceAttribute.UniqueID)
	}

	if g.DeviceAttribute.ComputePartition != "SPX" || g.DeviceAttribute.MemoryPartition != "NPS1" {
		t.Errorf("partitions = %q,%q, want SPX,NPS1",
			g.DeviceAttribute.ComputePartition, g.DeviceAttribute.MemoryPartition)
	}

	// The memory usage key was renamed to "GPU Memory Allocated (VRAM%)".
	if got := deref32(t, g.Performance.MemoryUsage); got != 0 {
		t.Errorf("memory_usage = %d, want 0", got)
	}

	// A used counter of 0 is a real reading, not a missing one.
	if got := deref64(t, gpus[0].Performance.VRAMMemoryUsed); got != 0 {
		t.Errorf("vram_memory_used = %d, want 0", got)
	}
}

// The "system" entry is not a card and must never be reported as one.
func TestParseIgnoresSystemEntry(t *testing.T) {
	gpus, err := parse([]byte(`{"system":{"Driver version":"6.10.5"}}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(gpus) != 0 {
		t.Fatalf("got %d GPUs, want 0", len(gpus))
	}
}

func TestParseRejectsNonJSON(t *testing.T) {
	if _, err := parse([]byte("ERROR: GPU not found\n")); err == nil {
		t.Fatal("want an error for output that is not JSON")
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

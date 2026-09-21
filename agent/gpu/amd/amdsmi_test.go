package amd

import "testing"

// The document shape follows amd-smi's own CLI source and tests: a gpu_data
// array of per card objects split into sections, every measured field carrying
// its unit as {"value": N, "unit": "..."}, and a reading the driver could not
// take written as the plain string "N/A" in the same position.
//
// It is hand-built rather than captured. No AMD device was available to run
// amd-smi on, so this pins the shape the reader was written against and not a
// reading taken from hardware.
func TestParseAMDSMIStatic(t *testing.T) {
	gpus, err := parseAMDSMI(load(t, "amd-smi-static.json"))
	if err != nil {
		t.Fatalf("parseAMDSMI: %v", err)
	}

	if len(gpus) != 2 {
		t.Fatalf("got %d GPUs, want 2", len(gpus))
	}

	first := gpus[0]

	if first.DeviceAttribute.Card != "card0" {
		t.Errorf("card = %q, want card0", first.DeviceAttribute.Card)
	}
	if first.DeviceAttribute.ProductName != "Instinct MI300X" {
		t.Errorf("product_name = %q", first.DeviceAttribute.ProductName)
	}
	if first.DeviceAttribute.PCIBusID != "0000:0C:00.0" {
		t.Errorf("pci_bus_id = %q", first.DeviceAttribute.PCIBusID)
	}
	if first.DeviceAttribute.DriverVersion != "6.10.5" {
		t.Errorf("driver_version = %q", first.DeviceAttribute.DriverVersion)
	}
	if first.DeviceAttribute.DeviceID != "0x74a1" {
		t.Errorf("device_id = %q", first.DeviceAttribute.DeviceID)
	}
	if first.DeviceAttribute.SerialNumber != "0x8B4F1C2D9E0A3B57" {
		t.Errorf("serial_number = %q", first.DeviceAttribute.SerialNumber)
	}

	// 196608 is the card's 192 GiB counted in binary megabytes. Read as decimal
	// MB it would come out as 187.5 GiB and disagree with the card for no
	// reason a reader of the field could see.
	if first.Performance.VRAMMemoryTotal == nil {
		t.Fatal("vram_memory_total is nil, want the reported size")
	}
	if *first.Performance.VRAMMemoryTotal != 196608 {
		t.Errorf("vram_memory_total = %d, want 196608 MiB", *first.Performance.VRAMMemoryTotal)
	}

	// The second card numbers itself, so it must not collapse onto the first.
	if gpus[1].DeviceAttribute.Card != "card1" {
		t.Errorf("second card = %q, want card1", gpus[1].DeviceAttribute.Card)
	}
}

// "N/A" is what amd-smi writes where a reading belongs. Kept as a value it
// becomes a serial number every such card shares, and a size of zero is a
// statement this reader has no basis for.
func TestParseAMDSMIKeepsNotAvailableAbsent(t *testing.T) {
	gpus, err := parseAMDSMI(load(t, "amd-smi-static.json"))
	if err != nil {
		t.Fatalf("parseAMDSMI: %v", err)
	}

	second := gpus[1]

	if second.DeviceAttribute.SerialNumber != "" {
		t.Errorf("serial_number = %q, want empty: the document says N/A", second.DeviceAttribute.SerialNumber)
	}
	if second.Performance.VRAMMemoryTotal != nil {
		t.Errorf("vram_memory_total = %d, want nil: the document says N/A",
			*second.Performance.VRAMMemoryTotal)
	}
}

// The live readings are not collected, and leaving them nil is the model's way
// of saying so. Filling them with zero would report an idle card.
func TestParseAMDSMILeavesLiveReadingsUnreported(t *testing.T) {
	gpus, err := parseAMDSMI(load(t, "amd-smi-static.json"))
	if err != nil {
		t.Fatalf("parseAMDSMI: %v", err)
	}

	first := gpus[0]

	if first.Performance.GPUUsage != nil {
		t.Error("gpu_usage must stay unreported: amd-smi static does not carry it")
	}
	if first.Performance.TemperatureGPU != nil {
		t.Error("temperature_gpu must stay unreported: amd-smi static does not carry it")
	}
	if first.Performance.PowerDraw != nil {
		t.Error("power_draw must stay unreported: amd-smi static does not carry it")
	}
}

// A document with no entries is a host with no AMD card, which is not an error.
func TestParseAMDSMIEmptyDocument(t *testing.T) {
	gpus, err := parseAMDSMI([]byte(`{"gpu_data":[]}`))
	if err != nil {
		t.Fatalf("parseAMDSMI: %v", err)
	}
	if len(gpus) != 0 {
		t.Errorf("got %d GPUs, want 0", len(gpus))
	}

	if gpus, err = parseAMDSMI([]byte("  \n")); err != nil || len(gpus) != 0 {
		t.Errorf("empty output gave (%d, %v), want (0, nil)", len(gpus), err)
	}
}

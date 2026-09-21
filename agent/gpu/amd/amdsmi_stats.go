package amd

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
)

// amdSMIDocument is what "amd-smi static ... --json" writes. The document is
// either an object holding gpu_data or a bare array, and amd-smi's own CLI
// tests accept both.
type amdSMIDocument struct {
	// A pointer so that a document carrying an empty gpu_data can be told from
	// one carrying no such key. The first is a host with no AMD card, which is
	// an answer; the second means this is the bare array form instead.
	GPUData *[]amdSMIGPU `json:"gpu_data"`
}

// amdSMIGPU is one card, split into the sections the static subcommand emits.
// Only the sections queried here are modelled; a newer amd-smi that adds one
// is ignored rather than rejected.
type amdSMIGPU struct {
	GPU  int `json:"gpu"`
	ASIC struct {
		MarketName string `json:"market_name"`
		VendorName string `json:"vendor_name"`
		DeviceID   string `json:"device_id"`
		ASICSerial string `json:"asic_serial"`
	} `json:"asic"`
	Bus struct {
		BDF string `json:"bdf"`
	} `json:"bus"`
	VRAM struct {
		Size amdSMIMeasure `json:"size"`
	} `json:"vram"`
	Driver struct {
		Version string `json:"version"`
	} `json:"driver"`
}

// amdSMIMeasure is a measured field in amd-smi's JSON, which carries its unit
// rather than being a bare number: {"value": 196608, "unit": "MB"}. A reading
// the driver could not take is the plain string "N/A" in the same position, so
// the two have to be decoded together and absence kept apart from zero.
type amdSMIMeasure struct {
	Value   float64
	Unit    string
	Present bool
}

func (m *amdSMIMeasure) UnmarshalJSON(data []byte) error {
	var wrapped struct {
		Value *float64 `json:"value"`
		Unit  string   `json:"unit"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil && wrapped.Value != nil {
		m.Value, m.Unit, m.Present = *wrapped.Value, wrapped.Unit, true

		return nil
	}

	var bare float64
	if err := json.Unmarshal(data, &bare); err == nil {
		m.Value, m.Present = bare, true

		return nil
	}

	// Anything else, "N/A" included, stays absent rather than becoming zero.
	return nil
}

// queryAMDSMI collects every AMD GPU visible to amd-smi.
//
// amd-smi is the supported successor to rocm-smi, which takes only critical
// fixes from ROCm 7.0 and is removed in 10.1. A host on a current ROCm may
// carry no rocm-smi at all, and this is the reader that answers there.
//
// Only the static blocks are collected. The live readings sit behind
// "amd-smi metric", and this agent inventories a source host rather than
// watching it, so every performance field is left unreported instead of being
// filled from a second command.
func queryAMDSMI() ([]infra.AMD, error) {
	output, err := runAMDSMI(amdSMIStaticArgs...)
	if err != nil {
		return []infra.AMD{}, err
	}

	return parseAMDSMI(output)
}

// parseAMDSMI turns an amd-smi static document into the common GPU model.
func parseAMDSMI(data []byte) ([]infra.AMD, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return []infra.AMD{}, nil
	}

	var document amdSMIDocument
	if err := json.Unmarshal([]byte(trimmed), &document); err != nil || document.GPUData == nil {
		var bare []amdSMIGPU
		if bareErr := json.Unmarshal([]byte(trimmed), &bare); bareErr != nil {
			if err != nil {
				return nil, err
			}

			return nil, bareErr
		}
		document.GPUData = &bare
	}

	entries := *document.GPUData
	gpus := make([]infra.AMD, 0, len(entries))

	for index, gpu := range entries {
		card := gpu.GPU
		if card == 0 && index > 0 {
			// A document that does not number its entries still has to keep
			// them apart, or every card is reported as card0.
			card = index
		}

		entry := infra.AMD{
			DeviceAttribute: infra.AMDDeviceAttribute{
				Card:          "card" + strconv.Itoa(card),
				GPUID:         amdSMIValue(gpu.ASIC.DeviceID),
				ProductName:   amdSMIValue(gpu.ASIC.MarketName),
				SerialNumber:  amdSMIValue(gpu.ASIC.ASICSerial),
				PCIBusID:      amdSMIValue(gpu.Bus.BDF),
				DriverVersion: amdSMIValue(gpu.Driver.Version),
				// The rocm-smi reader puts the vendor's name in this field
				// rather than its identifier, and one field holding two kinds
				// of value depending on which tool answered would be worse
				// than the naming already is.
				VendorID: amdSMIValue(gpu.ASIC.VendorName),
				DeviceID: amdSMIValue(gpu.ASIC.DeviceID),
			},
		}
		if total, ok := amdSMIMiB(gpu.VRAM.Size); ok {
			entry.Performance.VRAMMemoryTotal = &total
		}

		gpus = append(gpus, entry)
	}

	return gpus, nil
}

// amdSMIValue drops a field the driver could not report, which amd-smi writes
// as "N/A" rather than omitting.
func amdSMIValue(field string) string {
	trimmed := strings.TrimSpace(field)
	if strings.EqualFold(trimmed, "N/A") || strings.EqualFold(trimmed, "not supported") {
		return ""
	}

	return trimmed
}

// amdSMIMiB converts a VRAM reading to the MiB the GPU model uses.
//
// amd-smi labels the unit "MB" but counts in binary: an MI300X reports 196608,
// which is exactly the 192 GiB the card carries, where 196608 decimal MB would
// be 187.5 GiB. So a bare "MB" from this tool is read as MiB, and only the
// units that are unambiguous are converted.
func amdSMIMiB(size amdSMIMeasure) (uint64, bool) {
	if !size.Present || size.Value <= 0 {
		return 0, false
	}

	switch strings.ToLower(strings.TrimSpace(size.Unit)) {
	case "gib", "gb":
		return uint64(size.Value * 1024), true
	case "kib", "kb":
		return uint64(size.Value / 1024), true
	case "b", "bytes":
		return uint64(size.Value / bytesPerMiB), true
	default:
		return uint64(size.Value), true
	}
}

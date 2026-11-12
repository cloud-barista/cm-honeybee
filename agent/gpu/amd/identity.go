package amd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/gpu/pci"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/jollaman999/utils/logger"
)

// sysPCIDevicesDir is where the kernel lists PCI devices by address. A
// variable so tests can point it at a fixture tree.
var sysPCIDevicesDir = "/sys/bus/pci/devices"

// amdSMIListArgs asks amd-smi for the card list, the only place it prints a
// card's UUID. rocm-smi does not print one at all.
var amdSMIListArgs = []string{"list", "--json"}

// amdSMIListEntry is one card in "amd-smi list --json".
type amdSMIListEntry struct {
	BDF  string `json:"bdf"`
	UUID string `json:"uuid"`
}

// fillIdentity completes the identifiers neither reader takes from its main
// command: the PCI vendor id, which rocm-smi only prints as a name, and the
// UUID, which only "amd-smi list" prints. Both are matched to a card by its
// PCI address, and a card whose address is unknown or not found keeps the
// fields empty rather than taking another card's value.
func fillIdentity(gpus []infra.AMD) {
	if len(gpus) == 0 {
		return
	}

	uuids := map[string]string{}

	output, err := runAMDSMI(amdSMIListArgs...)
	if err == nil {
		uuids, err = parseAMDSMIList(output)
	}
	if err != nil {
		logger.Println(logger.DEBUG, false, "AMD: card UUIDs not collected: "+err.Error())
	}

	applyIdentity(gpus, uuids)
}

// applyIdentity fills each card's vendor id from sysfs and its UUID from the
// address-to-UUID map, leaving a field that already holds a value alone.
func applyIdentity(gpus []infra.AMD, uuids map[string]string) {
	for i := range gpus {
		attribute := &gpus[i].DeviceAttribute
		address := pci.NormalizeBusID(attribute.PCIBusID)
		if address == "" {
			continue
		}

		if attribute.VendorID == "" {
			attribute.VendorID = pciVendorID(address)
		}
		if attribute.GPUUUID == "" {
			attribute.GPUUUID = uuids[address]
		}
	}
}

// parseAMDSMIList maps each card's PCI address to its UUID. The document is a
// bare array on current amd-smi, and the gpu_data wrapper the static
// subcommand can use is accepted as well.
func parseAMDSMIList(data []byte) (map[string]string, error) {
	uuids := map[string]string{}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return uuids, nil
	}

	var entries []amdSMIListEntry
	if err := json.Unmarshal([]byte(trimmed), &entries); err != nil {
		var wrapped struct {
			GPUData []amdSMIListEntry `json:"gpu_data"`
		}
		if wrappedErr := json.Unmarshal([]byte(trimmed), &wrapped); wrappedErr != nil {
			return uuids, err
		}
		entries = wrapped.GPUData
	}

	for _, entry := range entries {
		address := pci.NormalizeBusID(amdSMIValue(entry.BDF))
		uuid := amdSMIValue(entry.UUID)
		if address == "" || uuid == "" {
			continue
		}
		uuids[address] = uuid
	}

	return uuids, nil
}

// pciVendorID reads the vendor id the kernel reports for a PCI device, such as
// "0x1002". It returns an empty string when the device is not listed.
func pciVendorID(address string) string {
	data, err := os.ReadFile(filepath.Join(sysPCIDevicesDir, address, "vendor"))
	if err != nil {
		return ""
	}

	return strings.ToLower(strings.TrimSpace(string(data)))
}

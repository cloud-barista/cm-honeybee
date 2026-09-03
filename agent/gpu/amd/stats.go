package amd

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/jollaman999/utils/logger"
)

// bytesPerMiB converts the byte counters rocm-smi reports for VRAM into the
// MiB the GPU model uses everywhere else.
const bytesPerMiB = 1024 * 1024

// rocm-smi labels its JSON keys with the same human-readable strings it prints
// in table mode, and it has renamed several of them across releases. Each entry
// lists the accepted spellings newest first.
var (
	keyProductName  = []string{"Device Name", "Card Series", "Card series", "Card Model", "Card model", "Card SKU"}
	keyDeviceID     = []string{"Device ID", "GPU ID"}
	keyUniqueID     = []string{"Unique ID", "GUID"}
	keyVendor       = []string{"Card Vendor", "Card vendor"}
	keyVBIOS        = []string{"VBIOS version"}
	keySerial       = []string{"Serial Number"}
	keyPCIBus       = []string{"PCI Bus"}
	keyGPUUse       = []string{"GPU use (%)"}
	keyMemoryUse    = []string{"GPU Memory Allocated (VRAM%)", "GPU memory use (%)"}
	keyVRAMTotal    = []string{"VRAM Total Memory (B)"}
	keyVRAMUsed     = []string{"VRAM Total Used Memory (B)"}
	keyPerfLevel    = []string{"Performance Level"}
	keyFanSpeed     = []string{"Fan speed (%)"}
	keyTempEdge     = []string{"Temperature (Sensor edge) (C)"}
	keyTempMemory   = []string{"Temperature (Sensor memory) (C)"}
	keyPowerDraw    = []string{"Average Graphics Package Power (W)", "Current Socket Graphics Package Power (W)"}
	keyPowerCap     = []string{"Max Graphics Package Power (W)"}
	keyClockSM      = []string{"sclk clock speed:"}
	keyClockMemory  = []string{"mclk clock speed:"}
	keyComputePart  = []string{"Compute Partition"}
	keyMemoryPart   = []string{"Memory Partition"}
	keyDriverSystem = []string{"Driver version"}
)

// QueryGPU collects every AMD GPU visible to rocm-smi.
//
// It returns an empty slice and an error when rocm-smi is missing or does not
// answer. A host with no AMD GPU is not an error: rocm-smi answers with a
// document that holds no card entry.
func QueryGPU() ([]infra.AMD, error) {
	output, err := runROCmSMI(queryArgs...)
	if errors.Is(err, ErrNotAvailable) {
		logger.Println(logger.DEBUG, false, err.Error())

		return []infra.AMD{}, err
	}

	if err != nil {
		// A release that rejects one of the explicit flags fails the whole run,
		// so retry once with the "-a" shorthand before giving up.
		logger.Println(logger.DEBUG, false, err.Error()+" (retrying with -a)")

		output, err = runROCmSMI(fallbackArgs...)
		if err != nil {
			logger.Println(logger.DEBUG, false, err.Error())

			return []infra.AMD{}, err
		}
	}

	gpus, err := parse(output)
	if err != nil {
		logger.Println(logger.DEBUG, false, "AMD: failed to parse rocm-smi output: "+err.Error())

		return []infra.AMD{}, err
	}

	return gpus, nil
}

// parse turns a rocm-smi JSON document into the common GPU model. The document
// is a map keyed by card name, plus a "system" entry that carries host-wide
// values such as the driver version.
func parse(data []byte) ([]infra.AMD, error) {
	var document map[string]map[string]json.RawMessage

	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}

	driverVersion := lookup(document["system"], keyDriverSystem)

	cards := make([]string, 0, len(document))
	for name := range document {
		if strings.HasPrefix(name, "card") {
			cards = append(cards, name)
		}
	}

	// Map iteration order is random, so sort by the trailing card index to keep
	// the reported order stable between collections.
	sort.Slice(cards, func(i, j int) bool { return cardIndex(cards[i]) < cardIndex(cards[j]) })

	gpus := make([]infra.AMD, 0, len(cards))

	for _, name := range cards {
		card := document[name]

		vramTotal := mib(lookup(card, keyVRAMTotal))
		vramUsed := mib(lookup(card, keyVRAMUsed))

		gpus = append(gpus, infra.AMD{
			DeviceAttribute: infra.AMDDeviceAttribute{
				Card:             name,
				GPUID:            lookup(card, keyDeviceID),
				UniqueID:         lookup(card, keyUniqueID),
				ProductName:      lookup(card, keyProductName),
				SerialNumber:     lookup(card, keySerial),
				PCIBusID:         lookup(card, keyPCIBus),
				VBIOSVersion:     lookup(card, keyVBIOS),
				DriverVersion:    driverVersion,
				VendorID:         lookup(card, keyVendor),
				DeviceID:         lookup(card, keyDeviceID),
				ComputePartition: lookup(card, keyComputePart),
				MemoryPartition:  lookup(card, keyMemoryPart),
			},
			Performance: infra.AMDPerformance{
				GPUUsage:          uint32Of(lookup(card, keyGPUUse)),
				MemoryUsage:       uint32Of(lookup(card, keyMemoryUse)),
				VRAMMemoryUsed:    vramUsed,
				VRAMMemoryTotal:   vramTotal,
				PerformanceLevel:  lookup(card, keyPerfLevel),
				FanSpeed:          uint32Of(lookup(card, keyFanSpeed)),
				TemperatureGPU:    float64Of(lookup(card, keyTempEdge)),
				TemperatureMemory: float64Of(lookup(card, keyTempMemory)),
				PowerDraw:         float64Of(lookup(card, keyPowerDraw)),
				PowerCap:          float64Of(lookup(card, keyPowerCap)),
				ClockSM:           clock(lookup(card, keyClockSM)),
				ClockMemory:       clock(lookup(card, keyClockMemory)),
			},
		})
	}

	return gpus, nil
}

// cardIndex extracts the numeric suffix of a card name so "card10" sorts after
// "card2". A name without a numeric suffix sorts last.
func cardIndex(name string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(name, "card"))
	if err != nil {
		return 1 << 30
	}

	return n
}

// lookup returns the first key that is present and carries a reading. rocm-smi
// answers a query it cannot serve with "N/A", so those are treated as absent.
func lookup(card map[string]json.RawMessage, keys []string) string {
	for _, key := range keys {
		raw, ok := card[key]
		if !ok {
			continue
		}

		value := unquote(raw)
		if value == "" || strings.EqualFold(value, "N/A") ||
			strings.EqualFold(value, "not supported") {
			continue
		}

		return value
	}

	return ""
}

// unquote renders a JSON value as a plain string. Most rocm-smi values are
// quoted strings, but some releases emit bare numbers.
func unquote(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}

	return strings.TrimSpace(string(raw))
}

func uint32Of(v string) *uint32 {
	if v == "" {
		return nil
	}

	i, err := strconv.ParseUint(strings.Fields(v)[0], 10, 32)
	if err != nil {
		return nil
	}

	u := uint32(i)

	return &u
}

func float64Of(v string) *float64 {
	if v == "" {
		return nil
	}

	f, err := strconv.ParseFloat(strings.Fields(v)[0], 64)
	if err != nil {
		return nil
	}

	return &f
}

// mib converts a byte counter to MiB.
func mib(v string) *uint64 {
	if v == "" {
		return nil
	}

	b, err := strconv.ParseUint(strings.Fields(v)[0], 10, 64)
	if err != nil {
		return nil
	}

	m := b / bytesPerMiB

	return &m
}

// clock parses a rocm-smi clock reading, which is printed with its unit and
// parentheses, such as "(1200Mhz)".
func clock(v string) *uint32 {
	if v == "" {
		return nil
	}

	s := strings.Trim(v, "()")
	s = strings.TrimSuffix(strings.TrimSuffix(s, "Mhz"), "MHz")

	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return nil
	}

	u := uint32(n)

	return &u
}

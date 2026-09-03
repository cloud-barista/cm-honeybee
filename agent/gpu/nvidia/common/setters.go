// Package common holds the value normalization shared by every nvidia-smi
// schema parser. nvidia-smi reports a reading it cannot take as "N/A" or
// "Not Supported" rather than omitting it, and newer drivers answer some
// queries with a deprecation notice. Turning any of those into a zero would
// be indistinguishable from a real zero, so every helper here returns nil (or
// an empty string) instead.
package common

import (
	"strconv"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
)

const deprecatedValue = "requested functionality has been deprecated"

// unusable reports whether v carries no reading.
func unusable(v string) bool {
	s := strings.ToLower(strings.TrimSpace(v))

	return s == "" || s == "n/a" || s == "not supported" ||
		s == "unknown error" || s == deprecatedValue
}

// token returns the leading whitespace-separated token of v, which is where
// nvidia-smi puts the number in readings such as "42 %", "1024 MiB" or
// "26.78 W". It returns an empty string when v carries no reading.
func token(v string) string {
	if unusable(v) {
		return ""
	}

	fields := strings.Fields(v)
	if len(fields) == 0 {
		return ""
	}

	return fields[0]
}

// Str returns the whole trimmed value, or an empty string when v carries no
// reading. Unlike the numeric helpers it keeps every token, since names such
// as "NVIDIA GeForce GTX 1660" are multi-word.
func Str(v string) string {
	if unusable(v) {
		return ""
	}

	return strings.TrimSpace(v)
}

// Uint32 parses the leading token of v as an unsigned 32-bit value.
func Uint32(v string) *uint32 {
	t := token(v)
	if t == "" {
		return nil
	}

	i, err := strconv.ParseUint(t, 10, 32)
	if err != nil {
		return nil
	}

	u := uint32(i)

	return &u
}

// Uint64 parses the leading token of v as an unsigned 64-bit value.
func Uint64(v string) *uint64 {
	t := token(v)
	if t == "" {
		return nil
	}

	i, err := strconv.ParseUint(t, 10, 64)
	if err != nil {
		return nil
	}

	return &i
}

// Int32 parses the leading token of v as a signed 32-bit value. Temperatures
// are signed because the v13 schema reports them relative to a thermal limit.
func Int32(v string) *int32 {
	t := token(v)
	if t == "" {
		return nil
	}

	i, err := strconv.ParseInt(t, 10, 32)
	if err != nil {
		return nil
	}

	n := int32(i)

	return &n
}

// Float64 parses the leading token of v as a floating point value.
func Float64(v string) *float64 {
	t := token(v)
	if t == "" {
		return nil
	}

	f, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return nil
	}

	return &f
}

// LinkWidth parses a PCIe link width such as "16x" into a lane count.
func LinkWidth(v string) *uint32 {
	t := token(v)
	if t == "" {
		return nil
	}

	return Uint32(strings.TrimSuffix(t, "x"))
}

// FirstStr returns the first value that carries a reading. It resolves the
// element renames newer schemas introduce while keeping the old name as a
// fallback, such as driver_version being superseded by kmd_version.
func FirstStr(values ...string) string {
	for _, v := range values {
		if s := Str(v); s != "" {
			return s
		}
	}

	return ""
}

// FirstFloat64 returns the first value that parses as a float.
func FirstFloat64(values ...string) *float64 {
	for _, v := range values {
		if f := Float64(v); f != nil {
			return f
		}
	}

	return nil
}

// FirstUint64 returns the first value that parses as an unsigned integer.
func FirstUint64(values ...string) *uint64 {
	for _, v := range values {
		if u := Uint64(v); u != nil {
			return u
		}
	}

	return nil
}

// Percent computes used/total as a rounded percentage. It returns nil when
// either side is missing or total is zero, so an unreadable pair is never
// reported as 0%.
func Percent(used, total *uint64) *uint32 {
	if used == nil || total == nil || *total == 0 {
		return nil
	}

	p := uint32((*used*100 + *total/2) / *total)

	return &p
}

// clocksEventReasonBits maps the nvidia-smi per-reason element suffix to the
// NVML clocks-event reason bit (nvml.h nvmlClocksEventReason*), so the folded
// value matches what NVML and DCGM report.
var clocksEventReasonBits = map[string]uint64{
	"gpu_idle":                    0x0000000000000001,
	"applications_clocks_setting": 0x0000000000000002,
	"sw_power_cap":                0x0000000000000004,
	"hw_slowdown":                 0x0000000000000008,
	"sync_boost":                  0x0000000000000010,
	"sw_thermal_slowdown":         0x0000000000000020,
	"hw_thermal_slowdown":         0x0000000000000040,
	"hw_power_brake_slowdown":     0x0000000000000080,
	"display_clocks_setting":      0x0000000000000100,
}

// ClocksEventReasons folds the per-reason "Active"/"Not Active" strings into a
// single bitmask. It returns nil when no reason was reported at all, so that a
// GPU which cannot report them is not confused with one that reports every
// reason inactive (mask 0, a real reading that must be kept).
func ClocksEventReasons(reasons map[string]string) *uint64 {
	var mask uint64
	reported := false

	for name, value := range reasons {
		s := strings.TrimSpace(value)
		if unusable(s) {
			continue
		}

		reported = true
		if strings.EqualFold(s, "Active") {
			mask |= clocksEventReasonBits[name]
		}
	}

	if !reported {
		return nil
	}

	return &mask
}

// maxProcessNameLength bounds the process name kept for a GPU process.
// nvidia-smi reports the whole command line, which for a browser or a
// container runtime runs to a thousand characters and would dominate the
// collected payload without telling the reader anything more.
const maxProcessNameLength = 512

// ProcessName returns the reported command line, shortened to a bounded
// length. The marker makes a shortened value visible rather than passing a
// truncated command line off as the whole one.
func ProcessName(v string) string {
	s := Str(v)
	if len(s) <= maxProcessNameLength {
		return s
	}

	return s[:maxProcessNameLength] + "...(truncated)"
}

// ECC assembles the ECC block, or returns nil when the GPU reported neither an
// ECC mode nor a single counter. Consumer GPUs have no ECC at all, and an
// empty block would suggest the counters were read and came back zero.
func ECC(mode string, volatile, aggregate infra.NVIDIAECCErrors) *infra.NVIDIAECC {
	if mode == "" && isEmptyECC(volatile) && isEmptyECC(aggregate) {
		return nil
	}

	return &infra.NVIDIAECC{
		Mode:      mode,
		Volatile:  volatile,
		Aggregate: aggregate,
	}
}

func isEmptyECC(e infra.NVIDIAECCErrors) bool {
	return e.SRAMCorrectable == nil && e.SRAMUncorrectable == nil &&
		e.DRAMCorrectable == nil && e.DRAMUncorrectable == nil &&
		e.SingleBit == nil && e.DoubleBit == nil
}

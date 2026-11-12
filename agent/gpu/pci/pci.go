// Package pci holds helpers shared by the GPU readers for PCI addresses.
package pci

import "strings"

// NormalizeBusID writes a PCI address the way sysfs and lspci name it: a
// four-digit lower-case domain followed by bus:device.function, for example
// "0000:01:00.0". nvidia-smi prints an eight-digit upper-case domain and
// rocm-smi prints the bus in upper case on some releases, so every GPU reader
// passes its address through here to report one device under one string.
//
// A domain wider than four digits keeps its digits. A value that is not
// domain:bus:device.function comes back empty.
func NormalizeBusID(address string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(address)), ":")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return ""
	}

	domain := strings.TrimLeft(parts[0], "0")
	for len(domain) < 4 {
		domain = "0" + domain
	}

	return domain + ":" + parts[1] + ":" + parts[2]
}

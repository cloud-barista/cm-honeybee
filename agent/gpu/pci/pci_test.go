package pci

import "testing"

func TestNormalizeBusID(t *testing.T) {
	cases := map[string]string{
		"0000:01:00.0":     "0000:01:00.0", // sysfs, lspci, amd-smi
		"00000000:01:00.0": "0000:01:00.0", // nvidia-smi
		"00000002:00:00.0": "0002:00:00.0",
		"0000:1E:00.0":     "0000:1e:00.0", // rocm-smi
		" 0002:00:00.0\n":  "0002:00:00.0",
		"10000:01:00.0":    "10000:01:00.0",
		"":                 "",
		"N/A":              "",
		"01:00.0":          "",
		"0000::00.0":       "",
	}

	for in, want := range cases {
		if got := NormalizeBusID(in); got != want {
			t.Errorf("NormalizeBusID(%q) = %q, want %q", in, got, want)
		}
	}
}

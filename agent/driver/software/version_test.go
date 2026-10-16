package software

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestVersionRawOutput(t *testing.T) {
	long := strings.Repeat("a", maxVersionRawLen+100)
	// A 3-byte rune straddling the cap must be dropped whole, not cut in half.
	straddle := strings.Repeat("a", maxVersionRawLen-1) + "\u4e2d" + "tail"

	tests := []struct {
		name    string
		out     string
		want    string
		wantLen int
	}{
		{name: "empty", out: "", want: ""},
		{
			name: "trimmed",
			out:  "\n  mysqld  Ver 10.11.11-MariaDB for debian-linux-gnu on x86_64\n\n",
			want: "mysqld  Ver 10.11.11-MariaDB for debian-linux-gnu on x86_64",
		},
		{name: "exactly at cap", out: long[:maxVersionRawLen], want: long[:maxVersionRawLen]},
		{name: "capped", out: long, want: long[:maxVersionRawLen]},
		{name: "cut on rune boundary", out: straddle, wantLen: maxVersionRawLen - 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := versionRawOutput([]byte(tt.out))
			if len(got) > maxVersionRawLen {
				t.Fatalf("versionRawOutput() length %d exceeds cap %d", len(got), maxVersionRawLen)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("versionRawOutput() returned invalid UTF-8")
			}
			if tt.wantLen != 0 {
				if len(got) != tt.wantLen {
					t.Errorf("versionRawOutput() length = %d, want %d", len(got), tt.wantLen)
				}
				return
			}
			if got != tt.want {
				t.Errorf("versionRawOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}

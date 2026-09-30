// Package spiderroot sets $CBSPIDER_ROOT before cb-spider's info-store init()
// runs, which exits the process when $CBSPIDER_ROOT/meta_db cannot be created.
//
// It must import nothing but "os". Go initializes packages in import-path
// order among those whose imports are done, so a package that only needs os
// and sorts before "path/filepath" (an info-store import that needs os) is
// initialized before info-store.
package spiderroot

import "os"

const envName = "CBSPIDER_ROOT"

// defaultDir follows honeybee's own root: $CMHONEYBEE_ROOT, or ~/.cm-honeybee
// when that is unset (see cmd/cm-honeybee/main.go).
func defaultDir() (string, error) {
	if root := os.Getenv("CMHONEYBEE_ROOT"); root != "" {
		return root + "/cb-spider", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home + "/.cm-honeybee/cb-spider", nil
}

func init() {
	if os.Getenv(envName) != "" {
		return
	}
	dir, err := defaultDir()
	if err != nil {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	_ = os.Setenv(envName, dir)
}

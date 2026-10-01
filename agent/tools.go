//go:build tools

// Package tools pins the cm-centipede transx-ex module the data-migration
// inspect endpoints (filesystem, object storage, database) are built on.
// Nothing imports it yet, and `make lint`/`make build` run `go mod tidy`, which
// would drop a require that no file imports. The tools build tag keeps this
// file out of every build while tidy still counts its import. Remove the import
// here once real code imports the same package.
package tools

import (
	_ "github.com/cloud-barista/cm-centipede/transx-ex"
)

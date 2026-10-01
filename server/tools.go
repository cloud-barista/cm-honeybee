//go:build tools

// Package tools pins the cm-centipede modules the data-migration import
// (filesystem, object storage, database) is built on. Nothing imports them yet,
// and `make lint`/`make build` run `go mod tidy`, which would drop a require
// that no file imports. The tools build tag keeps this file out of every build
// while tidy still counts its imports. Remove an import here once real code
// imports the same package.
package tools

import (
	_ "github.com/cloud-barista/cm-centipede/dmdl/common-model"
	_ "github.com/cloud-barista/cm-centipede/dmdl/source-model"
	_ "github.com/cloud-barista/cm-centipede/transx-ex"
)

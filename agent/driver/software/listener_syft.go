package software

import (
	"context"
	"fmt"

	"github.com/anchore/syft/syft/pkg"
	"github.com/anchore/syft/syft/pkg/cataloger/binary"
	"github.com/anchore/syft/syft/source"
	"github.com/anchore/syft/syft/source/filesource"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/jollaman999/utils/logger"
)

// attachSyftMatches fills Listener.Syft with what syft's binary classifier
// recognizes each listener's executable as. The classifier reads the file's
// embedded version strings, so it identifies a product even when it was not
// installed by a package and its version flag is missing or ambiguous. Only the
// listeners' executables are scanned, never the whole disk, and each distinct
// executable once. A failed or empty classification leaves Syft nil.
func attachSyftMatches(listeners []software.Listener) {
	cataloger := binary.NewClassifierCataloger(binary.DefaultClassifierCatalogerConfig())
	matches := map[string]*software.SyftMatch{}

	for i := range listeners {
		exe := listeners[i].ExecutablePath
		if exe == "" {
			continue
		}

		match, done := matches[exe]
		if !done {
			var err error
			match, err = classifyExecutable(cataloger, exe)
			if err != nil {
				logger.Println(logger.DEBUG, true,
					fmt.Sprintf("Listeners: syft classification failed for %s: %v", exe, err))
			}
			matches[exe] = match
		}

		listeners[i].Syft = match
	}
}

// classifyExecutable runs cataloger over the single file at path. It returns nil
// without an error when no classifier recognizes the file. Classifier errors are
// only reported when nothing matched, because syft collects per-classifier
// failures alongside successful matches.
func classifyExecutable(cataloger pkg.Cataloger, path string) (*software.SyftMatch, error) {
	// Archive extraction is skipped: the path is an executable, and unpacking
	// e.g. a self-extracting binary into a temp dir is not wanted on a source host.
	src, err := filesource.New(filesource.Config{Path: path, SkipExtractArchive: true})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = src.Close()
	}()

	resolver, err := src.FileResolver(source.SquashedScope)
	if err != nil {
		return nil, err
	}

	pkgs, _, err := cataloger.Catalog(context.Background(), resolver)
	if len(pkgs) == 0 {
		return nil, err
	}

	return &software.SyftMatch{Name: pkgs[0].Name, Version: pkgs[0].Version}, nil
}

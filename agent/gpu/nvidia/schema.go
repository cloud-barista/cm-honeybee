package nvidia

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/gpu/nvidia/schema_v11"
	"github.com/cloud-barista/cm-honeybee/agent/gpu/nvidia/schema_v12"
	"github.com/cloud-barista/cm-honeybee/agent/gpu/nvidia/schema_v13"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
)

// latestSchema is the newest schema this agent knows how to read. Output that
// declares a newer one is parsed with it, because NVIDIA adds elements far
// more often than it renames them.
const latestSchema = "v13"

// oldestSchema is the earliest schema this agent has a parser for. Output that
// declares an older one is read with it rather than with the newest parser:
// v13 renamed power_readings to gpu_power_readings, so reading a v9 document
// with it drops the power fields without reporting anything.
const oldestSchema = "v11"

// schemaOrdinal returns the version number of a "vN" schema name. The second
// result is false when the name is not that shape, which leaves no basis for
// deciding whether it is older or newer than the parsers we have.
func schemaOrdinal(schema string) (int, bool) {
	if !strings.HasPrefix(schema, "v") {
		return 0, false
	}

	n, err := strconv.Atoi(schema[1:])
	if err != nil {
		return 0, false
	}

	return n, true
}

// readAs names the schema that was detected together with the parser actually
// used, so a reading taken with a substitute parser is not reported as if the
// agent understood the document's own version.
func readAs(detected, parser string) string {
	if detected == parser {
		return detected
	}

	return detected + " (read as " + parser + ")"
}

// detectSchema reads the schema version out of the document's DOCTYPE
// declaration, which nvidia-smi writes as nvsmi_device_<version>.dtd. Older
// drivers emit no DOCTYPE at all; those predate the rename to
// clocks_event_reasons, so they are read as v11.
func detectSchema(data []byte) string {
	decoder := xml.NewDecoder(bytes.NewReader(data))

	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return "v11"
		}

		directive, ok := token.(xml.Directive)
		if !ok {
			continue
		}

		fields := strings.Fields(string(directive))
		if len(fields) == 0 || fields[0] != "DOCTYPE" {
			continue
		}

		name := strings.Trim(fields[len(fields)-1], `" `)
		if strings.HasPrefix(name, "nvsmi_device_") && strings.HasSuffix(name, ".dtd") {
			return strings.TrimSuffix(strings.TrimPrefix(name, "nvsmi_device_"), ".dtd")
		}

		break
	}

	return "v11"
}

// parse dispatches the document to the parser for its schema version and
// reports which one was used, so the caller can record what the reading was
// taken with.
func parse(data []byte) ([]infra.NVIDIA, string, error) {
	schema := detectSchema(data)

	switch schema {
	case "v10", "v11":
		gpus, err := schema_v11.Parse(data)

		return gpus, schema, err
	case "v12":
		gpus, err := schema_v12.Parse(data)

		return gpus, schema, err
	case latestSchema:
		gpus, err := schema_v13.Parse(data)

		return gpus, schema, err
	}

	// The version has no parser of its own. Pick by which side of our range it
	// falls on: a newer document mostly adds elements the newest parser can
	// ignore, while an older one predates renames the newest parser depends on.
	// Either way, report the version found alongside the parser used.
	if n, ok := schemaOrdinal(schema); ok && n < 11 {
		gpus, err := schema_v11.Parse(data)

		return gpus, readAs(schema, oldestSchema), err
	}

	gpus, err := schema_v13.Parse(data)

	return gpus, readAs(schema, latestSchema), err
}

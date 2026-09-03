package nvidia

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
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

	// An unknown version is newer than anything we know about, so read it with
	// the newest parser and say which one was actually used.
	gpus, err := schema_v13.Parse(data)

	return gpus, latestSchema, err
}

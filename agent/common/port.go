package common

import (
	"strconv"

	"github.com/jollaman999/utils/fileutil"
)

// WriteListenPort publishes the port the agent's API is actually listening on,
// next to the agent's uuid file. With a dynamic port (listen.port: 0) the port
// is only known after the listener is created and changes on every restart, so
// cm-honeybee reads this file over SSH instead of assuming a fixed port.
func WriteListenPort(port int) error {
	return fileutil.WriteFile(RootPath+"/port", strconv.Itoa(port))
}

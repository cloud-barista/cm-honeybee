package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloud-barista/cm-honeybee/server/common"
)

// TestReadConfigIgnoresOldSpiderBlock loads a cm-honeybee.yaml written while
// honeybee still called a cb-spider server. The spider block is no longer read
// and must not stop the server from starting.
func TestReadConfigIgnoresOldSpiderBlock(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "conf"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := `cm-honeybee:
    listen:
        port: 8081
    agent:
        port: 8082
        request_timeout: 600
    spider:
        endpoint: http://cb-spider:1024/spider
        username: default
        password: default
    openbao:
        address: http://openbao:8200
`
	if err := os.WriteFile(filepath.Join(root, "conf", cmHoneybeeConfigFile), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}

	savedRoot := common.RootPath
	common.RootPath = root
	t.Cleanup(func() { common.RootPath = savedRoot })
	for _, env := range []string{"HONEYBEE_LISTEN_PORT", "HONEYBEE_AGENT_PORT",
		"HONEYBEE_AGENT_REQUEST_TIMEOUT", "HONEYBEE_VAULT_ADDR"} {
		t.Setenv(env, "")
	}

	CMHoneybeeConfig = cmHoneybeeConfig{}
	if err := readCMHoneybeeConfigFile(); err != nil {
		t.Fatalf("old config with a spider block failed to load: %v", err)
	}

	c := CMHoneybeeConfig.CMHoneybee
	if c.Listen.Port != "8081" || c.Agent.Port != "8082" || c.Agent.RequestTimeout != "600" ||
		c.OpenBao.Address != "http://openbao:8200" {
		t.Fatalf("unexpected config after load: %+v", c)
	}
}

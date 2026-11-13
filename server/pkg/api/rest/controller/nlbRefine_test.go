package controller

import (
	"reflect"
	"testing"

	inframodel "github.com/cloud-barista/cm-beetle/imdl/on-premise-model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
)

func TestParseHAProxyBind(t *testing.T) {
	cases := []struct {
		bind string
		addr string
		port int
		ok   bool
	}{
		{bind: "*:80", addr: "*", port: 80, ok: true},
		{bind: ":443 ssl crt /etc/haproxy/site.pem", addr: "*", port: 443, ok: true},
		{bind: "0.0.0.0:9999", addr: "*", port: 9999, ok: true},
		{bind: ":::8080", addr: "*", port: 8080, ok: true},
		{bind: "10.0.1.50:8080", addr: "10.0.1.50", port: 8080, ok: true},
		{bind: "10.0.1.50:8080,10.0.1.50:8081", addr: "10.0.1.50", port: 8080, ok: true},
		{bind: "[fd00::1]:80", addr: "fd00::1", port: 80, ok: true},
		{bind: "", ok: false},
		{bind: "10.0.1.50", ok: false},
		{bind: "*:80-85", ok: false},
	}

	for _, c := range cases {
		addr, port, ok := parseHAProxyBind(c.bind)
		if ok != c.ok || addr != c.addr || port != c.port {
			t.Errorf("parseHAProxyBind(%q) = (%q, %d, %v), want (%q, %d, %v)",
				c.bind, addr, port, ok, c.addr, c.port, c.ok)
		}
	}
}

func TestHAProxyToNLBs(t *testing.T) {
	haproxy := infra.HAProxy{
		Defaults: map[string]string{"mode": "http", "balance": "leastconn"},
		Frontends: []infra.HAProxyFrontend{
			{Name: "web", Bind: "*:80", DefaultBackend: "web_servers", Options: map[string]string{}},
			{Name: "api", Bind: "10.0.0.10:8443", DefaultBackend: "api_servers", Options: map[string]string{"mode": "tcp"}},
			{Name: "orphan", Bind: "*:81", DefaultBackend: "missing", Options: map[string]string{}},
		},
		Backends: []infra.HAProxyBackend{
			{
				Name:    "web_servers",
				Balance: "roundrobin",
				Options: map[string]string{},
				Servers: []infra.HAProxyServer{
					{Name: "web1", Address: "10.0.0.11:8080", Options: "check weight 10"},
					{Name: "web2", Address: "10.0.0.12", Options: "check"},
				},
			},
			{
				Name:    "api_servers",
				Options: map[string]string{},
				Servers: []infra.HAProxyServer{
					{Name: "api1", Address: "10.0.0.21:9000"},
				},
			},
		},
		Listens: []infra.HAProxyListen{
			{
				Name:    "db",
				Bind:    "*:3306",
				Balance: "source",
				Options: map[string]string{"mode": "tcp"},
				Servers: []infra.HAProxyServer{{Name: "db1", Address: "10.0.0.31:3306"}},
			},
			{Name: "empty", Bind: "*:5000", Options: map[string]string{}},
		},
	}

	want := []inframodel.NlbProperty{
		{
			HostMachineId: "m-1",
			Software:      "haproxy",
			Listener:      inframodel.NlbListenerProperty{BindAddress: "*", Port: 80, Protocol: "tcp"},
			Backend: inframodel.NlbBackendProperty{
				Name:     "web_servers",
				Balance:  "roundrobin",
				Protocol: "http",
				Servers: []inframodel.NlbServerProperty{
					{Name: "web1", IP: "10.0.0.11", Port: 8080, Weight: 10},
					{Name: "web2", IP: "10.0.0.12", Port: 80},
				},
			},
			HealthCheck: inframodel.NlbHealthCheckProperty{Enabled: true},
		},
		{
			HostMachineId: "m-1",
			Software:      "haproxy",
			Listener:      inframodel.NlbListenerProperty{BindAddress: "10.0.0.10", Port: 8443, Protocol: "tcp"},
			Backend: inframodel.NlbBackendProperty{
				Name:     "api_servers",
				Balance:  "leastconn",
				Protocol: "tcp",
				Servers:  []inframodel.NlbServerProperty{{Name: "api1", IP: "10.0.0.21", Port: 9000}},
			},
		},
		{
			HostMachineId: "m-1",
			Software:      "haproxy",
			Listener:      inframodel.NlbListenerProperty{BindAddress: "*", Port: 3306, Protocol: "tcp"},
			Backend: inframodel.NlbBackendProperty{
				Name:     "db",
				Balance:  "source",
				Protocol: "tcp",
				Servers:  []inframodel.NlbServerProperty{{Name: "db1", IP: "10.0.0.31", Port: 3306}},
			},
		},
	}

	got := haproxyToNLBs(haproxy, "m-1")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("haproxyToNLBs mismatch\n got: %+v\nwant: %+v", got, want)
	}
}

func TestHAProxyToNLBsSingleBackend(t *testing.T) {
	haproxy := infra.HAProxy{
		Frontends: []infra.HAProxyFrontend{{Name: "fe", Bind: "*:80", Options: map[string]string{}}},
		Backends: []infra.HAProxyBackend{{
			Name:    "only",
			Options: map[string]string{},
			Servers: []infra.HAProxyServer{{Name: "s1", Address: "10.0.0.11:80"}},
		}},
	}

	got := haproxyToNLBs(haproxy, "m-1")
	if len(got) != 1 || got[0].Backend.Name != "only" || got[0].Backend.Balance != "roundrobin" {
		t.Fatalf("haproxyToNLBs = %+v, want one NLB on backend \"only\" with roundrobin", got)
	}
}

func TestHAProxyToNLBsNotCollected(t *testing.T) {
	if got := haproxyToNLBs(infra.HAProxy{}, "m-1"); got != nil {
		t.Errorf("haproxyToNLBs(empty) = %+v, want nil", got)
	}
}

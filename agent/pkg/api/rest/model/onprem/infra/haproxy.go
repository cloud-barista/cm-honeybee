package infra

// HAProxy represents HAProxy load balancer information
type HAProxy struct {
	Version    string            `json:"version"`
	ConfigPath string            `json:"config_path"`
	Global     map[string]string `json:"global"`
	Defaults   map[string]string `json:"defaults"`
	Frontends  []HAProxyFrontend `json:"frontends"`
	Backends   []HAProxyBackend  `json:"backends"`
	Listens    []HAProxyListen   `json:"listens"`
	Errors     []string          `json:"errors"`
}

// HAProxyFrontend represents a frontend configuration
type HAProxyFrontend struct {
	Name           string            `json:"name"`
	Bind           string            `json:"bind"`
	DefaultBackend string            `json:"default_backend,omitempty"`
	Options        map[string]string `json:"options"`
}

// HAProxyBackend represents a backend configuration
type HAProxyBackend struct {
	Name    string            `json:"name"`
	Balance string            `json:"balance,omitempty"`
	Options map[string]string `json:"options"`
	Servers []HAProxyServer   `json:"servers"`
}

// HAProxyListen represents a listen configuration (combined frontend/backend)
type HAProxyListen struct {
	Name    string            `json:"name"`
	Bind    string            `json:"bind"`
	Balance string            `json:"balance,omitempty"`
	Options map[string]string `json:"options"`
	Servers []HAProxyServer   `json:"servers"`
}

// HAProxyServer represents a backend server
type HAProxyServer struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Options string `json:"options,omitempty"`
}

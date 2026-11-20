package route

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
)

// /source_group/types is a static path next to /source_group/:sgId. Echo
// tries a static segment before a parameter, so the types route is reached
// and a group ID, a UUID, still reaches the group routes.
func TestSourceGroupTypesRoute(t *testing.T) {
	e := echo.New()
	RegisterSourceGroup(e)
	RegisterCSP(e)

	tests := []struct {
		method, path, want string
	}{
		{http.MethodGet, "/honeybee/source_group/types", "/honeybee/source_group/types"},
		{http.MethodGet, "/honeybee/source_group/6f1c2a3e-0000-4000-8000-000000000001", "/honeybee/source_group/:sgId"},
		{http.MethodPut, "/honeybee/source_group/6f1c2a3e-0000-4000-8000-000000000001", "/honeybee/source_group/:sgId"},
		{http.MethodGet, "/honeybee/source_group/6f1c2a3e-0000-4000-8000-000000000001/region", "/honeybee/source_group/:sgId/region"},
		{http.MethodGet, "/honeybee/csp/aws", "/honeybee/csp/:name"},
		{http.MethodPost, "/honeybee/csp/aws/region", "/honeybee/csp/:name/region"},
	}
	for _, tt := range tests {
		c := e.NewContext(nil, nil)
		e.Router().Find(tt.method, tt.path, c)
		if got := c.Path(); got != tt.want {
			t.Errorf("%s %s routed to %q, want %q", tt.method, tt.path, got, tt.want)
		}
	}
}

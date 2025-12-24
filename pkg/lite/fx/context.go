package fx

import (
	"encoding/json"
	"net/http"

	"github.com/fluxorio/fluxor/pkg/lite/core"
)

// JSON is a convenience alias for JSON objects.
type JSON map[string]any

// Context is a thin wrapper around http request/response plus the core context.
type Context struct {
	W       http.ResponseWriter
	R       *http.Request
	coreCtx *core.FluxorContext
}

func NewContext(w http.ResponseWriter, r *http.Request, cCtx *core.FluxorContext) *Context {
	return &Context{W: w, R: r, coreCtx: cCtx}
}

func (c *Context) Ok(data any) error {
	c.W.Header().Set("Content-Type", "application/json")
	c.W.WriteHeader(http.StatusOK)
	return json.NewEncoder(c.W).Encode(data)
}

func (c *Context) Error(code int, msg string) error {
	c.W.Header().Set("Content-Type", "application/json")
	c.W.WriteHeader(code)
	return json.NewEncoder(c.W).Encode(JSON{"error": msg})
}

func (c *Context) Worker() *core.WorkerPool  { return c.coreCtx.Worker() }
func (c *Context) Bus() *core.Bus            { return c.coreCtx.Bus() }
func (c *Context) Core() *core.FluxorContext { return c.coreCtx }

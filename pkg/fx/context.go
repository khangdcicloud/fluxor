package fx

import (
	"encoding/json"
	"net/http"
	"github.com/khangdcicloud/fluxor/pkg/core"
)

// JSON alias cho map (Dev UX)
type JSON map[string]any

// Context wrapper cho HTTP Handler
type Context struct {
	W       http.ResponseWriter
	R       *http.Request
	coreCtx *core.FluxorContext // Giấu hàng context gốc
}

func NewContext(w http.ResponseWriter, r *http.Request, cCtx *core.FluxorContext) *Context {
	return &Context{W: w, R: r, coreCtx: cCtx}
}

// Dev UX: Trả về OK JSON
func (c *Context) Ok(data any) error {
	c.W.Header().Set("Content-Type", "application/json")
	c.W.WriteHeader(200)
	return json.NewEncoder(c.W).Encode(data)
}

// Dev UX: Trả về Error
func (c *Context) Error(code int, msg string) error {
	c.W.Header().Set("Content-Type", "application/json")
	c.W.WriteHeader(code)
	return json.NewEncoder(c.W).Encode(JSON{"error": msg})
}

// Accessor tới Core
func (c *Context) Worker() *core.WorkerPool { return c.coreCtx.Worker() }
func (c *Context) Bus() *core.Bus           { return c.coreCtx.Bus() }

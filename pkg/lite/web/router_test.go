package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fluxorio/fluxor/pkg/lite/core"
	"github.com/fluxorio/fluxor/pkg/lite/fx"
	"github.com/fluxorio/fluxor/pkg/lite/web"
)

func TestRouter_Handle_NotFound(t *testing.T) {
	r := web.NewRouter()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/missing", nil)
	rec := httptest.NewRecorder()
	ctx := fx.NewContext(rec, req, core.NewFluxorContext(req.Context(), core.NewBus(), core.NewWorkerPool(1, 10), "test"))

	if err := r.Handle(ctx); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if rec.Code != 404 {
		t.Fatalf("status=%d, want 404", rec.Code)
	}
}

package verticles

import (
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/examples/load-balancing/contracts"
	"github.com/fluxorio/fluxor/pkg/core"
)

// WorkerVerticle processes work requests
type WorkerVerticle struct {
	*core.BaseVerticle
	id     string
	logger core.Logger
}

// NewWorkerVerticle creates a new worker
func NewWorkerVerticle(id string) *WorkerVerticle {
	return &WorkerVerticle{
		BaseVerticle: core.NewBaseVerticle(fmt.Sprintf("worker-%s", id)),
		id:           id,
		logger:       core.NewDefaultLogger(),
	}
}

// doStart initializes the worker
func (v *WorkerVerticle) doStart(ctx core.FluxorContext) error {
	v.logger.Info(fmt.Sprintf("Worker %s starting...", v.id))

	// Register consumer with a unique address (for direct addressing if needed)
	// AND a shared group address if we wanted queue-group style LB (but we are doing manual LB here)

	// For manual LB from Master, we listen on our specific address
	myAddress := fmt.Sprintf("%s.%s", contracts.WorkerAddress, v.id)

	v.Consumer(myAddress).Handler(func(c core.FluxorContext, msg core.Message) error {
		var req contracts.WorkRequest
		if err := msg.DecodeBody(&req); err != nil {
			return msg.Fail(400, "Invalid body")
		}

		v.logger.Info(fmt.Sprintf("Worker %s processing job %s", v.id, req.ID))

		// Simulate work
		time.Sleep(100 * time.Millisecond)

		resp := contracts.WorkResponse{
			ID:     req.ID,
			Result: fmt.Sprintf("Processed %s", req.Payload),
			Worker: v.id,
		}
		return msg.Reply(resp)
	})

	return nil
}

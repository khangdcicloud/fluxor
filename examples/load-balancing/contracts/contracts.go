package contracts

// Address definitions
const (
	WorkerAddress = "worker.process"
)

// WorkRequest represents a job for the worker
type WorkRequest struct {
	ID      string `json:"id"`
	Payload string `json:"payload"`
}

// WorkResponse represents the result
type WorkResponse struct {
	ID     string `json:"id"`
	Result string `json:"result"`
	Worker string `json:"worker"`
}

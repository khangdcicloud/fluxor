package core

// BaseService provides a Java-style abstract base class for service verticles
// Services typically handle request-reply patterns and provide business logic
// Combines BaseVerticle with service-specific patterns
type BaseService struct {
	*BaseVerticle

	// Service address (where this service listens)
	address string

	// Request handler
	requestHandler MessageHandler
}

// NewBaseService creates a new BaseService
func NewBaseService(name, address string) *BaseService {
	return &BaseService{
		BaseVerticle: NewBaseVerticle(name),
		address:      address,
	}
}

// doStart overrides BaseVerticle.doStart to register service handler
func (bs *BaseService) doStart(ctx FluxorContext) error {
	// Register service handler
	consumer := bs.Consumer(bs.address)
	consumer.Handler(bs.handleRequest)

	return nil
}

// handleRequest handles incoming service requests
func (bs *BaseService) handleRequest(ctx FluxorContext, msg Message) error {
	// If custom handler is set, use it
	if bs.requestHandler != nil {
		return bs.requestHandler(ctx, msg)
	}

	// Otherwise, call hook method
	return bs.doHandleRequest(ctx, msg)
}

// doHandleRequest is a hook method for subclasses to implement
// Default implementation returns error
func (bs *BaseService) doHandleRequest(ctx FluxorContext, msg Message) error {
	return &EventBusError{Code: "NOT_IMPLEMENTED", Message: "doHandleRequest must be implemented by subclass"}
}

// SetRequestHandler sets a custom request handler
func (bs *BaseService) SetRequestHandler(handler MessageHandler) {
	bs.requestHandler = handler
}

// Address returns the service address
func (bs *BaseService) Address() string {
	return bs.address
}

// Reply is a convenience method to reply to service requests
func (bs *BaseService) Reply(msg Message, body interface{}) error {
	return msg.Reply(body)
}

// Fail is a convenience method to fail service requests
func (bs *BaseService) Fail(msg Message, code int, message string) error {
	return msg.Fail(code, message)
}

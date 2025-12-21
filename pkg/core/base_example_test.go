package core_test

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/core"
)

// ExampleBaseVerticle demonstrates using BaseVerticle (Java-style abstract class)
func ExampleBaseVerticle() {
	// Create a custom verticle by embedding BaseVerticle
	type MyVerticle struct {
		*core.BaseVerticle
	}

	// Create and use the verticle
	verticle := &MyVerticle{
		BaseVerticle: core.NewBaseVerticle("my-verticle"),
	}

	// Note: In real usage, you would override doStart method:
	// func (v *MyVerticle) doStart(ctx core.FluxorContext) error {
	//     consumer := v.Consumer("my.address")
	//     consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
	//         return msg.Reply("processed")
	//     })
	//     return nil
	// }

	ctx := context.Background()
	vertx := core.NewVertx(ctx)

	// Deploy verticle
	vertx.DeployVerticle(verticle)
}

// ExampleBaseService demonstrates using BaseService (Java-style service pattern)
func ExampleBaseService() {
	// Create a custom service by embedding BaseService
	type UserService struct {
		*core.BaseService
	}

	// Create service
	service := &UserService{
		BaseService: core.NewBaseService("user-service", "user.service"),
	}

	// Note: In real usage, you would override doHandleRequest method:
	// func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     userID := msg.Body().(string)
	//     userData := map[string]interface{}{
	//         "id":   userID,
	//         "name": "John Doe",
	//     }
	//     return s.Reply(msg, userData)
	// }

	ctx := context.Background()
	vertx := core.NewVertx(ctx)

	// Deploy service
	vertx.DeployVerticle(service)
}

// ExampleBaseHandler demonstrates using BaseHandler (Java-style handler pattern)
func ExampleBaseHandler() {
	// Create a custom handler by embedding BaseHandler
	type UserHandler struct {
		*core.BaseHandler
	}

	// Create handler
	handler := &UserHandler{
		BaseHandler: core.NewBaseHandler("user-handler"),
	}

	// Note: In real usage, you would override doHandle method:
	// func (h *UserHandler) doHandle(ctx core.FluxorContext, msg core.Message) error {
	//     var request map[string]interface{}
	//     if err := h.DecodeBody(msg, &request); err != nil {
	//         return h.Fail(msg, 400, "Invalid request")
	//     }
	//     userID := request["id"].(string)
	//     userData := map[string]interface{}{
	//         "id":   userID,
	//         "name": "John Doe",
	//     }
	//     return h.Reply(msg, userData)
	// }

	_ = handler // Use handler
}

// ExampleBaseComponent demonstrates using BaseComponent (Java-style component pattern)
func ExampleBaseComponent() {
	// Create a custom component by embedding BaseComponent
	type DatabaseComponent struct {
		*core.BaseComponent
		connection string
	}

	// Create component
	component := &DatabaseComponent{
		BaseComponent: core.NewBaseComponent("database"),
	}

	// Note: In real usage, you would override doStart and doStop methods:
	// func (c *DatabaseComponent) doStart(ctx core.FluxorContext) error {
	//     c.connection = "connected"
	//     return nil
	// }
	// func (c *DatabaseComponent) doStop(ctx core.FluxorContext) error {
	//     c.connection = "disconnected"
	//     return nil
	// }

	_ = component // Use component
}

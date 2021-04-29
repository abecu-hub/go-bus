package mutation

import "github.com/abecu-hub/go-bus/pkg/servicebus"

//Add a header to the current context
func Header(key string, value interface{}) servicebus.OutgoingMutation {
	return func(ctx *servicebus.OutgoingMessageContext) {
		ctx.Headers[key] = value
	}
}

//Replace the current headers with a new map
func SetHeaders(headers map[string]interface{}) servicebus.OutgoingMutation {
	return func(ctx *servicebus.OutgoingMessageContext) {
		ctx.Headers = headers
	}
}

//Set the priority of the current context
func Priority(priority uint8) servicebus.OutgoingMutation {
	return func(ctx *servicebus.OutgoingMessageContext) {
		ctx.Priority = priority
	}
}

//Add a version to the current context
func Version(version string) servicebus.OutgoingMutation {
	return func(ctx *servicebus.OutgoingMessageContext) {
		ctx.Version = version
	}
}

package mutation

import "github.com/abecu-hub/go-bus/pkg/servicebus"

func Header(key string, value interface{}) servicebus.OutgoingMutation {
	return func(ctx *servicebus.OutgoingMessageContext) {
		ctx.Headers[key] = value
	}
}

func Priority(priority uint8) servicebus.OutgoingMutation {
	return func(ctx *servicebus.OutgoingMessageContext) {
		ctx.Priority = priority
	}
}

func Version(version string) servicebus.OutgoingMutation {
	return func(ctx *servicebus.OutgoingMessageContext) {
		ctx.Version = version
	}
}

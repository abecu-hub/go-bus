package mutation

import "dev.azure.com/finorun/Playground/_git/go-bus.git/pkg/serviceBus"

func Header(key string, value interface{}) serviceBus.OutgoingMutation {
	return func(ctx *serviceBus.OutgoingMessageContext) {
		ctx.Headers[key] = value
	}
}

func Priority(priority uint8) serviceBus.OutgoingMutation {
	return func(ctx *serviceBus.OutgoingMessageContext) {
		ctx.Priority = priority
	}
}

func Version(version string) serviceBus.OutgoingMutation {
	return func(ctx *serviceBus.OutgoingMessageContext) {
		ctx.Version = version
	}
}

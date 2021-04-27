package servicebus

import (
	"testing"
)

type TestTransport struct {
	routes map[string]interface{}
}

func (TestTransport) Start(endpointName string) error {
	panic("implement me")
}

func (t *TestTransport) RegisterRouting(route string) error {
	if t.routes == nil {
		t.routes = make(map[string]interface{})
	}
	t.routes[route] = route
	return nil
}

func (TestTransport) UnregisterRouting(route string) error {
	panic("implement me")
}

func (TestTransport) Publish(message *OutgoingMessageContext) error {
	panic("implement me")
}

func (TestTransport) Send(destination string, command *OutgoingMessageContext) error {
	panic("implement me")
}

func (TestTransport) SendLocal(command *OutgoingMessageContext) error {
	panic("implement me")
}

func (TestTransport) MessageReceived(contexts chan *IncomingMessageContext) chan *IncomingMessageContext {
	panic("implement me")
}

type MyMessage struct {
	Name string
}

func TestCreateOutgoingContext(t *testing.T) {
	serviceName := "ServiceName"
	endpoint := Create(serviceName, &TestTransport{}).(*ServiceBusEndpoint)
	payload := &MyMessage{Name: "HelloWorld"}
	ctx := endpoint.createMessageContext("MyMessage", payload, nil)
	if ctx.Origin != serviceName {
		t.Errorf("Origin not set!")
	}

	if ctx.MessageId == "" {
		t.Errorf("MessageId not set!")
	}

	if ctx.CorrelationId == "" {
		t.Errorf("CorrelationId not set!")
	}

	if ctx.Type == "" {
		t.Errorf("Type not set!")
	}

	if ctx.Payload != payload {
		t.Errorf("Payload not set!")
	}
}

func TestMutateOutgoingContext(t *testing.T) {
	serviceName := "ServiceName"
	messageName := "MyMessage"
	endpoint := Create(serviceName, &TestTransport{}).(*ServiceBusEndpoint)
	endpoint.Message(messageName).
		AsOutgoing().
		Mutate(func(ctx *OutgoingMessageContext) {
			ctx.Origin = "NewOrigin"
		}).
		Mutate(func(ctx *OutgoingMessageContext) {
			ctx.Version = "V1"
		})

	mutationHeader := func(ctx *OutgoingMessageContext) {
		ctx.Headers["MyHeader"] = "value"
	}
	mutationOverride := func(ctx *OutgoingMessageContext) {
		ctx.Version = "V2"
	}

	payload := &MyMessage{Name: "HelloWorld"}
	ctx := endpoint.createMessageContext(messageName, payload, []OutgoingMutation{mutationHeader, mutationOverride})

	if ctx.Origin != "NewOrigin" {
		t.Errorf("Mutation in message configuration failed.")
	}

	if ctx.Headers["MyHeader"] != "value" {
		t.Errorf("Mutation in publish/send/sendlocal failed.")
	}

	if ctx.Version != "V2" {
		t.Errorf("Mutation override failed!")
	}
}

func TestMessageValidationWithValidMessage(t *testing.T) {
	ctx := &IncomingMessageContext{
		Origin:        "MyService",
		Type:          "MyType",
		CorrelationId: "MyCorrelationId",
		MessageId:     "MyMessageId",
	}
	err := ctx.validate()
	if err != nil {
		t.Errorf("Message validation failed although message is valid.")
	}
}

func TestMessageValidationWithInvalidMessage(t *testing.T) {
	ctx := &IncomingMessageContext{
		Origin:        "MyService",
		Type:          "MyType",
		CorrelationId: "MyCorrelationId",
		MessageId:     "MyMessageId",
	}

	ctx.CorrelationId = ""
	err := ctx.validate()
	if err == nil {
		t.Errorf("Message validation did not fail with empty or unset CorrelationId.")
	}
	ctx.CorrelationId = "MyCorrelationId"

	ctx.Type = ""
	err = ctx.validate()
	if err == nil {
		t.Errorf("Message validation did not fail with empty or unset Type.")
	}
	ctx.Type = "MyType"

	ctx.Origin = ""
	err = ctx.validate()
	if err == nil {
		t.Errorf("Message validation did not fail with empty or unset Origin.")
	}
	ctx.Origin = "MyService"

	ctx.MessageId = ""
	err = ctx.validate()
	if err == nil {
		t.Errorf("Message validation did not fail with empty or unset MessageId.")
	}
}

func TestIncomingMessageConfigurationCreated(t *testing.T) {
	serviceName := "ServiceName"
	messageName := "MyMessage"
	endpoint := Create(serviceName, &TestTransport{}).(*ServiceBusEndpoint)
	endpoint.Message(messageName).AsIncoming()
	if endpoint.incomingMessages[messageName] == nil {
		t.Errorf("Failed to create incoming message configuration.")
	}
	endpoint.Message(messageName).
		AsIncoming().
		Handle(func(ctx *IncomingMessageContext) {})

	if endpoint.incomingMessages[messageName].handler == nil {
		t.Errorf("Failed to attach handler to message configuration or to register route in transport.")
	}
}

func TestOutgoingMessageConfigurationCreated(t *testing.T) {
	serviceName := "ServiceName"
	messageName := "MyMessage"
	endpoint := Create(serviceName, &TestTransport{}).(*ServiceBusEndpoint)
	endpoint.Message(messageName).AsOutgoing()
	if endpoint.outgoingMessages[messageName] == nil {
		t.Errorf("Failed to create outgoing message configuration.")
	}
}

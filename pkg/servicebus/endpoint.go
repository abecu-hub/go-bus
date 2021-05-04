package servicebus

import (
	"github.com/abecu-hub/go-bus/pkg/servicebus/saga"
	"github.com/google/uuid"
	"time"
)

type Endpoint interface {
	Message(messageType string) *MessageConfiguration
	Start() error
	Publish(messageType string, msg interface{}, options ...OutgoingMutation) error
	Send(messageType string, destination string, msg interface{}, options ...OutgoingMutation) error
	SendLocal(messageType string, msg interface{}, options ...OutgoingMutation) error
	SagaStore() saga.Store
}

type ServiceBusEndpoint struct {
	name             string
	transport        Transport
	incomingMessages map[string]*IncomingMessageConfiguration
	outgoingMessages map[string]*OutgoingMessageConfiguration
	sagaStore        saga.Store
}

/*
Create a new ServiceBusEndpoint by providing a name and a transport.
*/
func Create(name string, transport Transport, options ...func(endpoint *ServiceBusEndpoint)) Endpoint {
	endpoint := &ServiceBusEndpoint{
		name:             name,
		transport:        transport,
		incomingMessages: make(map[string]*IncomingMessageConfiguration),
		outgoingMessages: make(map[string]*OutgoingMessageConfiguration),
	}

	for _, option := range options {
		option(endpoint)
	}

	return endpoint
}

func (endpoint *ServiceBusEndpoint) createOrGetIncomingMessageConfig(mc *MessageConfiguration) *IncomingMessageConfiguration {
	if endpoint.incomingMessages[mc.messageType] == nil {
		endpoint.incomingMessages[mc.messageType] = &IncomingMessageConfiguration{
			messageConfiguration: mc,
		}
	}
	return endpoint.incomingMessages[mc.messageType]
}

func (endpoint *ServiceBusEndpoint) createOrGetOutgoingMessageConfig(mc *MessageConfiguration) *OutgoingMessageConfiguration {
	if endpoint.outgoingMessages[mc.messageType] == nil {
		endpoint.outgoingMessages[mc.messageType] = &OutgoingMessageConfiguration{
			messageConfiguration: mc,
		}
	}
	return endpoint.outgoingMessages[mc.messageType]
}

//Declare a message configuration.
func (endpoint *ServiceBusEndpoint) Message(messageType string) *MessageConfiguration {
	return &MessageConfiguration{
		messageType: messageType,
		endpoint:    endpoint,
	}
}

/*
Start receiving/sending messages with the ServiceBus and setup transport topology.
*/
func (endpoint *ServiceBusEndpoint) Start() error {
	if endpoint.transport == nil {
		panic("ServiceBusEndpoint has no transport configured.")
	}

	go endpoint.handleReceivedMessages()

	err := endpoint.transport.Start(endpoint.name)
	if err != nil {
		return err
	}

	return nil
}

/*
Publish a message to all subscribers
*/
func (endpoint *ServiceBusEndpoint) Publish(messageType string, msg interface{}, options ...OutgoingMutation) error {
	ctx := endpoint.createMessageContext(messageType, msg, options)
	if ctx.IsCancelled {
		return nil
	}

	cfg := endpoint.outgoingMessages[messageType]
	if cfg == nil || cfg.retryConfiguration == nil {
		return endpoint.transport.Publish(ctx)
	}

	err := cfg.retryConfiguration.Execute(func() error { return endpoint.transport.Publish(ctx) })
	if err != nil {
		return err
	}

	return nil
}

/*
Send a message to a specific ServiceBusEndpoint
*/
func (endpoint *ServiceBusEndpoint) Send(messageType string, destination string, msg interface{}, options ...OutgoingMutation) error {
	ctx := endpoint.createMessageContext(messageType, msg, options)
	if ctx.IsCancelled {
		return nil
	}

	cfg := endpoint.outgoingMessages[messageType]
	if cfg == nil || cfg.retryConfiguration == nil {
		return endpoint.transport.Send(destination, ctx)
	}

	err := cfg.retryConfiguration.Execute(func() error { return endpoint.transport.Send(destination, ctx) })
	if err != nil {
		return err
	}
	return nil
}

/*
Send the message to the local ServiceBusEndpoint
*/
func (endpoint *ServiceBusEndpoint) SendLocal(messageType string, msg interface{}, options ...OutgoingMutation) error {
	ctx := endpoint.createMessageContext(messageType, msg, options)
	if ctx.IsCancelled {
		return nil
	}

	cfg := endpoint.outgoingMessages[messageType]
	if cfg == nil || cfg.retryConfiguration == nil {
		return endpoint.transport.SendLocal(ctx)
	}

	err := cfg.retryConfiguration.Execute(func() error { return endpoint.transport.SendLocal(ctx) })
	if err != nil {
		return err
	}
	return nil
}

func (endpoint *ServiceBusEndpoint) SagaStore() saga.Store {
	return endpoint.sagaStore
}

func (endpoint *ServiceBusEndpoint) handleReceivedMessages() {
	received := endpoint.transport.MessageReceived(make(chan *IncomingMessageContext))
	for {
		msg := <-received
		msg.setEndpoint(endpoint)
		err := msg.validate()
		if err != nil {
			//If message could not be validated, it is most likely not produced by the go-bus service bus, so we discard the message by default.
			msg.Discard()
			return
		}

		if _, ok := endpoint.incomingMessages[msg.Type]; !ok {
			//unregister routing from transport as there is no handler any more and discard the message by default.
			_ = endpoint.transport.UnregisterRouting(msg.Type)
			msg.Discard()
			return
		}

		err = endpoint.startSagas(msg)
		if err != nil {
			//If saga could not be checked/created it is most likely a transient issue with database access, so the message will be requeued by default.
			msg.Retry()
			return
		}

		for _, handler := range endpoint.incomingMessages[msg.Type].handler {
			handler(msg)
		}
	}
}

func (endpoint *ServiceBusEndpoint) startSagas(ctx *IncomingMessageContext) error {
	config := endpoint.incomingMessages[ctx.Type]
	for _, s := range config.sagas {
		exists, err := endpoint.sagaStore.SagaExists(ctx.CorrelationId, s)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		err = endpoint.sagaStore.CreateSaga(ctx.CorrelationId, s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (endpoint *ServiceBusEndpoint) createMessageContext(messageType string, payload interface{}, mutations []OutgoingMutation) *OutgoingMessageContext {
	ctx := CreateOutgoingContext(endpoint)
	ctx.Payload = payload
	ctx.Type = messageType
	ctx.MessageId = uuid.New().String()
	ctx.Timestamp = time.Now().UTC()
	ctx.Origin = endpoint.name
	ctx.Headers = make(map[string]interface{})

	if _, ok := endpoint.outgoingMessages[""]; ok {
		for _, mutation := range endpoint.outgoingMessages[""].mutations {
			mutation(ctx)
		}
	}

	if _, ok := endpoint.outgoingMessages[messageType]; ok {
		for _, mutation := range endpoint.outgoingMessages[ctx.Type].mutations {
			mutation(ctx)
		}
	}

	for _, mutation := range mutations {
		mutation(ctx)
	}

	if ctx.CorrelationId == "" {
		ctx.CorrelationId = ctx.MessageId
	}

	if ctx.CorrelationTimestamp.IsZero() {
		ctx.CorrelationTimestamp = time.Now().UTC()
	}

	return ctx
}

func (cfg *RetryConfiguration) Execute(f func() error) error {
	var err error
	for i := 0; i < cfg.MaxRetries; i++ {
		err = cfg.Policy(i, f)
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}
	return nil
}

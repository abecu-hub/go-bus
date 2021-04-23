package servicebus

import (
	"errors"
	"github.com/abecu-hub/go-bus/pkg/servicebus/saga"
	"github.com/google/uuid"
	"time"
)

type Endpoint struct {
	Name             string
	Transport        Transport
	incomingMessages map[string]*IncomingMessageConfiguration
	outgoingMessages map[string]*OutgoingMessageConfiguration
	SagaStore        saga.Store
}

/*
Create a new service bus Endpoint by providing a transport e.g. RabbitMQ, MSMQ, Kafka, etc.
*/
func Create(name string, transport Transport, options ...func(endpoint *Endpoint)) *Endpoint {
	endpoint := &Endpoint{
		Name:             name,
		Transport:        transport,
		incomingMessages: make(map[string]*IncomingMessageConfiguration),
		outgoingMessages: make(map[string]*OutgoingMessageConfiguration),
	}

	for _, option := range options {
		option(endpoint)
	}

	return endpoint
}

func (endpoint *Endpoint) createOrGetIncomingMessageConfig(mc *MessageConfiguration) *IncomingMessageConfiguration {
	if endpoint.incomingMessages[mc.messageType] == nil {
		endpoint.incomingMessages[mc.messageType] = &IncomingMessageConfiguration{
			messageConfiguration: mc,
		}
	}
	return endpoint.incomingMessages[mc.messageType]
}

func (endpoint *Endpoint) createOrGetOutgoingMessageConfig(mc *MessageConfiguration) *OutgoingMessageConfiguration {
	if endpoint.outgoingMessages[mc.messageType] == nil {
		endpoint.outgoingMessages[mc.messageType] = &OutgoingMessageConfiguration{
			messageConfiguration: mc,
		}
	}
	return endpoint.outgoingMessages[mc.messageType]
}

//Declare a message configuration.
func (endpoint *Endpoint) Message(messageType string) *MessageConfiguration {
	return &MessageConfiguration{
		messageType: messageType,
		endpoint:    endpoint,
	}
}

/*
Start receiving message with the ServiceBus
*/
func (endpoint *Endpoint) Start() error {
	go endpoint.handleReceivedMessages()

	err := endpoint.Transport.Start(endpoint)
	if err != nil {
		return err
	}

	return nil
}

/*
Publish a message to all subscribers
*/
func (endpoint *Endpoint) Publish(messageType string, msg interface{}, options ...OutgoingMutation) error {
	ctx := endpoint.createMessageContext(messageType, msg, options)
	if ctx.IsCancelled {
		return nil
	}

	err := endpoint.Transport.Publish(ctx)
	if err != nil {
		return err
	}

	return nil
}

/*
Send a message to a specific Endpoint
*/
func (endpoint *Endpoint) Send(messageType string, destination string, msg interface{}, options ...OutgoingMutation) error {
	ctx := endpoint.createMessageContext(messageType, msg, options)
	if ctx.IsCancelled {
		return nil
	}

	err := endpoint.Transport.Send(destination, ctx)
	if err != nil {
		return err
	}

	return nil
}

/*
Send the message to the local Endpoint
*/
func (endpoint *Endpoint) SendLocal(messageType string, msg interface{}, options ...OutgoingMutation) error {
	ctx := endpoint.createMessageContext(messageType, msg, options)
	if ctx.IsCancelled {
		return nil
	}

	err := endpoint.Transport.SendLocal(ctx)
	if err != nil {
		return err
	}

	return nil
}

func validateMessage(ctx *IncomingMessageContext) error {
	if ctx.Origin == "" {
		return errors.New("Message has no Origin.")
	}
	if ctx.Type == "" {
		return errors.New("Message has no Type.")
	}
	if ctx.MessageId == "" {
		return errors.New("Message has no MessageId.")
	}
	if ctx.CorrelationId == "" {
		return errors.New("Message has no CorrelationId.")
	}
	return nil
}

func (endpoint *Endpoint) handleReceivedMessages() {
	received := endpoint.Transport.MessageReceived(make(chan *IncomingMessageContext))
	for {
		msg := <-received
		err := validateMessage(msg)
		if err != nil {
			return
		}

		if _, ok := endpoint.incomingMessages[msg.Type]; !ok {
			_ = endpoint.Transport.UnregisterRouting(msg.Type) //unregister routing from transport as there is no handler any more.
			return
		}

		endpoint.startSagas(msg)

		for _, handler := range endpoint.incomingMessages[msg.Type].handler {
			handler(msg)
		}
	}
}

func (endpoint *Endpoint) startSagas(ctx *IncomingMessageContext) {
	config := endpoint.incomingMessages[ctx.Type]
	for _, s := range config.sagas {
		exists, err := endpoint.SagaStore.SagaExists(ctx.CorrelationId, s)
		if err != nil {
			panic("Error on checking saga existence")
		}
		if exists {
			continue
		}

		err = endpoint.SagaStore.CreateSaga(ctx.CorrelationId, s)
		if err != nil {
			panic("Error on creating new saga")
		}

	}
}

func (endpoint *Endpoint) createMessageContext(messageType string, payload interface{}, mutations []OutgoingMutation) *OutgoingMessageContext {
	ctx := CreateOutgoingContext(endpoint)
	ctx.Payload = payload
	ctx.Type = messageType
	ctx.MessageId = uuid.New().String()
	ctx.Timestamp = time.Now().UTC()
	ctx.Origin = endpoint.Name
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

	return ctx
}

package azureservicebus

import (
	"github.com/abecu-hub/go-bus/pkg/servicebus"
)

type Transport struct {
	ConnectionString string
}

func Create(connectionString string) *Transport {
	return &Transport{}
}

func (Transport) Start(endpoint *servicebus.Endpoint) error {
	panic("implement me")
}

func (Transport) RegisterRouting(route string) error {
	panic("implement me")
}

func (Transport) UnregisterRouting(route string) error {
	panic("implement me")
}

func (Transport) Publish(message *servicebus.OutgoingMessageContext) error {
	panic("implement me")
}

func (Transport) Send(destination string, command *servicebus.OutgoingMessageContext) error {
	panic("implement me")
}

func (Transport) SendLocal(command *servicebus.OutgoingMessageContext) error {
	panic("implement me")
}

func (Transport) MessageReceived(contexts chan *servicebus.IncomingMessageContext) chan *servicebus.IncomingMessageContext {
	panic("implement me")
}

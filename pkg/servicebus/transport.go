package servicebus

type Transport interface {
	Start(endpointName string) error
	RegisterRouting(route string) error
	UnregisterRouting(route string) error
	Publish(message *OutgoingMessageContext) error
	Send(destination string, command *OutgoingMessageContext) error
	SendLocal(command *OutgoingMessageContext) error
	MessageReceived(chan *IncomingMessageContext) chan *IncomingMessageContext
}

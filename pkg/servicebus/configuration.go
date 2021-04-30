package servicebus

type OutgoingMutation func(ctx *OutgoingMessageContext)
type IncomingMutation func(ctx *IncomingMessageContext)

type MessageConfiguration struct {
	endpoint    *ServiceBusEndpoint
	messageType string
}

type OutgoingMessageConfiguration struct {
	messageConfiguration *MessageConfiguration
	mutations            []OutgoingMutation
}

//Mutates the outgoing message context with the given function. Multiple mutations will be executed in order of declaration.
func (config *OutgoingMessageConfiguration) Mutate(behavior OutgoingMutation) *OutgoingMessageConfiguration {
	config.mutations = append(config.mutations, behavior)
	return config
}

type IncomingMessageConfiguration struct {
	messageConfiguration *MessageConfiguration
	handler              []func(ctx *IncomingMessageContext)
	mutations            []IncomingMutation
	sagas                []string
}

//Handles the incoming message context with the given function
func (config *IncomingMessageConfiguration) Handle(handler func(ctx *IncomingMessageContext)) *IncomingMessageConfiguration {
	config.handler = append(config.handler, handler)
	config.messageConfiguration.endpoint.transport.RegisterRouting(config.messageConfiguration.messageType)
	return config
}

//Mutates the incoming message context with the given function. Multiple mutations will be executed in order of declaration.
func (config *IncomingMessageConfiguration) Mutate(behavior IncomingMutation) *IncomingMessageConfiguration {
	config.mutations = append(config.mutations, behavior)
	return config
}

//Start a saga of the given type whenever a message of this configuration has been received.
func (config *IncomingMessageConfiguration) StartSaga(saga string) *IncomingMessageConfiguration {
	if config.messageConfiguration.endpoint.sagaStore == nil {
		panic("ServiceBusEndpoint has no saga store configured.")
	}
	config.sagas = append(config.sagas, saga)
	return config
}

//Declare this message configuration to be an incoming message.
func (config *MessageConfiguration) AsIncoming() *IncomingMessageConfiguration {
	return config.endpoint.createOrGetIncomingMessageConfig(config)
}

//Declare this message configuration to be an outgoing message.
func (config *MessageConfiguration) AsOutgoing() *OutgoingMessageConfiguration {
	return config.endpoint.createOrGetOutgoingMessageConfig(config)
}

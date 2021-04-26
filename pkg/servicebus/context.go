package servicebus

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/abecu-hub/go-bus/pkg/servicebus/saga"
	"time"
)

type OutgoingMessageContext struct {
	Origin        string
	Type          string
	CorrelationId string
	MessageId     string
	Timestamp     time.Time
	Payload       interface{}
	Priority      uint8
	Headers       map[string]interface{}
	endpoint      *Endpoint
	Version       string
	IsCancelled   bool
}

func CreateOutgoingContext(endpoint *Endpoint) *OutgoingMessageContext {
	return &OutgoingMessageContext{
		endpoint: endpoint,
	}
}

func (context *OutgoingMessageContext) Cancel() {
	context.IsCancelled = true
}

/*
The IncomingMessageContext holds the message information of the Endpoint instance that handled the message.
*/
type IncomingMessageContext struct {
	Headers       map[string]interface{}
	Origin        string
	Payload       []byte
	Type          string
	CorrelationId string
	MessageId     string
	Timestamp     time.Time
	Priority      uint8
	endpoint      *Endpoint
	Ack           func()
	Retry         func()
	Discard       func()
	Fail          func()
	Test          string
}

func (context *IncomingMessageContext) setEndpoint(endpoint *Endpoint) {
	context.endpoint = endpoint
}

func (context *IncomingMessageContext) validate() error {

	if context.Origin == "" {
		return errors.New("Message has no Origin.")
	}
	if context.Type == "" {
		return errors.New("Message has no Type.")
	}
	if context.MessageId == "" {
		return errors.New("Message has no MessageId.")
	}
	if context.CorrelationId == "" {
		return errors.New("Message has no CorrelationId.")
	}
	return nil

}

/*
Bind the message payload to a struct object
*/
func (context *IncomingMessageContext) Bind(obj interface{}) error {
	err := json.Unmarshal(context.Payload, &obj)
	if err != nil {
		return err
	}
	return nil
}

/*
Reply with a message to the origin of the current message context.
*/
func (context *IncomingMessageContext) Reply(messageType string, msg interface{}, options ...OutgoingMutation) error {
	origin := fmt.Sprintf("%v", context.Headers["Origin"])
	options = append(options, func(m *OutgoingMessageContext) {
		m.CorrelationId = context.CorrelationId
	})
	err := context.endpoint.Send(messageType, origin, msg, options...)
	if err != nil {
		return err
	}
	return nil
}

/*
Send a message to a specific Endpoint.
*/
func (context *IncomingMessageContext) Send(messageType string, destination string, msg interface{}, options ...OutgoingMutation) error {
	options = append(options, func(m *OutgoingMessageContext) {
		m.CorrelationId = context.CorrelationId
	})
	err := context.endpoint.Send(messageType, destination, msg, options...)
	if err != nil {
		return err
	}
	return nil
}

/*
Publish a message to all subscribers.
*/
func (context *IncomingMessageContext) Publish(messageType string, msg interface{}, options ...OutgoingMutation) error {
	options = append(options, func(m *OutgoingMessageContext) {
		m.CorrelationId = context.CorrelationId
	})
	err := context.endpoint.Publish(messageType, msg, options...)
	if err != nil {
		return err
	}
	return nil
}

/*
Send the message to the local Endpoint.
*/
func (context *IncomingMessageContext) SendLocal(messageType string, msg interface{}, options ...OutgoingMutation) error {
	options = append(options, func(m *OutgoingMessageContext) {
		m.CorrelationId = context.CorrelationId
	})
	err := context.endpoint.SendLocal(messageType, msg, options...)
	if err != nil {
		return err
	}
	return nil
}

//Request a saga from the persistence store and applies a transaction lock
func (context *IncomingMessageContext) RequestSaga(sagaType string) (*saga.Context, error) {
	s, err := context.endpoint.SagaStore.RequestSaga(context.CorrelationId, sagaType)
	if err != nil {
		return nil, err
	}

	if s.State == nil {
		s.State = make(map[string]interface{})
	}

	return s, nil
}

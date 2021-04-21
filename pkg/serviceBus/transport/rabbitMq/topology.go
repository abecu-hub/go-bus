package rabbitMq

import (
	"github.com/streadway/amqp"
)

type Topology interface {
	Setup() error
	RegisterRouting(route string) error
	UnregisterRouting(route string) error
	Publish(msg *amqp.Publishing) error
	Send(destination string, msg *amqp.Publishing) error
	SendLocal(msg *amqp.Publishing) error
}

type DefaultTopology struct {
	Transport *Transport
	Exchange  string
}

func (t *DefaultTopology) Setup() error {
	err := t.Transport.currentChannel.ExchangeDeclare(
		t.Exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil)

	if err != nil {
		return err
	}

	_, err = t.Transport.currentChannel.QueueDeclare(
		t.Transport.InputQueue.Name,
		true,
		false,
		false,
		true,
		t.Transport.InputQueue.Args)

	if err != nil {
		return err
	}
	return nil
}

func (t *DefaultTopology) RegisterRouting(route string) error {
	err := t.Transport.currentChannel.QueueBind(t.Transport.InputQueue.Name, route, t.Exchange, false, t.Transport.InputQueue.Args)
	if err != nil {
		return err
	}
	return nil
}

func (t *DefaultTopology) UnregisterRouting(route string) error {
	err := t.Transport.currentChannel.QueueUnbind(t.Transport.InputQueue.Name, route, t.Exchange, nil)
	return err
}

func (t *DefaultTopology) Publish(msg *amqp.Publishing) error {
	err := t.Transport.currentChannel.Publish(t.Exchange, msg.Type, false, false, *msg)
	if err != nil {
		return err
	}
	return nil
}

func (t *DefaultTopology) Send(destination string, msg *amqp.Publishing) error {
	err := t.Transport.currentChannel.Publish("", destination, false, false, *msg)
	if err != nil {
		return err
	}
	return nil
}

func (t *DefaultTopology) SendLocal(msg *amqp.Publishing) error {
	err := t.Transport.currentChannel.Publish("", t.Transport.InputQueue.Name, false, false, *msg)
	if err != nil {
		return err
	}
	return nil
}

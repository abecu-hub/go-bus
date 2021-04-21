package rabbitMq

import (
	"fmt"
	"github.com/streadway/amqp"
)

type Topology interface {
	Setup()
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

func (t *DefaultTopology) Setup() {
	err := t.Transport.currentChannel.ExchangeDeclare(
		t.Exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil)

	if err != nil {
		fmt.Print(err)
	}

	_, err = t.Transport.currentChannel.QueueDeclare(
		t.Transport.InputQueue.Name,
		true,
		false,
		false,
		true,
		t.Transport.InputQueue.Args)

	if err != nil {
		fmt.Print(err)
	}
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

type FanoutTopology struct {
	Transport *Transport
}

func (t *FanoutTopology) Setup() {
	_, err := t.Transport.currentChannel.QueueDeclare(
		t.Transport.InputQueue.Name,
		true,
		false,
		false,
		true,
		nil)

	if err != nil {
		fmt.Print(err)
	}
}

func (t *FanoutTopology) RegisterRouting(route string) error {
	err := t.Transport.currentChannel.ExchangeDeclare(route, "fanout", true, false, false, false, nil)
	if err != nil {
		return err
	}
	err = t.Transport.currentChannel.QueueBind(t.Transport.InputQueue.Name, "", route, false, t.Transport.InputQueue.Args)
	if err != nil {
		return err
	}
	return nil
}

func (t *FanoutTopology) UnregisterRouting(route string) error {
	panic("Not implemented!")
}

func (t *FanoutTopology) Publish(msg *amqp.Publishing) error {
	err := t.Transport.currentChannel.Publish(msg.Type, "", false, false, *msg)
	if err != nil {
		return err
	}
	return nil
}

func (t *FanoutTopology) Send(destination string, msg *amqp.Publishing) error {
	panic("Not implemented")
}

func (t *FanoutTopology) SendLocal(msg *amqp.Publishing) error {
	panic("Not implemented")
}

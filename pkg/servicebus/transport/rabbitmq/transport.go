package rabbitmq

import (
	"encoding/json"
	"fmt"
	"github.com/abecu-hub/go-bus/pkg/servicebus"
	"github.com/streadway/amqp"
	"time"
)

type Transport struct {
	Url               string
	InputQueue        Queue
	topology          Topology
	routingBuffer     []string
	started           bool
	connection        *amqp.Connection
	currentChannel    *amqp.Channel
	errorNotification chan *amqp.Error
	messageReceived   chan *servicebus.IncomingMessageContext
}

type Queue struct {
	Name    string
	Durable bool
	AutoAck bool
	Args    amqp.Table
}

type ConsumeSettings struct {
	Consumer  string
	AutoAck   bool
	Exclusive bool
	NoLocal   bool
	NoWait    bool
}

func (rmq *Transport) connect() error {
	conn, err := amqp.Dial(rmq.Url)
	if err != nil {
		return err
	}
	rmq.connection = conn

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	rmq.currentChannel = ch

	err = rmq.topology.Setup()
	if err != nil {
		return err
	}

	rmq.errorNotification = conn.NotifyClose(make(chan *amqp.Error))

	return nil
}

/*
Create a new RabbitMQ Transport for the go-bus endpoint.
*/
func Create(url string, options ...func(*Transport)) *Transport {
	rmq := &Transport{
		Url:           url,
		routingBuffer: make([]string, 0),
	}

	UseDefaultTopology("amq.topic")(rmq)

	for _, option := range options {
		option(rmq)
	}

	return rmq
}

func (rmq *Transport) isConnected() bool {
	return rmq.started && !rmq.connection.IsClosed()
}

func (rmq *Transport) reconnect() {
	_ = rmq.currentChannel.Cancel("", false)
	_ = rmq.connection.Close()

	err := rmq.Start(rmq.InputQueue.Name)
	for err != nil {
		err = rmq.Start(rmq.InputQueue.Name)
		if err != nil {
			time.Sleep(5 * time.Second)
		}
	}
}

func (rmq *Transport) Start(endpointName string) error {

	rmq.InputQueue.Name = endpointName

	err := rmq.connect()
	if err != nil {
		return err
	}

	for _, route := range rmq.routingBuffer {
		err = rmq.topology.RegisterRouting(route)
		if err != nil {
			return err
		}
	}

	msgs, err := rmq.currentChannel.Consume(
		rmq.InputQueue.Name,
		"", //container name maybe?
		rmq.InputQueue.AutoAck,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return fmt.Errorf("Error consuming messages from RabbitMQ transport")
	}

	go func() {
	Loop:
		for {
			select {
			case err = <-rmq.errorNotification:
				rmq.reconnect()
				break Loop
			case d := <-msgs:
				rmq.messageReceived <- rmq.createIncomingContext(&d)
			}
		}
	}()

	return nil
}

func (rmq *Transport) MessageReceived(eventChannel chan *servicebus.IncomingMessageContext) chan *servicebus.IncomingMessageContext {
	rmq.messageReceived = eventChannel
	return eventChannel
}

func (rmq *Transport) RegisterRouting(route string) error {
	if !rmq.isConnected() {
		rmq.routingBuffer = append(rmq.routingBuffer, route)
		return nil
	}

	err := rmq.topology.RegisterRouting(route)
	if err != nil {
		return fmt.Errorf("Error on registering route " + route)
	}
	return nil
}

func (rmq *Transport) UnregisterRouting(route string) error {
	err := rmq.topology.UnregisterRouting(route)
	if err != nil {
		return fmt.Errorf("Error on unregistering route " + route)
	}
	return nil
}

func (rmq *Transport) Publish(ctx *servicebus.OutgoingMessageContext) error {
	msg, err := rmq.createTransportMessage(ctx)
	if err != nil {
		return err
	}

	err = rmq.topology.Publish(msg)
	if err != nil {
		return err
	}

	return nil
}

func (rmq *Transport) Send(destination string, ctx *servicebus.OutgoingMessageContext) error {
	msg, err := rmq.createTransportMessage(ctx)
	if err != nil {
		return err
	}

	err = rmq.topology.Send(destination, msg)
	if err != nil {
		return err
	}

	return nil
}

func (rmq *Transport) SendLocal(ctx *servicebus.OutgoingMessageContext) error {
	msg, err := rmq.createTransportMessage(ctx)
	if err != nil {
		return err
	}

	err = rmq.topology.SendLocal(msg)
	if err != nil {
		return err
	}

	return nil
}

func (rmq *Transport) createTransportMessage(ctx *servicebus.OutgoingMessageContext) (*amqp.Publishing, error) {
	payload, err := json.Marshal(ctx.Payload)
	if err != nil {
		return nil, err
	}

	ctx.Headers["Origin"] = ctx.Origin
	ctx.Headers["CorrelationTimestamp"] = ctx.CorrelationTimestamp
	return &amqp.Publishing{
		ContentType:   "text/json",
		Body:          payload,
		Headers:       ctx.Headers,
		Priority:      ctx.Priority,
		MessageId:     ctx.MessageId,
		Timestamp:     ctx.Timestamp,
		Type:          ctx.Type,
		CorrelationId: ctx.CorrelationId,
	}, nil
}

func (rmq *Transport) createIncomingContext(d *amqp.Delivery) *servicebus.IncomingMessageContext {
	ctx := new(servicebus.IncomingMessageContext)
	ctx.Headers = d.Headers
	ctx.Payload = d.Body
	ctx.Type = d.Type
	ctx.CorrelationId = d.CorrelationId
	ctx.MessageId = d.MessageId
	ctx.Timestamp = d.Timestamp
	ctx.Priority = d.Priority
	ctx.Origin = fmt.Sprint(d.Headers["Origin"])
	corrTime, castOk := d.Headers["CorrelationTimestamp"].(time.Time)
	if castOk {
		ctx.CorrelationTimestamp = corrTime
	}
	ctx.Ack = func() {
		_ = d.Ack(false)
	}
	ctx.Retry = func() {
		_ = d.Reject(true)
	}
	ctx.Discard = func() {
		_ = d.Reject(false)
	}
	return ctx
}

func (rmq *Transport) GetConnection() *amqp.Connection {
	return rmq.connection
}

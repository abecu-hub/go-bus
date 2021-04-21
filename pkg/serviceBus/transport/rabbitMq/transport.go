package rabbitMq

import (
	"dev.azure.com/finorun/Playground/_git/go-bus.git/pkg/serviceBus"
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"time"
)

type Transport struct {
	Url               string
	InputQueue        Queue
	topology          Topology
	endpoint          *serviceBus.Endpoint
	routingBuffer     []string
	started           bool
	connection        *amqp.Connection
	currentChannel    *amqp.Channel
	errorNotification chan *amqp.Error
	messageReceived   chan *serviceBus.IncomingMessageContext
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
		fmt.Print(err)
		return err
	}
	rmq.connection = conn

	ch, err := conn.Channel()
	if err != nil {
		fmt.Print(err)
		return err
	}
	rmq.currentChannel = ch

	rmq.topology.Setup()

	if err != nil {
		fmt.Print(err)
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

func (rmq *Transport) reconnect(endpoint *serviceBus.Endpoint) {
	_ = rmq.currentChannel.Cancel("", false)
	_ = rmq.connection.Close()

	err := rmq.Start(endpoint)
	for err != nil {
		err = rmq.Start(endpoint)
		if err != nil {
			time.Sleep(5 * time.Second)
		}
	}
}

func (rmq *Transport) Start(endpoint *serviceBus.Endpoint) error {
	rmq.endpoint = endpoint
	rmq.InputQueue.Name = endpoint.Name

	err := rmq.connect()
	if err != nil {
		return err
	}

	for _, route := range rmq.routingBuffer {
		rmq.topology.RegisterRouting(route)
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
				log.Println(err)
				rmq.reconnect(endpoint)
				break Loop
			case d := <-msgs:
				rmq.messageReceived <- rmq.createIncomingContext(&d)
			}
		}
	}()

	return nil
}

func (rmq *Transport) MessageReceived(eventChannel chan *serviceBus.IncomingMessageContext) chan *serviceBus.IncomingMessageContext {
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

func (rmq *Transport) Publish(ctx *serviceBus.OutgoingMessageContext) error {
	msg := rmq.createTransportMessage(ctx)

	err := rmq.topology.Publish(msg)
	if err != nil {
		return err
	}

	return nil
}

func (rmq *Transport) Send(destination string, ctx *serviceBus.OutgoingMessageContext) error {
	msg := rmq.createTransportMessage(ctx)

	err := rmq.topology.Send(destination, msg)
	if err != nil {
		return err
	}

	return nil
}

func (rmq *Transport) SendLocal(ctx *serviceBus.OutgoingMessageContext) error {
	msg := rmq.createTransportMessage(ctx)

	err := rmq.topology.SendLocal(msg)
	if err != nil {
		return err
	}

	return nil
}

func (rmq *Transport) createTransportMessage(ctx *serviceBus.OutgoingMessageContext) *amqp.Publishing {
	payload, err := json.Marshal(ctx.Payload)
	if err != nil {
		fmt.Println(err)
	}

	ctx.Headers["Origin"] = ctx.Origin
	return &amqp.Publishing{
		ContentType:   "text/json",
		Body:          payload,
		Headers:       ctx.Headers,
		Priority:      ctx.Priority,
		MessageId:     ctx.MessageId,
		Timestamp:     ctx.Timestamp,
		Type:          ctx.Type,
		CorrelationId: ctx.CorrelationId,
	}
}

func (rmq *Transport) createIncomingContext(d *amqp.Delivery) *serviceBus.IncomingMessageContext {
	ctx := serviceBus.CreateIncomingContext(rmq.endpoint)
	ctx.Headers = d.Headers
	ctx.Payload = d.Body
	ctx.Type = d.Type
	ctx.CorrelationId = d.CorrelationId
	ctx.MessageId = d.MessageId
	ctx.Timestamp = d.Timestamp
	ctx.Priority = d.Priority
	ctx.Origin = fmt.Sprint(d.Headers["Origin"])
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

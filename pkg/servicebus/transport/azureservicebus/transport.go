package azureservicebus

import (
	"context"
	"encoding/json"
	asb "github.com/Azure/azure-service-bus-go"
	"github.com/abecu-hub/go-bus/pkg/servicebus"
	"time"
)

type Transport struct {
	ConnectionString string
	Namespace        *asb.Namespace
	QueueManager     *asb.QueueManager
	Queue            *asb.Queue
	routingBuffer    []string
	messageReceived  chan *servicebus.IncomingMessageContext
}

func Create(connectionString string) *Transport {
	return &Transport{
		ConnectionString: connectionString,
	}
}

func (t *Transport) Start(endpointName string) error {
	ns, err := asb.NewNamespace(asb.NamespaceWithConnectionString(t.ConnectionString))
	if err != nil {
		return err
	}
	t.Namespace = ns
	t.QueueManager = ns.NewQueueManager()
	qe, err := t.ensureQueue(endpointName)
	if err != nil {
		return err
	}
	t.Queue, err = ns.NewQueue(qe.Name)
	if err != nil {
		return err
	}

	for _, route := range t.routingBuffer {
		err = t.subscribe(route)
		if err != nil {
			return err
		}
	}

	//Todo: Start listening to messages here
	go t.consume()

	return nil
}

func (t *Transport) consume() {
	var handler asb.HandlerFunc = func(ctx context.Context, msg *asb.Message) error {

		return msg.Complete(ctx)
	}
	for {
		t.Queue.Receive(context.Background(), handler)
	}
}

func (t *Transport) RegisterRouting(route string) error {
	if t.Namespace == nil {
		t.routingBuffer = append(t.routingBuffer, route)
		return nil
	}

	err := t.subscribe(route)
	if err != nil {
		return err
	}
	return nil
}

func (t *Transport) subscribe(route string) error {
	_, err := t.ensureTopic(route)
	if err != nil {
		return err
	}

	_, err = t.ensureSubscription(route)
	if err != nil {
		return err
	}

	return nil
}

func (Transport) UnregisterRouting(route string) error {
	panic("implement me")
}

func (t *Transport) Publish(message *servicebus.OutgoingMessageContext) error {
	panic("implement me")
}

func (Transport) Send(destination string, command *servicebus.OutgoingMessageContext) error {
	panic("implement me")
}

func (t *Transport) SendLocal(command *servicebus.OutgoingMessageContext) error {
	msg, err := t.createTransportMessage(command)
	if err != nil {
		return err
	}
	err = t.Queue.Send(context.Background(), msg)
	if err != nil {
		return err
	}
	return nil
}

func (t *Transport) MessageReceived(eventChannel chan *servicebus.IncomingMessageContext) chan *servicebus.IncomingMessageContext {
	t.messageReceived = eventChannel
	return eventChannel
}

func (t *Transport) ensureQueue(name string, opts ...asb.QueueManagementOption) (*asb.QueueEntity, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	qm := t.Namespace.NewQueueManager()
	qe, err := qm.Get(ctx, name)
	if err == nil {
		_ = qm.Delete(ctx, name)
	}

	qe, err = qm.Put(ctx, name, opts...)
	if err != nil {
		return nil, err
	}
	return qe, nil
}

func (t *Transport) ensureSubscription(topic string) (*asb.SubscriptionEntity, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	sm, err := t.Namespace.NewSubscriptionManager(topic)
	if err != nil {
		return nil, err
	}
	subscriptionName := topic + "_" + t.Queue.Name
	se, err := sm.Get(ctx, subscriptionName)
	if err != nil {
		_ = sm.Delete(ctx, subscriptionName)
	}

	se, err = sm.Put(ctx, subscriptionName, func(description *asb.SubscriptionDescription) error {
		description.ForwardTo = &t.Queue.Name
		return nil
	})
	if err != nil {
		return nil, err
	}

	return se, nil
}

func (t *Transport) ensureTopic(topic string) (*asb.TopicEntity, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	tm := t.Namespace.NewTopicManager()
	te, err := tm.Get(ctx, topic)
	if err != nil {
		err = tm.Delete(ctx, topic)
	}
	te, err = tm.Put(ctx, topic)
	if err != nil {
		return nil, err
	}
	return te, nil
}

func (t *Transport) createTransportMessage(ctx *servicebus.OutgoingMessageContext) (*asb.Message, error) {
	payload, err := json.Marshal(ctx)
	if err != nil {
		return nil, err
	}

	return asb.NewMessage(payload), nil
}

func (t *Transport) createIncomingContext(msg *asb.Message) (*servicebus.IncomingMessageContext, error) {
	ctx := new(servicebus.IncomingMessageContext)
	ctx.Ack = func() {
		_ = msg.Complete(context.Background())
	}
	return nil, nil
}

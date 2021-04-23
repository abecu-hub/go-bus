package main

import (
	"context"
	"github.com/abecu-hub/go-bus/pkg/servicebus"
	"github.com/abecu-hub/go-bus/pkg/servicebus/saga"
	"github.com/abecu-hub/go-bus/pkg/servicebus/transport/rabbitmq"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
	"time"
)

const (
	OrderSaga = "OrderSaga"
)

const (
	StartOrderMessage     = "StartOrder"
	OrderStartedMessage   = "OrderStarted"
	OrderPaidMessage      = "OrderPaid"
	OrderPreparedMessage  = "OrderPrepared"
	OrderDeliveredMessage = "OrderDelivered"
	OrderCompletedMessage = "OrderCompleted"
)

type StartOrder struct {
	OrderID string
}

type OrderStarted struct {
	OrderID string
}

type OrderPaid struct {
	OrderID       string
	PaymentMethod string
	Amount        int
}

type OrderPrepared struct {
	OrderID string
}

type OrderDelivered struct {
	OrderID string
}

func main() {
	mongoSagaStore, err := saga.CreateMongoStore(createMongoClient(),
		"OrderServiceDB",
		"sagas")
	if err != nil {
		panic("Error initializing MongoDB saga store!")
	}
	orderServiceEndpoint := servicebus.Create("OrderService",
		rabbitmq.Create("amqp://guest:guest@localhost:5672/"),
		servicebus.UseSagas(mongoSagaStore))

	orderServiceEndpoint.Message(StartOrderMessage).
		AsIncoming().
		Handle(orderService_StartOrder).
		StartSaga(OrderSaga)

	orderServiceEndpoint.Message(OrderPaidMessage).
		AsIncoming().
		Handle(orderService_OrderPaid)

	orderServiceEndpoint.Message(OrderPreparedMessage).
		AsIncoming().
		Handle(orderService_OrderPrepared)

	orderServiceEndpoint.Message(OrderDeliveredMessage).
		AsIncoming().
		Handle(orderService_OrderDelivered)

	err = orderServiceEndpoint.Start()
	if err != nil {
		panic("Error starting OrderService endpoint!")
	}

	paymentServiceEndpoint := servicebus.Create("PaymentService",
		rabbitmq.Create("amqp://guest:guest@localhost:5672/"))

	paymentServiceEndpoint.Message(OrderStartedMessage).
		AsIncoming().
		Handle(paymentService_OrderStarted)

	err = paymentServiceEndpoint.Start()
	if err != nil {
		panic("Error starting PaymentService endpoint!")
	}

	stockServiceEndpoint := servicebus.Create("StockService",
		rabbitmq.Create("amqp://guest:guest@localhost:5672/"))

	stockServiceEndpoint.Message(OrderPaidMessage).
		AsIncoming().
		Handle(stockService_OrderPaid)

	err = stockServiceEndpoint.Start()
	if err != nil {
		panic("Error starting StockService endpoint!")
	}

	deliveryServiceEndpoint := servicebus.Create("DeliveryService",
		rabbitmq.Create("amqp://guest:guest@localhost:5672/"))

	deliveryServiceEndpoint.Message(OrderPreparedMessage).
		AsIncoming().
		Handle(deliveryService_OrderPrepared)

	err = deliveryServiceEndpoint.Start()
	if err != nil {
		panic("Error starting DeliveryService endpoint!")
	}

	err = orderServiceEndpoint.SendLocal(StartOrderMessage, &StartOrder{OrderID: uuid.New().String()})
	if err != nil {
		panic("Error sending initial message!")
	}

	<-make(chan bool)
}
func orderService_StartOrder(ctx *servicebus.IncomingMessageContext) {
	s, err := ctx.RequestSaga(OrderSaga)
	if err != nil {
		ctx.Retry()
	}

	startOrder := new(StartOrder)
	err = ctx.Bind(startOrder)
	if err != nil {
		ctx.Fail()
	}

	s.State["OrderID"] = startOrder.OrderID

	s.Save()
	s.Done()

	err = ctx.Publish(OrderStartedMessage, &OrderStarted{
		OrderID: startOrder.OrderID,
	})
	if err != nil {
		ctx.Retry()
	}

	ctx.Ack()
}

func orderService_OrderPaid(ctx *servicebus.IncomingMessageContext) {
	s, err := ctx.RequestSaga(OrderSaga)
	if err != nil {
		ctx.Retry()
	}

	orderPaid := new(OrderPaid)
	err = ctx.Bind(orderPaid)
	if err != nil {
		ctx.Fail()
	}

	s.State["Payment"] = orderPaid

	s.Save()
	s.Done()

	ctx.Ack()
}

func orderService_OrderPrepared(ctx *servicebus.IncomingMessageContext) {
	s, err := ctx.RequestSaga(OrderSaga)
	if err != nil {
		ctx.Retry()
	}

	s.State["IsPrepared"] = true

	s.Save()
	s.Done()

	ctx.Ack()
}

type OrderCompleted struct {
	OrderID string
}

func orderService_OrderDelivered(ctx *servicebus.IncomingMessageContext) {
	s, err := ctx.RequestSaga(OrderSaga)
	if err != nil {
		ctx.Retry()
	}

	orderDelivered := new(OrderDelivered)
	err = ctx.Bind(orderDelivered)
	if err != nil {
		ctx.Fail()
	}

	s.State["IsDelivered"] = true

	s.Complete()

	s.Save()
	s.Done()

	err = ctx.Publish(OrderCompletedMessage, &OrderCompleted{
		OrderID: orderDelivered.OrderID,
	})
	if err != nil {
		ctx.Retry()
	}

	ctx.Ack()
}

func paymentService_OrderStarted(ctx *servicebus.IncomingMessageContext) {
	orderStarted := new(OrderStarted)
	err := ctx.Bind(orderStarted)
	if err != nil {
		ctx.Fail()
	}

	//Do actual payment stuff here...

	err = ctx.Publish(OrderPaidMessage, &OrderPaid{
		OrderID:       orderStarted.OrderID,
		PaymentMethod: "PayPal",
		Amount:        123311,
	})
	if err != nil {
		ctx.Retry()
	}

	ctx.Ack()
}

func stockService_OrderPaid(ctx *servicebus.IncomingMessageContext) {
	orderPaid := new(OrderPaid)
	err := ctx.Bind(orderPaid)
	if err != nil {
		ctx.Fail()
	}

	//Do actual order preparation stuff here...

	err = ctx.Publish(OrderPreparedMessage, &OrderPrepared{
		OrderID: orderPaid.OrderID,
	})
	if err != nil {
		ctx.Retry()
	}

	ctx.Ack()
}

func deliveryService_OrderPrepared(ctx *servicebus.IncomingMessageContext) {
	orderPrepared := new(OrderPrepared)
	err := ctx.Bind(orderPrepared)
	if err != nil {
		ctx.Fail()
	}

	//Do actual order delivery stuff here...

	err = ctx.Publish(OrderDeliveredMessage, &OrderDelivered{
		OrderID: orderPrepared.OrderID,
	})
	if err != nil {
		ctx.Retry()
	}

	ctx.Ack()
}

func createMongoClient() *mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tM := reflect.TypeOf(bson.M{})
	reg := bson.NewRegistryBuilder().RegisterTypeMapEntry(bsontype.EmbeddedDocument, tM).Build()

	o := options.Client().
		SetRegistry(reg).
		SetReplicaSet("my-mongo-set").
		ApplyURI("mongodb://localhost:27017")

	client, err := mongo.Connect(ctx, o)
	if err != nil {
		panic("MongoDB not connected!")
	}

	return client
}

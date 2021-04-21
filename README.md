# go-bus

go-bus is a transport agnostic service bus implementation written in go. go-bus aims to provide high-level messaging capabilities to your services with close to no configuration at all and enables your service architecture to asynchronously communicate in an event-driven way. 

## Getting started

Create a service bus endpoint with RabbitMQ transport

```go
endpoint := serviceBus.Create("awesomeService",
    rabbitMq.Create("amqp://guest:guest@localhost:5672/"))
```

You can then configure incoming messages you want to handle like this:

```go
endpoint.Message("CreateUser").
    AsIncoming().
    Handle(createUserHandler)

func createUserHandler(ctx *serviceBus.IncomingMessageContext) {
    fmt.Println("received CreateUser!")
    ctx.Ack()
}
```

By default messages need to be handled with acknowledgement.

- ctx.Ack() -> Tell the server that the msg has been acknowledged. Message will be removed from queue.
- ctx.Discard() -> Tell the server that the msg will not be acknowledged, but also do not requeue. Can be used in cases where you decide the msg is of no value to you.
- ctx.Retry() -> Tell the server that the msg will not be acknowledged, but the msg will be requeued. This should be used in cases of transient errors like an unreachable service or database.
- ctx.Fail() -> Tell the server that the msg will not be acknowledged and move the msg to the error queue. This should be used in cases of unrecoverable errors like for example schema or serialization issues.

In case of a request/response pattern, you may send a reply to the originator of the received msg with

```go
ctx.Reply("YourReplyMessage", yourMessageObj)
```

You may bind the message payload to a matching struct
```go
type User struct {
    UserName string
    Email    string
}

func createUserHandler(ctx *serviceBus.IncomingMessageContext) {
    user := new(CreateUser)
    ctx.Bind(user)
    fmt.Println(user.UserName + " created!")
    ctx.Ack()
}
```

(De)-Serialization is currently done by using the standard go json lib. 

Start the service bus endpoint to send and receive messages.

```go
endpoint.Start()
``` 

You can then publish events to 0 or many subscribers...

```go
endpoint.Publish("UserCreated", &UserCreated{
    UserName: "Name",
    Email: "Name@email.com",
})
```

...send commands to a single endpoint...
```go
endpoint.Send("CreateUser", "SomeOtherAwesomeService", &CreateUser{
    UserName: "Name",
    Email: "Name@email.com",
})
```

...or send messages to the local endpoint
```go
endpoint.SendLocal("CreateUser", &CreateUser{
    UserName: "Name",
    Email: "Name@email.com",
})
```
A simple local development example could finally look like this

```go
type CreateUser struct {
    UserName string
    Email    string
}

func main() {
    endpoint := serviceBus.Create("awesomeService",
        rabbitMq.Create("amqp://guest:guest@localhost:5672/"))

    endpoint.Message("CreateUser").
        AsIncoming().
        Handle(createUserHandler)

    endpoint.Start()

    err := endpoint.SendLocal("CreateUser", &CreateUser{
    	UserName: "Name",
    	Email: "Name@email.com",
    })
    if err != nil {
        fmt.Println(err)
    }
}

func createUserHandler(ctx *serviceBus.IncomingMessageContext) {
    user := new(CreateUser)
    err := ctx.Bind(user)
    if err != nil {
        fmt.Println(err)
    }

    fmt.Println(user.UserName + " created!")
    
    ctx.Ack()
}
```

## Mutators

When configuring messages you can use mutators that will be executed on each send/publish/sendlocal or in case of incoming messages before the handler will be triggered. This enables you to alter the incoming or outgoing message context as you see fit.

```go
mutation := func(ctx *serviceBus.OutgoingMessageContext) {
    ctx.Headers["CustomHeader"] = "value"
}

endpoint.Message("UserCreated").
    AsOutgoing().
    Mutate(mutation)
```

You can also use mutators whenever you want to deliver a message e.g.
```go
endpoint.Publish("UserCreated", &UserCreated{
    UserName: "Name",
    Email: "Name@email.com",
}, mutation)
```

A mutation that is declared during delivery can override the mutations declared during the message configuration. 

There are currently 3 pre-defined mutators for outgoing messages:

- mutate.Priority(priority uint8): sets the priority property to the message context. Can be used in combination with RabbitMQ priority queues.
- mutate.Version(version string): sets a header "Version" to declare a message version.
- mutate.Header(key string, value string): sets any custom header with a key/value pair.

## Sagas

A saga is a sequence of local transactions that is used for business transactions that span multiple services. Sagas are natively integrated in go-bus and can be enabled by declaring a saga store for the endpoint like so:

```go
sagaStore = saga.CreateMongoStore(createMongoClient(), "busdb", "sagas")

endpoint := serviceBus.Create("awesomeService", rmq, serviceBus.UseSagas(sagaStore))
```
*go-bus* currently only comes with a MongoDB implementation as a saga store but can easily be extended by using the the _saga.Store_ interface.

A saga can be started by configuring a *StartSaga* on an incoming message configuration and receiving a message of the given type.

```go
endpoint.Message("CreateUser").
    AsIncoming().
    Handle(createUserHandler).
    StartSaga("MySaga")
```

This tells the service bus endpoint to create a saga of the type *MySaga* whenever a *CreateUser* message is received and the combination of CorrelationID and saga type does not already exist.

You can access the saga by requesting it from the saga store in a message handler context:

```go
func createUserHandler(ctx *serviceBus.IncomingMessageContext) {
    saga, err := ctx.RequestSaga("MySaga")
    if err != nil {
        ctx.Retry()
    }
    ...

}
```

Requesting a saga will put a transaction lock on the entity in the database to prevent parallel access to the saga and state inconsistencies. If another handler already requested the saga and hasn't released it yet, the function will return an error. After requesting a saga you can read or write from/to its state.

```go
func createUserHandler(ctx *serviceBus.IncomingMessageContext) {
    saga, err := ctx.RequestSaga("MySaga")
    if err != nil {
        ctx.Retry()
    }
    
    saga.State["MyData"] = []string {"Some", "New", "Data"}
    
    saga.Save()
    saga.Done()
    
    ctx.Ack()
}
```
Calling .Save() will save the state change to the saga request session, but ultimately does not store the data in the saga store. You have to call .Done() in order to commit all changes done to the saga state and to release the transaction lock from the saga.

A saga needs to be completed when it has gathered all the data needed. You can complete a saga simply by using .Complete()

```go
func createUserHandler(ctx *serviceBus.IncomingMessageContext) {
    saga, err := ctx.RequestSaga("MySaga")
    if err != nil {
        ctx.Retry()
    }
    
    saga.State["MyData"] = []string {"Some", "New", "Data"}
    
    saga.Save()	

    if saga.State["MyData"] != nil && saga.State["MyOtherData"] != nil {
        saga.Complete()
        ctx.Publish("MyDataCompleted", &MyDataCompleted{})
    }

    saga.Done()
    ctx.Ack()
}
```

### MongoDB Saga Store
In order to use the MongoDB saga store you have to make sure the MongoDB instance you are connecting against is running in a replica set. For local development you can easily setup a docker container to run MongoDB in a single node replica set.

Start a container instance like this:

```shell script
docker run -p 27017:27017 -d --name mongodb-replset mongo mongod --replSet my-mongo-set
```

Connect to the mongo shell

```shell script
docker exec -it mongodb-replset mongo
```

Create a config object in the mongo shell

```javascript
config = {
    "_id" : "my-mongo-set",
    "members" : [
        {
            "_id" : 0,
            "host" : "localhost:27017"
        }
    ]
}
```

Initiate the replica set with the config object

```javascript
rs.initiate(config)
```

And that's it! Your mongodb instance is now running in a single node replica set.

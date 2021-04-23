# go-bus

![image](https://user-images.githubusercontent.com/80049306/115728969-447db900-a385-11eb-972d-40c493946723.png)

go-bus is a transport agnostic service bus implementation written in go. go-bus aims to provide high-level messaging capabilities to your services with close to no configuration at all and enables your service architecture to asynchronously communicate in an event-driven way. 

Build & Test: ![build&test](https://github.com/abecu-hub/go-bus/actions/workflows/go.yml/badge.svg)

**Please note: This library is not production ready yet.**

Please read the [wiki](https://github.com/abecu-hub/go-bus/wiki) for more information.

## Getting started

Create a service bus endpoint with RabbitMQ transport

```go
endpoint := servicebus.Create("awesomeService",
    rabbitmq.Create("amqp://guest:guest@localhost:5672/"))
```

You can then configure incoming messages you want to handle like this:

```go
endpoint.Message("CreateUser").
    AsIncoming().
    Handle(createUserHandler)

func createUserHandler(ctx *servicebus.IncomingMessageContext) {
    fmt.Println("received CreateUser!")
    ctx.Ack()
}
```

By default messages need to be handled with acknowledgement.

```go
ctx.Ack()       // Tell the server that the msg has been acknowledged. Message will be removed from queue.
ctx.Discard()   // Tell the server that the msg will not be acknowledged, but also do not requeue. Can be used in cases where you decide the msg is of no value to you.
ctx.Retry()     // Tell the server that the msg will not be acknowledged, but the msg will be requeued. This should be used in cases of transient errors like an unreachable service or database.
ctx.Fail()      // Tell the server that the msg will not be acknowledged and move the msg to the error queue. This should be used in cases of unrecoverable errors like for example schema or serialization issues.
```

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

func createUserHandler(ctx *servicebus.IncomingMessageContext) {
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
    endpoint := servicebus.Create("awesomeService",
        rabbitmq.Create("amqp://guest:guest@localhost:5672/"))

    endpoint.Message("CreateUser").
        AsIncoming().
        Handle(createUserHandler)

    err := endpoint.Start()
    if err != nil {
        panic(err)
    }

    err = endpoint.SendLocal("CreateUser", &CreateUser{
    	UserName: "Name",
    	Email: "Name@email.com",
    })
    if err != nil {
        fmt.Println(err)
    }
}

func createUserHandler(ctx *servicebus.IncomingMessageContext) {
    user := new(CreateUser)
    err := ctx.Bind(user)
    if err != nil {
        fmt.Println(err)
    }

    fmt.Println(user.UserName + " created!")
    
    ctx.Ack()
}
```

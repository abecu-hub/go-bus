package main

import (
	"fmt"
	"github.com/abecu-hub/go-bus/pkg/servicebus"
	"github.com/abecu-hub/go-bus/pkg/servicebus/retrypolicy"
	"github.com/abecu-hub/go-bus/pkg/servicebus/transport/rabbitmq"
	"github.com/google/uuid"
	"time"
)

const (
	CreateUserMessage  = "CreateUser"
	UserCreatedMessage = "UserCreated"
)

type CreateUser struct {
	Name  string
	Email string
}

type User struct {
	ID    string
	Name  string
	Email string
}

func main() {
	endpoint := servicebus.Create("awesomeService",
		rabbitmq.Create("amqp://guest:guest@localhost:5672/"))

	endpoint.Message(CreateUserMessage).
		AsIncoming().
		Handle(createUser)

	endpoint.Message(CreateUserMessage).
		AsOutgoing().
		Retry(10, retrypolicy.Backoff(2*time.Second))

	endpoint.Message(UserCreatedMessage).
		AsIncoming().
		Handle(userCreated)

	err := endpoint.Start()
	if err != nil {
		panic(err)
	}

	for {
		err = endpoint.SendLocal(CreateUserMessage, &CreateUser{
			Name:  "UserName",
			Email: "username@useremail.com",
		})
		if err != nil {
			fmt.Println(err)
		}

		time.Sleep(2 * time.Second)
	}
}

func createUser(ctx *servicebus.IncomingMessageContext) {
	user := new(CreateUser)
	err := ctx.Bind(user)
	if err != nil {
		fmt.Println(err)
	}

	err = ctx.Publish(UserCreatedMessage, &User{
		ID:    uuid.New().String(),
		Name:  user.Name,
		Email: user.Email,
	})
	if err != nil {
		fmt.Println(err)
	}

	ctx.Ack()
}

func userCreated(ctx *servicebus.IncomingMessageContext) {
	user := new(User)
	err := ctx.Bind(user)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("User created!", "ID:", user.ID, "| name:", user.Name, "| Email:", user.Email)

	ctx.Ack()
}

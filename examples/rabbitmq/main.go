package main

import (
	"fmt"
	"github.com/abecu-hub/go-bus/pkg/serviceBus"
	"github.com/abecu-hub/go-bus/pkg/serviceBus/transport/rabbitMq"
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
	endpoint := serviceBus.Create("awesomeService",
		rabbitMq.Create("amqp://guest:guest@localhost:5672/"))

	endpoint.Message(CreateUserMessage).
		AsIncoming().
		Handle(createUser)

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

func createUser(ctx *serviceBus.IncomingMessageContext) {
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

func userCreated(ctx *serviceBus.IncomingMessageContext) {
	user := new(User)
	err := ctx.Bind(user)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("User created!", "ID:", user.ID, "| Name:", user.Name, "| Email:", user.Email)

	ctx.Ack()
}

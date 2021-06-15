package main

import (
	"fmt"
	"os"

	"github.com/abecu-hub/go-bus/pkg/servicebus"
	"github.com/abecu-hub/go-bus/pkg/servicebus/transport/azureservicebus"
)

type MyMessage struct {
	Message string
}

func main() {
	asb := azureservicebus.Create(os.Getenv("CONSTRING"))
	endpoint := servicebus.Create("awesomeservice", asb)
	endpoint.Message("MyMessage").
		AsIncoming().
		Handle(myMessageHandle)

	err := endpoint.Start()
	if err != nil {
		panic(err)
	}
	_, err = endpoint.SendLocal("MyMessage", &MyMessage{
		Message: "Hallo Welt!",
	})
	if err != nil {
		panic(err)
	}

	<-make(chan bool)
}

func myMessageHandle(ctx *servicebus.IncomingMessageContext) {
	fmt.Println("MyMessage received!")
}

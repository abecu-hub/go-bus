package main

import (
	"context"
	"fmt"
	"github.com/Azure/azure-service-bus-go"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	ns, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(
		os.Getenv("CONSTRING")))
	if err != nil {
		panic("Namespace failed")
	}
	qm := ns.NewQueueManager()
	source, err := ensureQueue(ctx, qm, "awesomeService")
	if err != nil {
		fmt.Println(err)
		return
	}
	sourceQueue, err := ns.NewQueue(source.Name)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		_ = sourceQueue.Close(ctx)
	}()

	if err := sourceQueue.Send(ctx, servicebus.NewMessageFromString("Hello World!")); err != nil {
		fmt.Println(err)
		return
	}
}

func ensureQueue(ctx context.Context, qm *servicebus.QueueManager, name string, opts ...servicebus.QueueManagementOption) (*servicebus.QueueEntity, error) {
	qe, err := qm.Get(ctx, name)
	if err == nil {
		_ = qm.Delete(ctx, name)
	}

	qe, err = qm.Put(ctx, name, opts...)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return qe, nil
}

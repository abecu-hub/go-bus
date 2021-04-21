package serviceBus

import (
	"dev.azure.com/finorun/Playground/_git/go-bus.git/pkg/serviceBus/saga"
)

func UseSagas(store saga.Store) func(endpoint *Endpoint) error {
	return func(endpoint *Endpoint) error {
		endpoint.SagaStore = store
		return nil
	}
}

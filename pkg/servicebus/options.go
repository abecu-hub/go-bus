package servicebus

import (
	"github.com/abecu-hub/go-bus/pkg/servicebus/saga"
)

func UseSagas(store saga.Store) func(endpoint *Endpoint) {
	return func(endpoint *Endpoint) {
		endpoint.SagaStore = store
	}
}

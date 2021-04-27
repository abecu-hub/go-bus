package servicebus

import (
	"github.com/abecu-hub/go-bus/pkg/servicebus/saga"
)

func UseSagas(store saga.Store) func(endpoint *ServiceBusEndpoint) {
	return func(endpoint *ServiceBusEndpoint) {
		endpoint.sagaStore = store
	}
}

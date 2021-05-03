package retrypolicy

import (
	"github.com/abecu-hub/go-bus/pkg/servicebus"
	"time"
)

//Multiplies the retry count with the given backoff duration to gradually reduce the retry frequency.
func Backoff(backoffDuration time.Duration) servicebus.RetryPolicy {
	return func(retryCount int, retry func() error) error {
		time.Sleep(backoffDuration * time.Duration(retryCount))
		return retry()
	}
}

//Waits for the given duration until the next retry.
func Simple(duration time.Duration) servicebus.RetryPolicy {
	return func(retryCount int, retry func() error) error {
		time.Sleep(duration)
		return retry()
	}
}

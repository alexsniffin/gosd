package gosd

import (
	"time"
)

// ScheduledMessage is a message to schedule with the Dispatchers ingest channel
// `At` is when the message will dispatched
// `Message` is the content of the message
type ScheduledMessage[T any] struct {
	At      time.Time
	Message T
}

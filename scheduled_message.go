package gosd

import (
	"time"
)

type ScheduledMessage struct {
	At      time.Time
	Message interface{}
}

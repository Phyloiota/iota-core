package clock

import (
	"time"

	"github.com/iotaledger/hive.go/runtime/event"
)

// ClockEvents contains a dictionary of events that are triggered by the Clock.
type ClockEvents struct {
	// AcceptedTimeUpdated is triggered when the accepted time is updated.
	AcceptedTimeUpdated *event.Event1[time.Time]

	// ConfirmedTimeUpdated is triggered when the confirmed time is updated.
	ConfirmedTimeUpdated *event.Event1[time.Time]

	// Group is trait that makes the dictionary linkable.
	event.Group[ClockEvents, *ClockEvents]
}

// NewClockEvents is the constructor of the ClockEvents object.
var NewClockEvents = event.CreateGroupConstructor(func() *ClockEvents {
	return &ClockEvents{
		AcceptedTimeUpdated:  event.New1[time.Time](),
		ConfirmedTimeUpdated: event.New1[time.Time](),
	}
})

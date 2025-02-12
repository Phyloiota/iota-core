package mempoolv1

import (
	"github.com/iotaledger/iota-core/pkg/core/promise"
)

// inclusionFlags represents important flags and events that relate to the inclusion of an entity in the distributed ledger.
type inclusionFlags struct {
	// accepted gets triggered when the entity gets marked as accepted.
	accepted *promise.Value[bool]

	// committed gets triggered when the entity gets marked as committed.
	committed *promise.Event

	// rejected gets triggered when the entity gets marked as rejected.
	rejected *promise.Event

	// orphaned gets triggered when the entity gets marked as orphaned.
	orphaned *promise.Event
}

// newInclusionFlags creates a new inclusionFlags instance.
func newInclusionFlags() *inclusionFlags {
	return &inclusionFlags{
		accepted:  promise.NewValue[bool](),
		committed: promise.NewEvent(),
		rejected:  promise.NewEvent(),
		orphaned:  promise.NewEvent(),
	}
}

func (s *inclusionFlags) IsPending() bool {
	return !s.accepted.Get()
}

// IsAccepted returns true if the entity was accepted.
func (s *inclusionFlags) IsAccepted() bool {
	return s.accepted.Get()
}

// OnAccepted registers a callback that gets triggered when the entity gets accepted.
func (s *inclusionFlags) OnAccepted(callback func()) {
	s.accepted.OnUpdate(func(wasAccepted, isAccepted bool) {
		if isAccepted && !wasAccepted {
			callback()
		}
	})
}

// OnPending registers a callback that gets triggered when the entity gets pending.
func (s *inclusionFlags) OnPending(callback func()) {
	s.accepted.OnUpdate(func(wasAccepted, isAccepted bool) {
		if !isAccepted && wasAccepted {
			callback()
		}
	})
}

// IsRejected returns true if the entity was rejected.
func (s *inclusionFlags) IsRejected() bool {
	return s.rejected.WasTriggered()
}

// OnRejected registers a callback that gets triggered when the entity gets rejected.
func (s *inclusionFlags) OnRejected(callback func()) {
	s.rejected.OnTrigger(callback)
}

// IsCommitted returns true if the entity was committed.
func (s *inclusionFlags) IsCommitted() bool {
	return s.committed.WasTriggered()
}

// OnCommitted registers a callback that gets triggered when the entity gets committed.
func (s *inclusionFlags) OnCommitted(callback func()) {
	s.committed.OnTrigger(callback)
}

// IsOrphaned returns true if the entity was orphaned.
func (s *inclusionFlags) IsOrphaned() bool {
	return s.orphaned.WasTriggered()
}

// OnOrphaned registers a callback that gets triggered when the entity gets orphaned.
func (s *inclusionFlags) OnOrphaned(callback func()) {
	s.orphaned.OnTrigger(callback)
}

// setAccepted marks the entity as accepted.
func (s *inclusionFlags) setAccepted() {
	s.accepted.Set(true)
}

// setPending marks the entity as pending.
func (s *inclusionFlags) setPending() {
	s.accepted.Set(false)
}

// setRejected marks the entity as rejected.
func (s *inclusionFlags) setRejected() {
	s.rejected.Trigger()
}

// setCommitted marks the entity as committed.
func (s *inclusionFlags) setCommitted() {
	s.committed.Trigger()
}

// setOrphaned marks the entity as orphaned.
func (s *inclusionFlags) setOrphaned() {
	s.orphaned.Trigger()
}

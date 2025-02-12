package weight

import (
	"sync"

	"github.com/iotaledger/hive.go/core/account"
	"github.com/iotaledger/hive.go/runtime/event"
	"github.com/iotaledger/hive.go/stringify"
	"github.com/iotaledger/iota-core/pkg/core/acceptance"
	iotago "github.com/iotaledger/iota.go/v4"
)

// Weight represents a mutable multi-tiered weight value that can be updated in-place.
type Weight struct {
	// OnUpdate is an event that is triggered when the weight value is updated.
	OnUpdate *event.Event1[Value]

	// Voters is the set of validators that are contributing to the validators weight.
	Voters *account.SelectedAccounts[iotago.AccountID, *iotago.AccountID]

	// value is the current weight Value.
	value Value

	// mutex is used to synchronize access to the weight value.
	mutex sync.RWMutex
}

// New creates a new Weight instance.
func New(weights *account.SelectedAccounts[iotago.AccountID, *iotago.AccountID]) *Weight {
	w := &Weight{
		Voters:   weights.SelectAccounts(),
		OnUpdate: event.New1[Value](),
	}

	return w
}

// CumulativeWeight returns the cumulative weight of the Weight.
func (w *Weight) CumulativeWeight() int64 {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	return w.value.CumulativeWeight()
}

// SetCumulativeWeight sets the cumulative weight of the Weight and returns the Weight (for chaining).
func (w *Weight) SetCumulativeWeight(cumulativeWeight int64) *Weight {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.value.CumulativeWeight() != cumulativeWeight {
		w.value = w.value.SetCumulativeWeight(cumulativeWeight)
		w.OnUpdate.Trigger(w.value)
	}

	return w
}

// AddCumulativeWeight adds the given weight to the cumulative weight and returns the Weight (for chaining).
func (w *Weight) AddCumulativeWeight(delta int64) *Weight {
	if delta != 0 {
		w.mutex.Lock()
		defer w.mutex.Unlock()

		w.value = w.value.AddCumulativeWeight(delta)
		w.OnUpdate.Trigger(w.value)
	}

	return w
}

// RemoveCumulativeWeight removes the given weight from the cumulative weight and returns the Weight (for chaining).
func (w *Weight) RemoveCumulativeWeight(delta int64) *Weight {
	if delta != 0 {
		w.mutex.Lock()
		defer w.mutex.Unlock()

		w.value = w.value.RemoveCumulativeWeight(delta)
		w.OnUpdate.Trigger(w.value)
	}

	return w
}

// AddVoter adds the given voter to the list of Voters, updates the weight and returns the Weight (for chaining).
func (w *Weight) AddVoter(id iotago.AccountID) *Weight {
	if w.Voters.Add(id) {
		if newValue, valueUpdated := w.updateValidatorsWeight(); valueUpdated {
			w.OnUpdate.Trigger(newValue)
		}
	}

	return w
}

// DeleteVoter removes the given voter from the list of Voters, updates the weight and returns the Weight (for chaining).
func (w *Weight) DeleteVoter(id iotago.AccountID) *Weight {
	if w.Voters.Delete(id) {
		if newValue, valueUpdated := w.updateValidatorsWeight(); valueUpdated {
			w.OnUpdate.Trigger(newValue)
		}
	}

	return w
}

func (w *Weight) updateValidatorsWeight() (Value, bool) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	newValidatorWeight := w.Voters.TotalWeight()
	if w.value.ValidatorsWeight() != newValidatorWeight {
		w.value = w.value.SetValidatorsWeight(newValidatorWeight)

		return w.value, true
	}

	return w.value, false
}

// AcceptanceState returns the acceptance state of the weight.
func (w *Weight) AcceptanceState() acceptance.State {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	return w.value.AcceptanceState()
}

// SetAcceptanceState sets the acceptance state of the weight and returns the previous acceptance state.
func (w *Weight) SetAcceptanceState(acceptanceState acceptance.State) (previousState acceptance.State) {
	if previousState = w.setAcceptanceState(acceptanceState); previousState != acceptanceState {
		w.OnUpdate.Trigger(w.value)
	}

	return previousState
}

// WithAcceptanceState sets the acceptance state of the weight and returns the Weight instance.
func (w *Weight) WithAcceptanceState(acceptanceState acceptance.State) *Weight {
	w.setAcceptanceState(acceptanceState)

	return w
}

// Value returns an immutable copy of the Weight.
func (w *Weight) Value() Value {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	return w.value
}

// Compare compares the Weight to the given other Weight.
func (w *Weight) Compare(other *Weight) Comparison {
	switch {
	case w == nil && other == nil:
		return Equal
	case w == nil:
		return Heavier
	case other == nil:
		return Lighter
	default:
		return w.value.Compare(other.value)
	}
}

// String returns a human-readable representation of the Weight.
func (w *Weight) String() string {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	return stringify.Struct("Weight",
		stringify.NewStructField("Value", w.value),
		stringify.NewStructField("Voters", w.Voters.Members()),
	)
}

// setAcceptanceState sets the acceptance state of the weight and returns the previous acceptance state.
func (w *Weight) setAcceptanceState(acceptanceState acceptance.State) (previousState acceptance.State) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if previousState = w.value.AcceptanceState(); previousState != acceptanceState {
		w.value = w.value.SetAcceptanceState(acceptanceState)
	}

	return previousState
}

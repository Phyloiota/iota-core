package ledger

import (
	"github.com/iotaledger/iota-core/pkg/interfaces/ledger/mempool"
	"github.com/iotaledger/iota-core/pkg/slot"
)

// StateDiffs is a submodule that provides access to the state diffs of the ledger state.
type StateDiffs interface {
	// ForEachCreatedOutput streams the created outputs of the given slot index.
	ForEachCreatedOutput(slot.Index, func(output *mempool.OutputWithMetadata) error) error

	// ForEachSpentOutput streams the spent outputs of the given slot index.
	ForEachSpentOutput(slot.Index, func(output *mempool.OutputWithMetadata) error) error
}

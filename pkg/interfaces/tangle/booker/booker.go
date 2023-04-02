package booker

import (
	"github.com/iotaledger/goshimmer/packages/core/votes/sequencetracker"
	"github.com/iotaledger/goshimmer/packages/protocol/markers"
	"github.com/iotaledger/goshimmer/packages/protocol/models"
	"github.com/iotaledger/iota-core/pkg/interfaces/ledger/utxo"
	"github.com/iotaledger/iota-core/pkg/module"
	"github.com/iotaledger/iota-core/pkg/slot"
)

type Booker interface {
	MarkerManager() MarkerManager

	SlotTracker() SlotTracker

	// Block retrieves a Block with metadata from the in-memory storage of the Booker.
	Block(id models.BlockID) (block *Block, exists bool)

	// BlockConflicts returns the Conflict related details of the given Block.
	BlockConflicts(block *Block) (blockConflictIDs utxo.TransactionIDs)

	// LatestAttachment returns the latest attachment for a given transaction ID.
	// returnOrphaned parameter specifies whether the returned attachment may be orphaned.
	LatestAttachment(txID utxo.TransactionID) (attachment *Block)

	module.Interface
}

type SlotTracker interface {
	// SlotVotersTotalWeight retrieves the total weight of the Validators voting for a given slot.
	SlotVotersTotalWeight(slotIndex slot.Index) int64
}

type MarkerManager interface {
	SequenceManager() *markers.SequenceManager

	SequenceTracker() *sequencetracker.SequenceTracker[BlockVotePower]

	// BlockCeiling returns the smallest Index that is >= the given Marker and a boolean value indicating if it exists.
	BlockCeiling(marker markers.Marker) (ceilingMarker markers.Marker, exists bool)

	// BlockFromMarker retrieves the Block of the given Marker.
	BlockFromMarker(marker markers.Marker) (block *Block, exists bool)

	// MarkerVotersTotalWeight retrieves Validators supporting a given marker.
	MarkerVotersTotalWeight(marker markers.Marker) int64

	FirstUnacceptedIndex(sequenceID markers.SequenceID) markers.Index

	SetAccepted(marker markers.Marker) (updated bool)
}

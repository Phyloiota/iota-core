package consensus

import (
	"github.com/iotaledger/iota-core/pkg/models"
	"github.com/iotaledger/iota-core/pkg/module"
	"github.com/iotaledger/iota-core/pkg/slot"
)

type Consensus interface {
	BlockGadget() BlockGadget

	SlotGadget() SlotGadget

	module.Interface
}

type BlockGadget interface {
	// IsBlockAccepted returns whether the given block is accepted.
	IsBlockAccepted(blockID models.BlockID) (accepted bool)

	IsBlockConfirmed(blockID models.BlockID) bool

	module.Interface
}

type SlotGadget interface {
	LastConfirmedSlot() slot.Index

	module.Interface
}

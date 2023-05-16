package therealledger

import (
	"io"

	"github.com/iotaledger/hive.go/runtime/module"
	"github.com/iotaledger/iota-core/pkg/protocol/engine/blocks"
	"github.com/iotaledger/iota-core/pkg/protocol/engine/ledgerstate"
	"github.com/iotaledger/iota-core/pkg/protocol/engine/mempool"
	iotago "github.com/iotaledger/iota.go/v4"
)

type Ledger interface {
	AttachTransaction(block *blocks.Block) (transactionMetadata mempool.TransactionMetadata, containsTransaction bool)
	Output(id iotago.OutputID) (*ledgerstate.Output, error)
	CommitSlot(index iotago.SlotIndex) (stateRoot iotago.Identifier, mutationRoot iotago.Identifier, err error)

	Import(reader io.ReadSeeker) error
	Export(writer io.WriteSeeker, targetIndex iotago.SlotIndex) error

	module.Interface
}
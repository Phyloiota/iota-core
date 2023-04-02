package ledger

import (
	"github.com/iotaledger/hive.go/ds/types"
	"github.com/iotaledger/iota-core/pkg/interfaces/ledger/mempool"
	"github.com/iotaledger/iota-core/pkg/interfaces/ledger/utxo"
	"github.com/iotaledger/iota-core/pkg/module"
)

// UnspentOutputs is a submodule that provides access to the unspent outputs of the ledger state.
type UnspentOutputs interface {
	// AddOutput applies the given output to the unspent outputs.
	AddOutput(output *mempool.OutputWithMetadata) error

	Output(id utxo.OutputID) (*mempool.OutputWithMetadata, error)

	ForEach(func(output *mempool.OutputWithMetadata) error) error

	Root() types.Identifier

	// Interface embeds the required methods of the module.Interface.
	module.Interface
}

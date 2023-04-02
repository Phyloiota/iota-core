package tangle

import (
	"github.com/iotaledger/iota-core/pkg/interfaces/tangle/blockdag"
	"github.com/iotaledger/iota-core/pkg/interfaces/tangle/booker"
	"github.com/iotaledger/iota-core/pkg/module"
)

type Tangle interface {
	BlockDAG() blockdag.BlockDAG

	Booker() booker.Booker

	module.Interface
}

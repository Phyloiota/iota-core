package filter

import (
	"github.com/iotaledger/goshimmer/packages/protocol/models"
	"github.com/iotaledger/hive.go/crypto/identity"
	"github.com/iotaledger/iota-core/pkg/module"
)

type Filter interface {
	// ProcessReceivedBlock processes block from the given source.
	ProcessReceivedBlock(block *models.Block, source identity.ID)

	module.Interface
}

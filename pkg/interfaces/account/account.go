package account

import (
	"github.com/iotaledger/hive.go/crypto/identity"
	"github.com/iotaledger/hive.go/ds/types"
	"github.com/iotaledger/iota-core/pkg/slot"
)

type Weight interface {
	Value() int64
	UpdateTime() slot.Index
}

type AuthenticatedAccounts interface {
	Accounts
	Root() types.Identifier
}

type Accounts interface {
	Get(id identity.ID) (weight Weight, exists bool)
	Has(id identity.ID) bool

	ForEach(callback func(id identity.ID, weight Weight) error) error

	TotalWeight() int64
}

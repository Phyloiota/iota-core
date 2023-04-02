package throughputquota

import (
	"github.com/iotaledger/hive.go/crypto/identity"
	"github.com/iotaledger/iota-core/pkg/module"
)

type ThroughputQuota interface {
	// Balance returns the balance of the given identity.
	Balance(id identity.ID) (mana int64, exists bool)

	// TotalBalance returns the total amount of throughput quota.
	TotalBalance() (totalQuota int64)

	// Interface embeds the required methods of the module.Interface.
	module.Interface
}

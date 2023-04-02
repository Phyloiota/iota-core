package notarization

import (
	"io"

	"github.com/iotaledger/hive.go/ads"
	"github.com/iotaledger/hive.go/crypto/identity"
	"github.com/iotaledger/iota-core/pkg/module"
	"github.com/iotaledger/iota-core/pkg/slot"
)

type Notarization interface {
	Attestations() Attestations

	// IsBootstrapped returns if notarization finished committing all pending slots up to the current acceptance time.
	IsBootstrapped() bool

	Import(reader io.ReadSeeker) (err error)

	Export(writer io.WriteSeeker, targetSlot slot.Index) (err error)

	//TODO: Try to remove this from engine switching
	PerformLocked(perform func(m Notarization))

	module.Interface
}

type Attestations interface {
	Get(index slot.Index) (attestations *ads.Map[identity.ID, Attestation, *identity.ID, *Attestation], err error)

	// LastCommittedSlot returns the last committed slot.
	LastCommittedSlot() (index slot.Index)

	module.Interface
}

package sybilprotection

import (
	"github.com/iotaledger/iota-core/pkg/interfaces/account"
	"github.com/iotaledger/iota-core/pkg/module"
	"github.com/iotaledger/iota-core/pkg/slot"
)

// SybilProtection is the minimal interface for the SybilProtection component of the IOTA protocol.
type SybilProtection interface {
	// Validators returns the weights of identities in the SybilProtection.
	Validators() account.AuthenticatedAccounts
	/*
		The weights represent an account based ledger and the weightedset represent subset of those accounts.
		The account based ledger can be notified of updates to be applied to its ledger through the `WeightsUpdated` event.

	*/

	// Committee returns the set of validators that is used to track confirmation.
	Committee() account.Accounts

	// OnlineCommittee returns the set of online validators that is used to track acceptance.
	OnlineCommittee() account.Accounts

	// LastCommittedSlot returns the last committed slot.
	LastCommittedSlot() slot.Index

	// Interface embeds the required methods of the module.Interface.
	module.Interface
}

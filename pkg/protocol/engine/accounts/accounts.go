package accounts

import (
	"crypto"
	"crypto/ed25519"

	"github.com/iotaledger/hive.go/ds/shrinkingmap"
	"github.com/iotaledger/hive.go/ds/types"
	"github.com/iotaledger/hive.go/runtime/module"
	iotago "github.com/iotaledger/iota.go/v4"
)

// TODO do we still need this separate interface, together with ledger
// BlockIssuanceCredits is the minimal interface for the Accounts component of the IOTA protocol.
type BlockIssuanceCredits interface {
	// BIC returns Block Issuer Credits of a specific account for a specific slot index.
	BIC(id iotago.AccountID, slot iotago.SlotIndex) (account *Account, err error)

	// Interface embeds the required methods of the module.Interface.
	module.Interface
}

type Holdings interface {
	// Mana is the stored and potential value of an account collected on the UTXO layer - used by the Scheduler.
	Mana(id iotago.AccountID) (mana *Mana, err error)
}

type Account interface {
	ID() iotago.AccountID
	Credits() *Credits
	IsPublicKeyAllowed(ed25519.PublicKey) bool
}

type AccountImpl struct {
	id         iotago.AccountID
	credits    *Credits
	pubKeysMap *shrinkingmap.ShrinkingMap[crypto.PublicKey, types.Empty]
}

func NewAccount(id iotago.AccountID, credits *Credits, pubKeys []ed25519.PublicKey) *AccountImpl {
	pubKeysMap := shrinkingmap.New[crypto.PublicKey, types.Empty](shrinkingmap.WithShrinkingThresholdCount(10))
	for _, pubKey := range pubKeys {
		_ = pubKeysMap.Set(pubKey, types.Void)
	}

	return &AccountImpl{
		id:         id,
		credits:    credits,
		pubKeysMap: pubKeysMap,
	}
}

func (a *AccountImpl) ID() iotago.AccountID {
	return a.id
}

func (a *AccountImpl) Credits() *Credits {
	return a.credits
}

func (a *AccountImpl) AddPublicKey(pubKey ed25519.PublicKey) {
	_ = a.pubKeysMap.Set(pubKey, types.Void)
}

func (a *AccountImpl) RemovePublicKey(pubKey ed25519.PublicKey) {
	_ = a.pubKeysMap.Delete(pubKey)
}

func (a *AccountImpl) IsPublicKeyAllowed(pubKey ed25519.PublicKey) bool {
	return a.pubKeysMap.Has(pubKey)
}

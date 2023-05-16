package utxoledger

import (
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/iota-core/pkg/protocol/engine/ledger"
	iotago "github.com/iotaledger/iota.go/v4"
)

type Transaction struct {
	Transaction *iotago.Transaction
}

func (t *Transaction) ID() (iotago.TransactionID, error) {
	return t.Transaction.ID()
}

func (t *Transaction) Inputs() ([]ledger.StateReference, error) {
	references := make([]ledger.StateReference, len(t.Transaction.Essence.Inputs))
	for i, input := range t.Transaction.Essence.Inputs {
		inputReferencer, ok := input.(iotago.IndexedUTXOReferencer)
		if !ok {
			return nil, ErrUnexpectedUnderlyingType
		}

		references[i] = ledger.StoredStateReference(inputReferencer.Ref())
	}

	return references, nil
}

func (t *Transaction) String() string {
	return "iotago.Transaction(" + lo.PanicOnErr(t.ID()).ToHex() + ")"
}
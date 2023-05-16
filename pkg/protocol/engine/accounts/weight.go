package accounts

import (
	"context"

	"github.com/iotaledger/hive.go/serializer/v2/serix"
	iotago "github.com/iotaledger/iota.go/v4"
)

// Credits is a weight annotated with the slot it was last updated in.
type Credits struct {
	Value      int64            `serix:"0"`
	UpdateTime iotago.SlotIndex `serix:"1"`
}

// NewWeight creates a new Credits instance.
func NewWeight(value int64, updateTime iotago.SlotIndex) (newWeight *Credits) {
	return &Credits{
		Value:      value,
		UpdateTime: updateTime,
	}
}

// Bytes returns a serialized version of the Credits.
func (w Credits) Bytes() ([]byte, error) {
	return serix.DefaultAPI.Encode(context.Background(), w)
}

// FromBytes parses a serialized version of the Credits.
func (w *Credits) FromBytes(bytes []byte) (int, error) {
	return serix.DefaultAPI.Decode(context.Background(), bytes, w)
}
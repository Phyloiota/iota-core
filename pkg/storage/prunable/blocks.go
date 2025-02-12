package prunable

import (
	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/iota-core/pkg/model"
	iotago "github.com/iotaledger/iota.go/v4"
)

type Blocks struct {
	slot  iotago.SlotIndex
	store kvstore.KVStore

	api iotago.API
}

func NewBlocks(slot iotago.SlotIndex, store kvstore.KVStore, api iotago.API) (newBlocks *Blocks) {
	return &Blocks{
		slot:  slot,
		store: store,
		api:   api,
	}
}

func (b *Blocks) Load(id iotago.BlockID) (*model.Block, error) {
	blockBytes, err := b.store.Get(id[:])
	if err != nil {
		if errors.Is(err, kvstore.ErrKeyNotFound) {
			//nolint:nilnil // expected behavior
			return nil, nil
		}

		return nil, errors.Wrapf(err, "failed to get block %s", id)
	}

	return model.BlockFromIDAndBytes(id, blockBytes, b.api)
}

func (b *Blocks) Store(block *model.Block) error {
	blockID := block.ID()
	return b.store.Set(blockID[:], block.Data())
}

func (b *Blocks) Delete(id iotago.BlockID) (err error) {
	return b.store.Delete(id[:])
}

func (b *Blocks) ForEachBlockIDInSlot(consumer func(blockID iotago.BlockID) error) error {
	var innerErr error
	if err := b.store.IterateKeys(kvstore.EmptyPrefix, func(key kvstore.Key) bool {
		var blockID iotago.BlockID
		blockID, innerErr = iotago.SlotIdentifierFromBytes(key)
		if innerErr != nil {
			return false
		}

		return consumer(blockID) != nil
	}); err != nil {
		return errors.Wrapf(err, "failed to stream blockIDs for slot %s", b.slot)
	}

	if innerErr != nil {
		return errors.Wrapf(innerErr, "failed to deserialize blockIDs for slot %s", b.slot)
	}

	return nil
}

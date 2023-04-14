package blockdag

import (
	"fmt"
	"sync"
	"time"

	"github.com/iotaledger/hive.go/ds/types"
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/hive.go/stringify"
	"github.com/iotaledger/iota-core/pkg/model"
	iotago "github.com/iotaledger/iota.go/v4"
)

// Block represents a Block annotated with Tangle related metadata.
type Block struct {
	missing        bool
	missingBlockID iotago.BlockID

	solid               bool
	invalid             bool
	orphaned            bool
	future              bool
	strongChildren      []*Block
	weakChildren        []*Block
	shallowLikeChildren []*Block
	mutex               sync.RWMutex

	*rootBlock
	*ModelsBlock
}

type rootBlock struct {
	blockID      iotago.BlockID
	commitmentID iotago.CommitmentID
	issuingTime  time.Time
}

type ModelsBlock = model.Block

// NewBlock creates a new Block with the given options.
func NewBlock(data *model.Block) *Block {
	return &Block{
		ModelsBlock: data,
	}
}

func NewRootBlock(blockID iotago.BlockID, commitmentID iotago.CommitmentID, issuingTime time.Time) *Block {
	return &Block{
		rootBlock: &rootBlock{
			blockID:      blockID,
			commitmentID: commitmentID,
			issuingTime:  issuingTime,
		},
		solid: true,
	}
}

func NewMissingBlock(blockID iotago.BlockID) *Block {
	return &Block{
		missing:        true,
		missingBlockID: blockID,
	}
}

func (b *Block) ID() iotago.BlockID {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	if b.missing {
		return b.missingBlockID
	}

	if b.rootBlock != nil {
		return b.rootBlock.blockID
	}

	return b.ModelsBlock.ID()
}

func (b *Block) IssuingTime() time.Time {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	if b.missing {
		return time.Time{}
	}

	if b.rootBlock != nil {
		return b.rootBlock.issuingTime
	}

	return b.ModelsBlock.IssuingTime()
}

func (b *Block) SlotCommitmentID() iotago.CommitmentID {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	if b.missing {
		return iotago.CommitmentID{}
	}

	if b.rootBlock != nil {
		return b.rootBlock.commitmentID
	}

	return b.ModelsBlock.SlotCommitmentID()
}

// IsMissing returns a flag that indicates if the underlying Block data hasn't been stored, yet.
func (b *Block) IsMissing() (isMissing bool) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.missing
}

// IsSolid returns true if the Block is solid (the entire causal history is known).
func (b *Block) IsSolid() (isSolid bool) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.solid
}

// IsInvalid returns true if the Block was marked as invalid.
func (b *Block) IsInvalid() (isInvalid bool) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.invalid
}

// IsFuture returns true if the Block is a future Block (we haven't committed to its commitment slot yet).
func (b *Block) IsFuture() (isFuture bool) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.future
}

// SetFuture marks the Block as future block.
func (b *Block) SetFuture() (wasUpdated bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.future {
		return false
	}

	b.future = true
	return true
}

// IsOrphaned returns true if the Block is orphaned (either due to being marked as orphaned itself or because it has
// orphaned Blocks in its past cone).
func (b *Block) IsOrphaned() (isOrphaned bool) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.orphaned
}

// Children returns the children of the Block.
func (b *Block) Children() (children []*Block) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	seenBlockIDs := make(map[iotago.BlockID]types.Empty)
	for _, parentsByType := range [][]*Block{
		b.strongChildren,
		b.weakChildren,
		b.shallowLikeChildren,
	} {
		for _, childMetadata := range parentsByType {
			if _, exists := seenBlockIDs[childMetadata.ID()]; !exists {
				children = append(children, childMetadata)
				seenBlockIDs[childMetadata.ID()] = types.Void
			}
		}
	}

	return children
}

func (b *Block) StrongChildren() []*Block {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return lo.CopySlice(b.strongChildren)
}

func (b *Block) WeakChildren() []*Block {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return lo.CopySlice(b.weakChildren)
}

func (b *Block) ShallowLikeChildren() []*Block {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return lo.CopySlice(b.shallowLikeChildren)
}

// SetSolid marks the Block as solid.
func (b *Block) SetSolid() (wasUpdated bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if wasUpdated = !b.solid; wasUpdated {
		b.solid = true
	}

	return
}

// SetInvalid marks the Block as invalid.
func (b *Block) SetInvalid() (wasUpdated bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.invalid {
		return false
	}

	b.invalid = true

	return true
}

// SetOrphaned sets the orphaned flag of the Block.
func (b *Block) SetOrphaned(orphaned bool) (wasUpdated bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.orphaned == orphaned {
		return false
	}
	b.orphaned = orphaned

	return true
}

func (b *Block) AppendChild(child *Block, childType model.ParentsType) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	switch childType {
	case model.StrongParentType:
		b.strongChildren = append(b.strongChildren, child)
	case model.WeakParentType:
		b.weakChildren = append(b.weakChildren, child)
	case model.ShallowLikeParentType:
		b.shallowLikeChildren = append(b.shallowLikeChildren, child)
	}
}

// Update publishes the given Block data to the underlying Block and marks it as no longer missing.
func (b *Block) Update(data *model.Block) (wasPublished bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if !b.missing {
		return
	}

	b.ModelsBlock = data
	b.missing = false

	return true
}

func (b *Block) String() string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	builder := stringify.NewStructBuilder("BlockDAG.Block", stringify.NewStructField("id", b.ID()))
	builder.AddField(stringify.NewStructField("Missing", b.missing))
	builder.AddField(stringify.NewStructField("Solid", b.solid))
	builder.AddField(stringify.NewStructField("Invalid", b.invalid))
	builder.AddField(stringify.NewStructField("Orphaned", b.orphaned))

	for index, child := range b.strongChildren {
		builder.AddField(stringify.NewStructField(fmt.Sprintf("strongChildren%d", index), child.ID().String()))
	}

	for index, child := range b.weakChildren {
		builder.AddField(stringify.NewStructField(fmt.Sprintf("weakChildren%d", index), child.ID().String()))
	}

	for index, child := range b.shallowLikeChildren {
		builder.AddField(stringify.NewStructField(fmt.Sprintf("shallowLikeChildren%d", index), child.ID().String()))
	}

	builder.AddField(stringify.NewStructField("ModelsBlock", b.ModelsBlock))

	return builder.String()
}
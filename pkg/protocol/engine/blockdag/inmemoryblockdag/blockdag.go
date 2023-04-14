package inmemoryblockdag

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/core/causalorder"
	"github.com/iotaledger/hive.go/core/memstorage"
	"github.com/iotaledger/hive.go/ds/advancedset"
	"github.com/iotaledger/hive.go/ds/walker"
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/hive.go/runtime/event"
	"github.com/iotaledger/hive.go/runtime/module"
	"github.com/iotaledger/hive.go/runtime/options"
	"github.com/iotaledger/hive.go/runtime/workerpool"
	"github.com/iotaledger/iota-core/pkg/model"
	"github.com/iotaledger/iota-core/pkg/protocol/engine"
	"github.com/iotaledger/iota-core/pkg/protocol/engine/blockdag"
	"github.com/iotaledger/iota-core/pkg/protocol/engine/eviction"
	iotago "github.com/iotaledger/iota.go/v4"
)

// region BlockDAG /////////////////////////////////////////////////////////////////////////////////////////////////////

// BlockDAG is a causally ordered DAG that forms the central data structure of the IOTA protocol.
type BlockDAG struct {
	// Events contains the Events of the BlockDAG.
	events *blockdag.Events

	// evictionState contains information about the current eviction state.
	evictionState *eviction.State

	// memStorage contains the in-memory storage of the BlockDAG.
	memStorage *memstorage.IndexedStorage[iotago.SlotIndex, iotago.BlockID, *blockdag.Block]

	// solidifier contains the solidifier instance used to determine the solidity of Blocks.
	solidifier *causalorder.CausalOrder[iotago.SlotIndex, iotago.BlockID, *blockdag.Block]

	// commitmentFunc is a function that returns the commitment corresponding to the given slot index.
	commitmentFunc func(index iotago.SlotIndex) (*iotago.Commitment, error)

	// futureBlocks contains blocks with a commitment in the future, that should not be passed to the booker yet.
	futureBlocks *memstorage.IndexedStorage[iotago.SlotIndex, iotago.CommitmentID, *advancedset.AdvancedSet[*blockdag.Block]]

	nextIndexToPromote iotago.SlotIndex

	// The Queue always read-locks the eviction mutex of the solidifier, and then evaluates if the block is
	// future thus read-locking the futureBlocks mutex. At the same time, when re-adding parked blocks,
	// promoteFutureBlocksMethod write-locks the futureBlocks mutex, and then read-locks the eviction mutex
	// of the solidifier. As the locks are non-starving, and locks are interlocked in different orders a
	// deadlock can occur only when an eviction is triggered while the above scenario unfolds.
	solidifierMutex sync.RWMutex

	futureBlocksMutex sync.RWMutex

	// evictionMutex is a mutex that is used to synchronize the eviction of elements from the BlockDAG.
	evictionMutex sync.RWMutex

	rootBlockProvider func(iotago.BlockID) (*blockdag.Block, bool)

	workers    *workerpool.Group
	workerPool *workerpool.WorkerPool

	module.Module
}

func NewProvider(opts ...options.Option[BlockDAG]) module.Provider[*engine.Engine, blockdag.BlockDAG] {
	return module.Provide(func(e *engine.Engine) blockdag.BlockDAG {
		b := New(e.Workers.CreateGroup("BlockDAG"), e.EvictionState, e.Storage.Commitments.Load, opts...)

		e.HookConstructed(func() {
			e.Events.Filter.BlockAllowed.Hook(func(block *model.Block) {
				if _, _, err := b.Attach(block); err != nil {
					e.Events.Error.Trigger(errors.Wrapf(err, "failed to attach block with %s (issuerID: %s)", block.ID(), block.Block().IssuerID))
				}
			}, event.WithWorkerPool(b.workers.CreatePool("BlockDAG.Attach", 2)))

			e.Events.BlockDAG.LinkTo(b.events)

			b.Initialize(e.RootBlock)
		})

		return b
	})
}

// New is the constructor for the BlockDAG and creates a new BlockDAG instance.
func New(workers *workerpool.Group, evictionState *eviction.State, latestCommitmentFunc func(iotago.SlotIndex) (*iotago.Commitment, error), opts ...options.Option[BlockDAG]) (newBlockDAG *BlockDAG) {
	return options.Apply(&BlockDAG{
		events:         blockdag.NewEvents(),
		evictionState:  evictionState,
		memStorage:     memstorage.NewIndexedStorage[iotago.SlotIndex, iotago.BlockID, *blockdag.Block](),
		commitmentFunc: latestCommitmentFunc,
		futureBlocks:   memstorage.NewIndexedStorage[iotago.SlotIndex, iotago.CommitmentID, *advancedset.AdvancedSet[*blockdag.Block]](),
		workers:        workers,
		workerPool:     workers.CreatePool("Solidifier", 2),
	}, opts,
		func(b *BlockDAG) {
			b.solidifier = causalorder.New(
				b.workerPool,
				b.Block,
				(*blockdag.Block).IsSolid,
				b.markSolid,
				b.markInvalid,
				(*blockdag.Block).Parents,
				causalorder.WithReferenceValidator[iotago.SlotIndex, iotago.BlockID](checkReference),
			)

			evictionState.Events.SlotEvicted.Hook(b.evictSlot, event.WithWorkerPool(b.workerPool))
		},
		(*BlockDAG).TriggerConstructed,
		(*BlockDAG).TriggerInitialized,
	)
}

func (b *BlockDAG) Initialize(rootBlockProvider func(iotago.BlockID) (*blockdag.Block, bool)) {
	b.rootBlockProvider = rootBlockProvider

	b.TriggerInitialized()
}

var _ blockdag.BlockDAG = new(BlockDAG)

func (b *BlockDAG) Events() *blockdag.Events {
	return b.events
}

func (b *BlockDAG) EvictionState() *eviction.State {
	return b.evictionState
}

// Attach is used to attach new Blocks to the BlockDAG. It is the main function of the BlockDAG that triggers Events.
func (b *BlockDAG) Attach(data *model.Block) (block *blockdag.Block, wasAttached bool, err error) {
	if block, wasAttached, err = b.attach(data); wasAttached {
		b.events.BlockAttached.Trigger(block)

		b.solidifierMutex.RLock()
		defer b.solidifierMutex.RUnlock()

		b.solidifier.Queue(block)
	}

	return
}

// Block retrieves a Block with metadata from the in-memory storage of the BlockDAG.
func (b *BlockDAG) Block(id iotago.BlockID) (block *blockdag.Block, exists bool) {
	b.evictionMutex.RLock()
	defer b.evictionMutex.RUnlock()

	return b.block(id)
}

// SetInvalid marks a Block as invalid.
func (b *BlockDAG) SetInvalid(block *blockdag.Block, reason error) (wasUpdated bool) {
	if wasUpdated = block.SetInvalid(); wasUpdated {
		b.events.BlockInvalid.Trigger(&blockdag.BlockInvalidEvent{
			Block:  block,
			Reason: reason,
		})
	}

	return
}

// SetOrphaned marks a Block as orphaned and propagates it to its future cone.
func (b *BlockDAG) SetOrphaned(block *blockdag.Block) (updated bool) {
	if !block.SetOrphaned(true) {
		return
	}

	b.events.BlockOrphaned.Trigger(block)

	return true
}

func (b *BlockDAG) PromoteFutureBlocksUntil(index iotago.SlotIndex) {
	b.solidifierMutex.RLock()
	defer b.solidifierMutex.RUnlock()
	b.futureBlocksMutex.Lock()
	defer b.futureBlocksMutex.Unlock()

	for i := b.nextIndexToPromote; i <= index; i++ {
		cm, err := b.commitmentFunc(i)
		if err != nil {
			panic(fmt.Sprintf("failed to load commitment for index %d: %s", i, err))
		}
		if storage := b.futureBlocks.Get(i, false); storage != nil {
			if futureBlocks, exists := storage.Get(cm.MustID()); exists {
				_ = futureBlocks.ForEach(func(futureBlock *blockdag.Block) (err error) {
					b.solidifier.Queue(futureBlock)
					return nil
				})
			}
		}
		b.futureBlocks.Evict(i)
	}

	b.nextIndexToPromote = index + 1
}

func (b *BlockDAG) Shutdown() {
	b.workers.Shutdown()
	b.TriggerStopped()
}

// evictSlot is used to evict Blocks from committed slots from the BlockDAG.
func (b *BlockDAG) evictSlot(index iotago.SlotIndex) {
	b.solidifierMutex.Lock()
	defer b.solidifierMutex.Unlock()
	b.solidifier.EvictUntil(index)

	b.evictionMutex.Lock()
	defer b.evictionMutex.Unlock()

	b.memStorage.Evict(index)
}

func (b *BlockDAG) markSolid(block *blockdag.Block) (err error) {
	// Future blocks already have passed these checks, as they are revisited again at a later point in time.
	// It is important to note that this check only passes once for a specific future block, as it is not yet marked as
	// such. The next time the block is added to the causal order and marked as solid (that is, when it got promoted),
	// it will be marked solid straight away, without checking if it is still a future block: that's why this method
	// must be only called at most twice for any block.
	if !block.IsFuture() {
		if err := b.checkParents(block); err != nil {
			return err
		}

		if b.isFutureBlock(block) {
			return
		}
	}

	// It is important to only set the block as solid when it was not "parked" as a future block.
	// Future blocks are queued for solidification again when the slot is committed.
	block.SetSolid()

	b.events.BlockSolid.Trigger(block)

	return nil
}

func (b *BlockDAG) isFutureBlock(block *blockdag.Block) (isFutureBlock bool) {
	b.futureBlocksMutex.RLock()
	defer b.futureBlocksMutex.RUnlock()

	// If we are not able to load the commitment for the block, it means we haven't committed this slot yet.
	if _, err := b.commitmentFunc(block.Block().SlotCommitment.Index); err != nil {
		// We set the block as future block so that we can skip some checks when revisiting it later in markSolid via the solidifier.
		block.SetFuture()

		lo.Return1(b.futureBlocks.Get(block.Block().SlotCommitment.Index, true).GetOrCreate(block.Block().SlotCommitment.MustID(), func() *advancedset.AdvancedSet[*blockdag.Block] {
			return advancedset.New[*blockdag.Block]()
		})).Add(block)
		return true
	}

	return false
}

func (b *BlockDAG) checkParents(block *blockdag.Block) (err error) {
	for _, parentID := range block.Parents() {
		parent, parentExists := b.Block(parentID)
		if !parentExists {
			panic(fmt.Sprintf("parent %s of block %s should exist as block was marked ordered by the solidifier", parentID, block.ID()))
		}

		// check timestamp monotonicity
		if parent.IssuingTime().After(block.IssuingTime()) {
			return errors.Errorf("timestamp monotonicity check failed for parent %s with timestamp %s. block timestamp %s", parent.ID(), parent.IssuingTime(), block.IssuingTime())
		}

		// check commitment monotonicity
		if parent.SlotCommitmentID().Index() > block.SlotCommitmentID().Index() {
			return errors.Errorf("commitment monotonicity check failed for parent %s with commitment index %d. block commitment index %d", parentID, parent.SlotCommitmentID().Index(), block.SlotCommitmentID().Index())
		}
	}

	return nil
}

func (b *BlockDAG) markInvalid(block *blockdag.Block, reason error) {
	b.SetInvalid(block, errors.Wrap(reason, "block marked as invalid in BlockDAG"))
}

// attach tries to attach the given Block to the BlockDAG.
func (b *BlockDAG) attach(data *model.Block) (block *blockdag.Block, wasAttached bool, err error) {
	b.evictionMutex.RLock()
	defer b.evictionMutex.RUnlock()

	if block, wasAttached, err = b.canAttach(data); !wasAttached {
		return
	}

	if block, wasAttached = b.memStorage.Get(data.ID().Index(), true).GetOrCreate(data.ID(), func() *blockdag.Block { return blockdag.NewBlock(data) }); !wasAttached {
		if wasAttached = block.Update(data); !wasAttached {
			return
		}

		b.events.MissingBlockAttached.Trigger(block)
	}

	block.ForEachParent(func(parent model.Parent) {
		b.registerChild(block, parent)
	})

	return
}

// canAttach determines if the Block can be attached (does not exist and addresses a recent slot).
func (b *BlockDAG) canAttach(data *model.Block) (block *blockdag.Block, canAttach bool, err error) {
	if b.evictionState.InEvictedSlot(data.ID()) && !b.evictionState.IsRootBlock(data.ID()) {
		return nil, false, errors.Errorf("block data with %s is too old (issued at: %s)", data.ID(), data.Block().IssuingTime)
	}

	storedBlock, storedBlockExists := b.block(data.ID())
	if storedBlockExists && !storedBlock.IsMissing() {
		return storedBlock, false, nil
	}

	return b.canAttachToParents(storedBlock, data)
}

// canAttachToParents determines if the Block references parents in a non-pruned slot. If a Block is found to violate
// this condition but exists as a missing entry, we mark it as invalid.
func (b *BlockDAG) canAttachToParents(storedBlock *blockdag.Block, data *model.Block) (block *blockdag.Block, canAttach bool, err error) {
	for _, parentID := range data.Parents() {
		if b.evictionState.InEvictedSlot(parentID) && !b.evictionState.IsRootBlock(parentID) {
			if storedBlock != nil {
				b.SetInvalid(storedBlock, errors.Errorf("block with %s references too old parent %s", data.ID(), parentID))
			}

			return storedBlock, false, errors.Errorf("parent %s of block %s is too old", parentID, data.ID())
		}
	}

	return storedBlock, true, nil
}

// registerChild registers the given Block as a child of the parent. It triggers a BlockMissing event if the referenced
// Block does not exist, yet.
func (b *BlockDAG) registerChild(child *blockdag.Block, parent model.Parent) {
	if b.evictionState.IsRootBlock(parent.ID) {
		return
	}

	parentBlock, _ := b.memStorage.Get(parent.ID.Index(), true).GetOrCreate(parent.ID, func() (newBlock *blockdag.Block) {
		newBlock = blockdag.NewMissingBlock(parent.ID)
		b.events.BlockMissing.Trigger(newBlock)

		return newBlock
	})

	parentBlock.AppendChild(child, parent.Type)
}

// block retrieves the Block with given id from the mem-storage.
func (b *BlockDAG) block(id iotago.BlockID) (block *blockdag.Block, exists bool) {
	if rootBlock, isRootBlock := b.rootBlockProvider(id); isRootBlock {
		return rootBlock, true
	}

	storage := b.memStorage.Get(id.Index(), false)
	if storage == nil {
		return nil, false
	}

	return storage.Get(id)
}

// walkFutureCone traverses the future cone of the given Block and calls the given callback for each Block.
func (b *BlockDAG) walkFutureCone(blocks []*blockdag.Block, callback func(currentBlock *blockdag.Block) (nextChildren []*blockdag.Block)) {
	for childWalker := walker.New[*blockdag.Block](false).PushAll(blocks...); childWalker.HasNext(); {
		childWalker.PushAll(callback(childWalker.Next())...)
	}
}

// checkReference checks if the reference between the child and its parent is valid.
func checkReference(child *blockdag.Block, parent *blockdag.Block) (err error) {
	if parent.IsInvalid() {
		return errors.Errorf("parent %s of child %s is marked as invalid", parent.ID(), child.ID())
	}

	return nil
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
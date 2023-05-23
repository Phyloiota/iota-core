package protocol

import (
	"github.com/iotaledger/hive.go/runtime/event"
	"github.com/iotaledger/iota-core/pkg/network/protocols/core"
	"github.com/iotaledger/iota-core/pkg/protocol/chainmanager"
	"github.com/iotaledger/iota-core/pkg/protocol/engine"
	"github.com/iotaledger/iota-core/pkg/protocol/tipmanager"
)

type Events struct {
	Error *event.Event1[error]

	Network      *core.Events
	Engine       *engine.Events
	TipManager   *tipmanager.Events
	ChainManager *chainmanager.Events
	//Mempool      *mempool.Events

	event.Group[Events, *Events]
}

var NewEvents = event.CreateGroupConstructor(func() (newEvents *Events) {
	return &Events{
		Error: event.New1[error](),

		Network:      core.NewEvents(),
		Engine:       engine.NewEvents(),
		TipManager:   tipmanager.NewEvents(),
		ChainManager: chainmanager.NewEvents(),
		//Mempool:      mempool.NewEvents(),
	}
})

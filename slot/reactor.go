package slot

import (
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
)

const (
	SlotChannel = byte(0x10)
)

type Reactor struct {
	p2p.BaseReactor

	config *config.ConsensusConfig
}

func NewReactor(config *config.ConsensusConfig) *Reactor {
	reactor := &Reactor{
		config: config,
	}

	return reactor
}

// InitPeer implements Reactor
func (slotR *Reactor) InitPeer(peer p2p.Peer) p2p.Peer {
	return peer
}

// SetLogger sets the Logger on the reactor and the underlying mempool.
func (slotR *Reactor) SetLogger(l log.Logger) {
	slotR.Logger = l
}

// OnStart implements p2p.BaseReactor.
func (slotR *Reactor) OnStart() error {
	slotR.Logger.Info("Mempool Reactor started.")
	return nil
}

// GetChannels implements Reactor by returning the list of channels for this
// reactor.
func (slotR *Reactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		//{
		//	ID:                  SlotChannel,
		//	Priority:            10,
		//	RecvMessageCapacity: 100, // TODO 根据编码需求设定
		//},
	}
}

// AddPeer implements Reactor.
func (slotR *Reactor) AddPeer(peer p2p.Peer) {
}

// RemovePeer implements Reactor.
func (slotR *Reactor) RemovePeer(peer p2p.Peer, reason interface{}) {
}

// Receive implements Reactor.
// It adds any received transactions to the mempool.
func (slotR *Reactor) Receive(chID byte, src p2p.Peer, msgBytes []byte) {

}

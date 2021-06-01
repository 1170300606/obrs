package consensus

import (
	"fmt"
	"github.com/tendermint/tendermint/libs/cmap"
	"github.com/tendermint/tendermint/libs/sync"
	"github.com/tendermint/tendermint/p2p"
	"math/rand"
	"time"
)

const (
	TestChannel        = byte(0x30)
	StateChannel       = byte(0x20)
	DataChannel        = byte(0x21)
	VoteChannel        = byte(0x22)
	VoteSetBitsChannel = byte(0x23)

	maxMsgSize = 1048576 // 1MB; NOTE/TODO: keep in sync with types.PartSet sizes.

	blocksToContributeToBecomeGoodPeer = 10000
	votesToContributeToBecomeGoodPeer  = 10000
)

func init() {
	rand.Seed(time.Now().Unix())
}

// ------ Message ------
type Message interface {
	ValidateBasic() error
}

// ------- Reactor ------
type Reactor struct {
	p2p.BaseReactor

	mtx sync.RWMutex

	peers *cmap.CMap

	quit chan struct{}

	seed int64
	id   p2p.ID

	forward *cmap.CMap

	consensus *ConsensusState
}

func (conR *Reactor) SetId(id p2p.ID) {
	conR.id = id
}

type ReactorOption func(*Reactor)

func NewReactor(options ...ReactorOption) *Reactor {
	conR := &Reactor{
		quit:    make(chan struct{}, 1),
		peers:   cmap.NewCMap(),
		seed:    rand.Int63(),
		forward: cmap.NewCMap(),
	}
	conR.BaseReactor = *p2p.NewBaseReactor("Consensus", conR)

	for _, option := range options {
		option(conR)
	}

	return conR
}

func (conR *Reactor) OnStart() error {
	conR.Logger.Info("Consensus Reactor started.")
	go func() {
		//  fake consensus
		d := 10 * time.Second
		timer := time.NewTimer(d)
	LOOP:
		for {
			select {
			case <-conR.quit:
				timer.Stop()
				break LOOP
			case <-timer.C:
				conR.Logger.Info(fmt.Sprintf("%s=%v", conR.id, conR.seed))

				msgBytes := []byte(fmt.Sprintf("%s=%s", conR.id, conR.seed))

				go func(seed int64) {
					resp := <-conR.Switch.Broadcast(TestChannel, msgBytes)
					conR.Logger.Info(fmt.Sprintf("%v的广播结果: %v", seed, resp))
				}(conR.seed)
				conR.seed += 1

				timer.Reset(d)
			}
		}
	}()
	return nil
}

func (conR *Reactor) OnStop() {
	conR.quit <- struct{}{}
}

func (conR *Reactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		{
			ID:                 TestChannel,
			Priority:           10,
			SendQueueCapacity:  100,
			RecvBufferCapacity: maxMsgSize,
		},
	}
}

func (conR *Reactor) InitPeer(peer p2p.Peer) p2p.Peer {
	conR.Logger.With("reactor", "Consensus").Info("new peer come, ", peer.ID())
	return peer
}

func (conR *Reactor) AddPeer(peer p2p.Peer) {
	conR.Logger.Info("addpeer res", peer.Send(TestChannel, []byte("consensus")))

	conR.peers.Set(p2p.IDAddressString(peer.ID(), ""), peer)
}

func (conR *Reactor) RemovePeer(peer p2p.Peer, reason interface{}) {
	return
}

func (conR *Reactor) Receive(chID byte, src p2p.Peer, msgBytes []byte) {
	if !conR.IsRunning() {
		conR.Logger.Debug("Receive", "src", src, "chID", chID, "bytes", msgBytes)
	}

	msg, err := decode(msgBytes)
	if err != nil {
		conR.Logger.Error("Peer Send us invalid msg", "src", src, "msg", msg, "err", err)
		return
	}
	switch chID {
	case TestChannel:
		conR.Logger.Info(fmt.Sprintf("Receive msg from %s, msg=%s", src.ID(), msg))
		if conR.forward.Has(msg) {
			// non-gossip
			break
		}
		conR.Switch.Broadcast(TestChannel, msgBytes)
		conR.forward.Set(msg, struct{}{})

	default:
		conR.Logger.Error(fmt.Sprintf("Unknown chID %X", chID))
	}
}

// --------------------------
func decode(msgBytes []byte) (string, error) {
	return string(msgBytes), nil
}

package consensus

import (
	"chainbft_demo/types"
	"fmt"
	"github.com/tendermint/tendermint/libs/cmap"
	"github.com/tendermint/tendermint/libs/events"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/libs/sync"
	"github.com/tendermint/tendermint/p2p"
	"math/rand"
	"time"
)

const (
	TestChannel        = byte(0x30)
	StateChannel       = byte(0x20)
	ProposalChannel    = byte(0x21)
	VoteChannel        = byte(0x22)
	VoteSetBitsChannel = byte(0x23)

	maxMsgSize = 1048576 // 1MB; NOTE/TODO: keep in sync with types.PartSet sizes.

	blocksToContributeToBecomeGoodPeer = 10000
	votesToContributeToBecomeGoodPeer  = 10000
)

func init() {
	rand.Seed(time.Now().Unix())
}

// ------ Event ------
// reactor监听的consensus广播事件
const (
	EventNewProposal = "NewProposal"
	EventNewVote     = "NewVote"
)

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
	LOOP:
		for {
			select {
			case <-conR.quit:
				break LOOP
				conR.Logger.Info(fmt.Sprintf("%s=%v", conR.id, conR.seed))

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
	// 各自解析数据
	switch chID {
	case VoteChannel:
		// 收到新的投票
		var vote types.Vote

		if err := tmjson.Unmarshal(msgBytes, &vote); err != nil {
			conR.consensus.Logger.Error("try to unmarshal vote failed", "err", err, "msgBytes", msgBytes)
			break
		}

		conR.Logger.Debug(fmt.Sprintf("Receive vote from #{%v}", src.ID()), "vote", vote)
		conR.consensus.peerMsgQueue <- msgInfo{
			Msg:    &VoteMessage{Vote: &vote},
			PeerID: src.ID(),
		}

	case ProposalChannel:
		var proposal types.Proposal

		if err := tmjson.Unmarshal(msgBytes, &proposal); err != nil {
			conR.consensus.Logger.Error("try to unmarshal proposal failed", "err", err, "msgBytes", msgBytes)
			break
		}

		conR.Logger.Debug(fmt.Sprintf("Receive proposal from #{%v}", src.ID()), "proposal", proposal)
		conR.consensus.peerMsgQueue <- msgInfo{
			Msg:    &ProposalMessage{Proposal: &proposal},
			PeerID: src.ID(),
		}

	default:
		conR.Logger.Error(fmt.Sprintf("Unknown chID %X", chID))
	}
}

// subscribeToBroadcastEvents订阅consensus需要广播的消息
func (conR *Reactor) subscribeToBroadcastEvents() {
	const scriber = "consensus-reactor"

	// 监听提案广播事件 - 当consensus成功setProposal以后才会触发事件
	conR.consensus.eventSwitch.AddListenerForEvent(scriber, EventNewProposal, func(data events.EventData) {
		// consensus已经验证过提案的合法性，这里只要简单的广播即可
		// 退一步即使是恶意提案，接收者还需要判断
		conR.broadcastProposal(data.(*types.Proposal))
	})

	// 监听提案投票事件 - 当consensus成功addVote以后才会触发事件
	conR.consensus.eventSwitch.AddListenerForEvent(scriber, EventNewProposal, func(data events.EventData) {
		// consensus已经验证过提案的合法性，这里只要简单的广播即可
		// 退一步即使是恶意提案，接收者还需要判断
		conR.broadcastVote(data.(*types.Vote))
	})
}

func (conR *Reactor) broadcastProposal(proposal *types.Proposal) {
	pBytes, err := tmjson.Marshal(proposal)
	if err != nil {
		conR.Logger.Error("Marshal Proposal failed.", "err", err)
		conR.Logger.Debug("Marshal Proposal failed.", "proposal", proposal)
	}
	conR.Logger.Debug("ready to broadcast Proposal ", "proposal", proposal)
	conR.Switch.Broadcast(ProposalChannel, pBytes)
}

func (conR *Reactor) broadcastVote(vote *types.Vote) {
	vBytes, err := tmjson.Marshal(vote)
	if err != nil {
		conR.Logger.Error("Marshal Vote failed.", "err", err)
		conR.Logger.Debug("Marshal Vote failed.", "vote", vote)
	}
	conR.Logger.Debug("ready to broadcast Vote", "vote", vote)
	conR.Switch.Broadcast(VoteChannel, vBytes)
}

// --------------------------
func decode(msgBytes []byte) (string, error) {
	return string(msgBytes), nil
}

type ProposalMessage struct {
	Proposal *types.Proposal
}

func (msg *ProposalMessage) ValidateBasic() error {
	return msg.Proposal.ValidteBasic()
}

func (msg *ProposalMessage) String() string {
	return fmt.Sprintf("[Proposal %v]", msg.Proposal)
}

type VoteMessage struct {
	Vote *types.Vote
}

func (msg *VoteMessage) ValidateBasic() error {
	// TODO 验证
	return nil
}

func (msg *VoteMessage) String() string {
	return fmt.Sprintf("[Vote %v]", msg.Vote)
}

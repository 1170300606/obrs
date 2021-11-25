package consensus

import (
	"chainbft_demo/types"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/tendermint/tendermint/libs/cmap"
	"github.com/tendermint/tendermint/libs/events"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/sync"
	"github.com/tendermint/tendermint/p2p"
	tmtime "github.com/tendermint/tendermint/types/time"
	"math"
	"math/rand"
	"time"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	StateChannel       = byte(0x20)
	ProposalChannel    = byte(0x21)
	VoteChannel        = byte(0x22)
	VoteSetBitsChannel = byte(0x23)

	maxMsgSize = 10485760 // 10MB; NOTE/TODO: keep in sync with types.PartSet sizes.

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

	consensus *ConsensusState
}

type ReactorOption func(*Reactor)

func NewReactor(conS *ConsensusState, options ...ReactorOption) *Reactor {
	conR := &Reactor{
		quit:      make(chan struct{}, 1),
		peers:     cmap.NewCMap(),
		consensus: conS,
	}
	conR.BaseReactor = *p2p.NewBaseReactor("Consensus", conR)

	for _, option := range options {
		option(conR)
	}

	return conR
}

func (conR *Reactor) SetLogger(l log.Logger) {
	conR.Logger = l
	conR.consensus.SetLogger(l)
}

func (conR *Reactor) OnStart() error {
	genblock := conR.consensus.state.GenesisBlock()
	if genblock.ProposalTime.After(time.Now()) {
		// 如果当前时间早于定义的时间
		conR.Logger.Info("wait genesis time", "time", genblock.ProposalTime, "sleep", genblock.ProposalTime.Sub(time.Now()))
		time.Sleep(genblock.ProposalTime.Sub(time.Now()))
	}

	conR.Logger.Info("Consensus Reactor started.")

	conR.subscribeToBroadcastEvents()
	conR.consensus.slotClock.OnStart()

	conR.consensus.OnStart()

	// 设置初始化时间，离创世区块最近的第1个完整slot的起始时间
	tmnow := tmtime.Now()
	proposalTime := conR.consensus.state.GenesisBlock().ProposalTime
	diffs := float64(tmnow.Sub(proposalTime).Milliseconds()) / 1000.0

	diffSlot := int(math.Ceil(diffs/SlotTimeOut.Seconds())) + Slotdiffs

	// 计算理论上的启动时间 = 离创世区块时间最近的一个slot的起始时间
	startTime := proposalTime.Add(time.Duration(diffSlot*int(SlotTimeOut.Seconds())) * time.Second)

	conR.Logger.Info("timetime", "proposal", proposalTime, "now", tmnow, "idea", startTime)
	conR.Logger.Info("wait start time", "sleep", startTime.Sub(tmnow))
	conR.consensus.slotClock.ResetClock(startTime.Sub(tmnow)) // slot 间隔
	return nil
}

func (conR *Reactor) OnStop() {
	conR.quit <- struct{}{}
}

func (conR *Reactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		{
			ID:                 ProposalChannel,
			Priority:           10,
			SendQueueCapacity:  100,
			RecvBufferCapacity: maxMsgSize,
		},
		{
			ID:                 VoteChannel,
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

		if err := json.Unmarshal(msgBytes, &vote); err != nil {
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

		if err := json.Unmarshal(msgBytes, &proposal); err != nil {
			conR.consensus.Logger.Error("try to unmarshal proposal failed", "err", err)
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
	conR.consensus.eventSwitch.AddListenerForEvent(scriber, EventNewVote, func(data events.EventData) {
		// consensus已经验证过提案的合法性，这里只要简单的广播即可
		// 退一步即使是恶意提案，接收者还需要判断
		conR.broadcastVote(data.(*types.Vote))
	})
}

func (conR *Reactor) broadcastProposal(proposal *types.Proposal) {
	pBytes, err := json.Marshal(proposal)
	if err != nil {
		conR.Logger.Error("Marshal Proposal failed.", "err", err)
		conR.Logger.Debug("Marshal Proposal failed.", "proposal", proposal)
	}
	conR.Logger.Debug("ready to broadcast Proposal ", "proposal", proposal)
	conR.Switch.Broadcast(ProposalChannel, pBytes)
}

func (conR *Reactor) broadcastVote(vote *types.Vote) {
	conR.Logger.Debug("prepare to send vote")
	vBytes, err := json.Marshal(vote)
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

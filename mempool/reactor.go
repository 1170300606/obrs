package mempool

import (
	"chainbft_demo/types"
	"fmt"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/clist"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
	"math"
	"sync"
	"time"
)

const (
	MempoolChannel = byte(0x20)

	peerCatchupSleepIntervalMS = 100 // If peer is behind, sleep this amount

	// UnknownPeerID is the peer ID to use when running CheckTx when there is
	// no peer (e.g. RPC)
	UnknownPeerID uint16 = 0

	maxActiveIDs = math.MaxUint16
)

type Reactor struct {
	p2p.BaseReactor

	mtx sync.Mutex

	config config.MempoolConfig

	mempool *ListMempool
	ids     *mempoolIDs
}

type ReactorOption func(*Reactor)

type mempoolIDs struct {
	mtx       sync.RWMutex
	peerMap   map[p2p.ID]uint16 // map from p2p.ID to mempoolIDs
	nextID    uint16            // nextID指向最后一个可用ID+1的值，但该值不一定可用
	activeIDs map[uint16]struct{}
}

// ReserveForPeer 为peer节点附带一个唯一id
func (ids *mempoolIDs) ReserveForPeer(peer p2p.Peer) {
	ids.mtx.Lock()
	defer ids.mtx.Unlock()

	curID := ids.nextPeerID()
	ids.peerMap[peer.ID()] = curID
	ids.activeIDs[curID] = struct{}{}
}

// nextPeerID 返回下一个可用的id
// 由caller负责lock/unlock.
func (ids *mempoolIDs) nextPeerID() uint16 {
	if len(ids.activeIDs) == maxActiveIDs {
		panic(fmt.Sprintf("node has maximum %d active IDs and wanted to get one more", maxActiveIDs))
	}

	_, idExists := ids.activeIDs[ids.nextID]
	for idExists {
		ids.nextID++
		_, idExists = ids.activeIDs[ids.nextID]
	}
	curID := ids.nextID
	ids.nextID++
	return curID
}

// Reclaim 释放peer对应的id.
func (ids *mempoolIDs) Reclaim(peer p2p.Peer) {
	ids.mtx.Lock()
	defer ids.mtx.Unlock()

	removedID, ok := ids.peerMap[peer.ID()]
	if ok {
		delete(ids.activeIDs, removedID)
		delete(ids.peerMap, peer.ID())
	}
}

// GetForPeer 返回peer的id.
func (ids *mempoolIDs) GetForPeer(peer p2p.Peer) uint16 {
	ids.mtx.RLock()
	defer ids.mtx.RUnlock()

	return ids.peerMap[peer.ID()]
}

func newMempoolIDs() *mempoolIDs {
	return &mempoolIDs{
		peerMap:   make(map[p2p.ID]uint16),
		activeIDs: map[uint16]struct{}{0: {}},
		nextID:    1, // 为unknownPeerID保留0，节点之间广播使用unKnownPeerId
	}
}

func NewReactor(mempool *ListMempool, options ...ReactorOption) *Reactor {
	reactor := &Reactor{
		mempool: mempool,
		ids:     newMempoolIDs(),
	}
	reactor.BaseReactor = *p2p.NewBaseReactor("Mempool", reactor)

	return reactor
}

// InitPeer implements Reactor
// 为peer生成一个唯一的id
// ConsensusReactor要负责在peer注册PeerState
func (memR *Reactor) InitPeer(peer p2p.Peer) p2p.Peer {
	memR.ids.ReserveForPeer(peer)
	return peer
}

// SetLogger sets the Logger on the reactor and the underlying mempool.
func (memR *Reactor) SetLogger(l log.Logger) {
	memR.Logger = l
	memR.mempool.SetLogger(l)
}

// OnStart implements p2p.BaseReactor.
func (memR *Reactor) OnStart() error {
	memR.Logger.Info("Mempool Reactor started.")
	return nil
}

// GetChannels implements Reactor by returning the list of channels for this
// reactor.
func (memR *Reactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		{
			ID:                  MempoolChannel,
			Priority:            5,
			RecvMessageCapacity: 1024 * 1024, // TODO 根据编码需求设定
		},
	}
}

// AddPeer implements Reactor.
// 启动broadcast routine在节点之间广播tx
func (memR *Reactor) AddPeer(peer p2p.Peer) {
	go memR.broadcastTxRoutine(peer)
}

// RemovePeer implements Reactor.
func (memR *Reactor) RemovePeer(peer p2p.Peer, reason interface{}) {
	memR.ids.Reclaim(peer)
	// broadcast routine checks if peer is gone and returns
}

// Receive implements Reactor.
// It adds any received transactions to the mempool.
func (memR *Reactor) Receive(chID byte, src p2p.Peer, msgBytes []byte) {
	//msg, err := memR.decodeMsg(msgBytes)
	//if err != nil {
	//	memR.Logger.Error("Error decoding message", "src", src, "chId", chID, "err", err)
	//	memR.Switch.StopPeerForError(src, err)
	//	return
	//}
	memR.Logger.Info("Receive Tx", "src", src, "chId", chID, "msg", msgBytes)

	txInfo := TxInfo{SenderID: memR.ids.GetForPeer(src)}
	if src != nil {
		txInfo.SenderP2PID = src.ID()
	}
	//for _, tx := range msg.Txs {
	err := memR.mempool.CheckTx(msgBytes, txInfo)
	if err != nil {
		memR.Logger.Info("Could not check tx", "tx", txID(msgBytes), "err", err)
	}
	//}
}

// --------------------------------
func (memR *Reactor) broadcastTxRoutine(peer p2p.Peer) {
	// TODO 在节点之间广播
	// 不要将交易原路返回
	// 监听mempool的txs的chan

	peerID := memR.ids.GetForPeer(peer)
	var next *clist.CElement

	for {
		if !memR.IsRunning() || !peer.IsRunning() {
			memR.Logger.Error(fmt.Sprintf("peer {} didn't start", peerID))
			return
		}

		// 意义不明白
		if next == nil {
			select {
			case <-memR.mempool.TxsWaitChan():
				if next = memR.mempool.TxsFront(); next == nil {
					continue
				}
			case <-peer.Quit():
				return
			case <-memR.Quit():
				return
			}
		}

		memTx := next.Value.(*mempoolTx)

		if _, ok := memTx.senders.Load(peerID); !ok {
			// 没有从该节点收到这条tx，准备向该节点发送tx
			memR.Logger.Info("prepare to send tx to other peer", "tx", memTx.tx, "destination", peer)
			if success := peer.Send(MempoolChannel, memTx.tx); !success {
				// 如果发送不成功，间隔peerCatchupSleepIntervalMS后再看是否需要发送
				memR.Logger.Error("send tx failed")
				time.Sleep(peerCatchupSleepIntervalMS * time.Millisecond)
				continue
			}
		}

		select {
		// 当next有下一个元素时，它的nextWaitch关闭，<-会读出来nil，流程继续
		// 如果没有下一个元素，则会在这里block
		case <-next.NextWaitChan():
			next = next.Next()
		case <-peer.Quit():
			return
		case <-memR.Quit():
			return
		}
	}
}

// 将byte数组解码会tx的数组格式
// bytes data => []types.Tx
func (memR *Reactor) decodeMsg([]byte) (TxsMessage, error) {
	// TODO 确定编码协议 proto？
	return TxsMessage{}, nil
}

// ---------------------------------
type TxsMessage struct {
	Txs []types.Tx
}

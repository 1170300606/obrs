package mempool

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/go-kit/kit/log/term"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chainbft_demo/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/p2p/mock"
)

const (
	numTxs       = 1000
	timeout      = 120 * time.Second // ridiculously high because CircleCI is slow
	peerStateKey = "ConsensusReactor.peerState"
)

type peerState struct {
	height int64
}

func (ps peerState) GetHeight() int64 {
	return ps.height
}

// 测试节点之间的mempool同步
// 测试向节点a的mempool加入一组交易，节点b也能收到这些交易
func TestReactorBroadcastTxsMessage(t *testing.T) {
	config := cfg.TestConfig()

	const N = 2
	reactors := makeAndConnectReactors(config, N)
	defer func() {
		for _, r := range reactors {
			if err := r.Stop(); err != nil {
				assert.NoError(t, err)
			}
		}
	}()
	for _, r := range reactors {
		for _, peer := range r.Switch.Peers().List() {
			peer.Set(peerStateKey, peerState{1})
		}
	}

	txs := checkTxs(t, reactors[0].mempool, numTxs, UnknownPeerID)

	// 期待reactor[1]按序收到txs的交易
	waitForTxsOnReactors(t, txs, reactors)
}

// 多Reactor的并发测试
// 同时向2个Reactor发送交易，reactor a在同步reactor b交易的同时模拟update提交自己增加的交易
func TestReactorConcurrency(t *testing.T) {
	config := cfg.TestConfig()
	const N = 2
	reactors := makeAndConnectReactors(config, N)
	defer func() {
		for _, r := range reactors {
			if err := r.Stop(); err != nil {
				assert.NoError(t, err)
			}
		}
	}()
	for _, r := range reactors {
		for _, peer := range r.Switch.Peers().List() {
			peer.Set(peerStateKey, peerState{1})
		}
	}
	var wg sync.WaitGroup

	const numTxs = 5

	for i := 0; i < 1000; i++ {
		wg.Add(2)

		// 1. submit a bunch of txs
		// 2. update the whole mempool
		txs := checkTxs(t, reactors[0].mempool, numTxs, UnknownPeerID)
		go func() {
			defer wg.Done()

			reactors[0].mempool.Lock()
			defer reactors[0].mempool.Unlock()

			err := reactors[0].mempool.Update(1, txs)
			assert.NoError(t, err)
		}()

		// 1. submit a bunch of txs
		// 2. update none
		_ = checkTxs(t, reactors[1].mempool, numTxs, UnknownPeerID)
		go func() {
			defer wg.Done()

			reactors[1].mempool.Lock()
			defer reactors[1].mempool.Unlock()
			err := reactors[1].mempool.Update(1, []types.Tx{})
			assert.NoError(t, err)
		}()

		// 1. flush the mempool
		reactors[1].mempool.Flush()

	}

	wg.Wait()
}

// 模拟reactor b向reactor a发送交易，但是reactor b不会再次收到这些交易
func TestReactorNoBroadcastToSender(t *testing.T) {
	config := cfg.TestConfig()
	const N = 2
	reactors := makeAndConnectReactors(config, N)
	defer func() {
		for _, r := range reactors {
			if err := r.Stop(); err != nil {
				assert.NoError(t, err)
			}
		}
	}()
	for _, r := range reactors {
		for _, peer := range r.Switch.Peers().List() {
			peer.Set(peerStateKey, peerState{1})
		}
	}

	const peerID = 1
	checkTxs(t, reactors[0].mempool, numTxs, peerID)
	ensureNoTxs(t, reactors[peerID], 100*time.Millisecond)
}

// 测试Reactor发送极限大小的tx
func TestReactor_MaxTxBytes(t *testing.T) {
	config := cfg.TestConfig()

	const N = 2
	reactors := makeAndConnectReactors(config, N)
	defer func() {
		for _, r := range reactors {
			if err := r.Stop(); err != nil {
				assert.NoError(t, err)
			}
		}
	}()
	for _, r := range reactors {
		for _, peer := range r.Switch.Peers().List() {
			peer.Set(peerStateKey, peerState{1})
		}
	}

	// tx1的大小为规定的最大的大小
	// 确保reactor 2要能收到该交易
	tx1 := tmrand.Bytes(config.Mempool.MaxTxBytes)
	err := reactors[0].mempool.CheckTx(types.NormalTx(tx1), TxInfo{SenderID: UnknownPeerID})
	require.NoError(t, err)
	waitForTxsOnReactors(t, []types.Tx{types.NormalTx(tx1)}, reactors)

	reactors[0].mempool.Flush()
	reactors[1].mempool.Flush()

	// tx2大小超过规定大小
	// 确保reactor a不接受该交易
	tx2 := tmrand.Bytes(config.Mempool.MaxTxBytes + 1)
	err = reactors[0].mempool.CheckTx(types.NormalTx(tx2), TxInfo{SenderID: UnknownPeerID})
	require.Error(t, err)
}

// 测试当有节点退出时不会出现goroutine泄漏
func TestBroadcastTxForPeerStopsWhenPeerStops(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	config := cfg.TestConfig()
	const N = 2
	reactors := makeAndConnectReactors(config, N)
	defer func() {
		for _, r := range reactors {
			if err := r.Stop(); err != nil {
				assert.NoError(t, err)
			}
		}
	}()

	// stop peer
	sw := reactors[1].Switch
	sw.StopPeerForError(sw.Peers().List()[0], errors.New("some reason"))

	// check that we are not leaking any go-routines
	// i.e. broadcastTxRoutine finishes when peer is stopped
	leaktest.CheckTimeout(t, 10*time.Second)()
}

// 测试节点mempool Reactor能否正常退出
func TestBroadcastTxForPeerStopsWhenReactorStops(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	config := cfg.TestConfig()
	const N = 2
	reactors := makeAndConnectReactors(config, N)

	// stop reactors
	for _, r := range reactors {
		if err := r.Stop(); err != nil {
			assert.NoError(t, err)
		}
	}

	// check that we are not leaking any go-routines
	// i.e. broadcastTxRoutine finishes when reactor is stopped
	leaktest.CheckTimeout(t, 10*time.Second)()
}

// 测试MempoolIds能否正常分配id回收id
func TestMempoolIDsBasic(t *testing.T) {
	ids := newMempoolIDs()

	peer := mock.NewPeer(net.IP{127, 0, 0, 1})

	ids.ReserveForPeer(peer)
	assert.EqualValues(t, 1, ids.GetForPeer(peer))
	ids.Reclaim(peer)

	ids.ReserveForPeer(peer)
	assert.EqualValues(t, 2, ids.GetForPeer(peer))
	ids.Reclaim(peer)
}

// 测试MempoolIds已经分配完id后，再次请求分配能否panic
func TestMempoolIDsPanicsIfNodeRequestsOvermaxActiveIDs(t *testing.T) {
	if testing.Short() {
		return
	}

	// 0 is already reserved for UnknownPeerID
	ids := newMempoolIDs()

	for i := 0; i < maxActiveIDs-1; i++ {
		peer := mock.NewPeer(net.IP{127, 0, 0, 1})
		ids.ReserveForPeer(peer)
	}

	assert.Panics(t, func() {
		peer := mock.NewPeer(net.IP{127, 0, 0, 1})
		ids.ReserveForPeer(peer)
	})
}

func TestDontExhaustMaxActiveIDs(t *testing.T) {
	config := cfg.TestConfig()
	const N = 1
	reactors := makeAndConnectReactors(config, N)
	defer func() {
		for _, r := range reactors {
			if err := r.Stop(); err != nil {
				assert.NoError(t, err)
			}
		}
	}()
	reactor := reactors[0]

	for i := 0; i < maxActiveIDs+1; i++ {
		peer := mock.NewPeer(nil)
		reactor.Receive(MempoolChannel, peer, []byte{0x1, 0x2, 0x3})
		reactor.AddPeer(peer)
	}
}

// mempoolLogger is a TestingLogger which uses a different
// color for each validator ("validator" key must exist).
func mempoolLogger() log.Logger {
	return log.TestingLoggerWithColorFn(func(keyvals ...interface{}) term.FgBgColor {
		for i := 0; i < len(keyvals)-1; i += 2 {
			if keyvals[i] == "validator" {
				return term.FgBgColor{Fg: term.Color(uint8(keyvals[i+1].(int) + 1))}
			}
		}
		return term.FgBgColor{}
	})
}

// connect N mempool reactors through N switches
func makeAndConnectReactors(config *cfg.Config, n int) []*Reactor {
	reactors := make([]*Reactor, n)
	logger := mempoolLogger()
	for i := 0; i < n; i++ {
		mempool, cleanup := newMempool()
		defer cleanup()

		reactors[i] = NewReactor(config.Mempool, mempool) // so we dont start the consensus states
		reactors[i].SetLogger(logger.With("validator", i))
	}

	p2p.MakeConnectedSwitches(config.P2P, n, func(i int, s *p2p.Switch) *p2p.Switch {
		s.AddReactor("MEMPOOL", reactors[i])
		return s
	}, p2p.Connect2Switches)
	return reactors
}

func waitForTxsOnReactors(t *testing.T, txs types.Txs, reactors []*Reactor) {
	// wait for the txs in all mempools
	wg := new(sync.WaitGroup)
	for i, reactor := range reactors {
		wg.Add(1)
		go func(r *Reactor, reactorIndex int) {
			defer wg.Done()
			waitForTxsOnReactor(t, txs, r, reactorIndex)
		}(reactor, i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	timer := time.After(timeout)
	select {
	case <-timer:
		t.Fatal("Timed out waiting for txs")
	case <-done:
	}
}

func waitForTxsOnReactor(t *testing.T, txs types.Txs, reactor *Reactor, reactorIndex int) {
	mempool := reactor.mempool
	for mempool.Size() < len(txs) {
		time.Sleep(time.Millisecond * 100)
	}

	reapedTxs := mempool.ReapMaxTxs(len(txs))
	assert.Equal(t, len(txs), len(reapedTxs))
	for i, tx := range txs {
		assert.Equalf(t, tx, reapedTxs[i],
			"txs at index %d on reactor %d don't match: %v vs %v", i, reactorIndex, tx, reapedTxs[i])
	}
}

// ensure no txs on reactor after some timeout
func ensureNoTxs(t *testing.T, reactor *Reactor, timeout time.Duration) {
	time.Sleep(timeout) // wait for the txs in all mempools
	assert.Zero(t, reactor.mempool.Size())
}

package consensus

import (
	mempool "chainbft_demo/mempool"
	bkstate "chainbft_demo/state"
	"chainbft_demo/types"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"os"
	"testing"
	"time"
)

type cleanup func()

func newConsensusState() (*ConsensusState, cleanup) {
	conR := NewReactor()

	cs, clean := newConsensusStateWithConfig(config.ResetTestRoot("consensus_test"))
	SetReactor(conR)
	return cs, clean
}

func newConsensusStateWithConfig(config *config.Config) (*ConsensusState, cleanup) {
	chainID := "CONSENSUS_TEST"
	geneBlock := types.MakeGenesisBlock(chainID, []byte("signature"))

	state := bkstate.MakeGenesisState(chainID, types.LtimeZero, geneBlock)

	mempool := mempool.NewListMempool(config.Mempool)
	blockExec := bkstate.NewBlockExecutor(mempool)
	cs := NewConsensusState(config.Consensus, types.LtimeZero, blockExec, nil, state)

	cs.SetLogger(log.NewFilter(log.TestingLogger(), log.AllowDebug()))
	return cs, func() {
		os.RemoveAll(config.RootDir)
	}
}

func TestNewDefaultConsensusState(t *testing.T) {
	cs, clean := newConsensusState()
	defer clean()
	cs.OnStart()
	cs.slotClock.OnStart()
	cs.enterNewSlot(types.LtimeZero) //手动触发
	time.Sleep(20 * time.Second)
	cs.OnStop()
}

func TestEnterAlppyInWrongWay(t *testing.T) {
	cs, clean := newConsensusState()
	defer clean()
	cs.OnStart()
	cs.slotClock.OnStart()

	assert.Panics(t, func() {
		cs.enterApply()
	}) //手动触发

	cs.OnStop()
}

func TestEnterProposeInWrongWay(t *testing.T) {
	cs, clean := newConsensusState()
	defer clean()
	cs.OnStart()
	cs.slotClock.OnStart()

	assert.Panics(t, func() {
		cs.enterPropose()
	}) //手动触发

	cs.OnStop()
}

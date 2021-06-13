package consensus

import (
	cstypes "chainbft_demo/consensus/types"
	mempl "chainbft_demo/mempool"
	bkstate "chainbft_demo/state"
	"chainbft_demo/types"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/privval"
	"os"
	"testing"
	"time"
)

type cleanup func()
type memplFunc func(mempool mempl.Mempool)

func generateGenesis(chainID string) (*types.Block, bkstate.State) {
	genblock := types.MakeGenesisBlock("state_test", []byte("signature"))
	genstate := bkstate.MakeGenesisState("state_test", types.LtimeZero, genblock)

	return genblock, genstate
}

func newBlockExecutorWithConfig(config *config.Config, mempool mempl.Mempool, logger log.Logger) (bkstate.BlockExecutor, cleanup) {
	blockexec := bkstate.NewBlockExecutor(mempool)
	blockexec.SetLogger(logger)

	return blockexec, func() {
		os.RemoveAll(config.RootDir)
	}
}
func newConsensusState(memplfunc ...memplFunc) (*ConsensusState, cleanup) {
	logger := log.NewFilter(log.TestingLogger(), log.AllowDebug())

	conR := NewReactor()
	conR.SetLogger(logger)

	cs, clean := newConsensusStateWithConfig(config.ResetTestRoot("consensus_test"), logger, memplfunc...)
	SetReactor(conR)(cs)
	return cs, clean
}

func newConsensusStateWithConfig(config *config.Config, logger log.Logger, memplfunc ...memplFunc) (*ConsensusState, cleanup) {
	chainID := "CONSENSUS_TEST"
	geneBlock := types.MakeGenesisBlock(chainID, []byte("signature"))

	state := bkstate.MakeGenesisState(chainID, types.LtimeZero, geneBlock)

	mempool := mempl.NewListMempool(config.Mempool)
	mempool.SetLogger(logger)
	blockExec, _ := newBlockExecutorWithConfig(config, mempool, logger)

	cs := NewConsensusState(config.Consensus, types.LtimeZero, blockExec, nil, state, SetValidtor(privval.GenFilePV("", "")))

	cs.SetLogger(logger)

	for _, f := range memplfunc {
		f(mempool)
	}

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

// TestProposal 测试consensus能否正确的通过blockexec生成新的提案,同时测试下setproposal的正确性
// NOTE: proposal的正确性由blockexec保证
func TestProposal(t *testing.T) {
	txs := []types.Tx{}
	cs, clean := newConsensusState(func(mem mempl.Mempool) {
		// 默认创建10条交易

		for i := 0; i < 10; i++ {
			tx := []byte(fmt.Sprintf("tx=%v", i))
			txs = append(txs, tx)
			err := mem.CheckTx(tx, mempl.TxInfo{
				SenderID: mempl.UnknownPeerID,
			})
			assert.NoError(t, err, "add %vth tx into mempool failed.", i)
		}

	})

	defer clean()
	cs.OnStart()
	cs.slotClock.OnStart()

	// 手动进入proposal step，验证是否可以运行和正确打包
	cs.updateStep(cstypes.RoundStepApply)

	var proposal *types.Proposal

	assert.NotPanics(t, func() {
		proposal = cs.defaultProposal()

		cs.defaultSetProposal(proposal)

	}, "propose failed.")

	assert.Equal(t, proposal, cs.Proposal, "proposal不一致，setProposal失败")
}

// TestApply 测试consensus能否正确通过blockExec执行提交操作，consensus需要保证quorum的正确生成
// NOTE: state更新的正确性由blockexec保证
func TestApplyWithSuspectProposal(t *testing.T) {
	txs := []types.Tx{}
	cs, clean := newConsensusState(func(mem mempl.Mempool) {
		// 默认创建10条交易

		for i := 0; i < 10; i++ {
			tx := []byte(fmt.Sprintf("tx=%v", i))
			txs = append(txs, tx)
			err := mem.CheckTx(tx, mempl.TxInfo{
				SenderID: mempl.UnknownPeerID,
			})
			assert.NoError(t, err, "add %vth tx into mempool failed.", i)
		}

	})

	defer clean()
	cs.OnStart()

	// 手动设置提案
	cs.updateStep(cstypes.RoundStepApply)
	proposal := cs.defaultProposal()
	cs.defaultSetProposal(proposal)

	// 手动enterNewSlot
	cs.updateStep(cstypes.RoundStepSlot)

	prevState := cs.state
	assert.NotPanics(t, func() {
		cs.enterApply()
	})

	assert.Equal(t, prevState, cs.lastState)
	assert.NotEqual(t, prevState, cs.state)
	assert.Equal(t, types.SuspectBlock, proposal.BlockState) //  没有投票，该提案为suspect block
}

// TODO TestApplyWithSuppotQuorum 测试consensus apply动作，设置的提案有足够的supportVote

func TestApplyWithSuppotQuorum(t *testing.T) {
	txs := []types.Tx{}
	cs, clean := newConsensusState(func(mem mempl.Mempool) {
		// 默认创建10条交易

		for i := 0; i < 10; i++ {
			tx := []byte(fmt.Sprintf("tx=%v", i))
			txs = append(txs, tx)
			err := mem.CheckTx(tx, mempl.TxInfo{
				SenderID: mempl.UnknownPeerID,
			})
			assert.NoError(t, err, "add %vth tx into mempool failed.", i)
		}

	})

	defer clean()
	cs.OnStart()

	// 手动设置提案
	cs.updateStep(cstypes.RoundStepApply)
	proposal := cs.defaultProposal()
	cs.defaultSetProposal(proposal)

	// TODO 手动增加投票

	// 手动enterNewSlot
	cs.updateStep(cstypes.RoundStepSlot)

	prevState := cs.state
	assert.NotPanics(t, func() {
		cs.enterApply()
	})

	assert.Equal(t, prevState, cs.lastState)
	assert.NotEqual(t, prevState, cs.state)
	assert.Equal(t, types.PrecommitBlock, proposal.BlockState) //  没有投票，该提案为suspect block
}

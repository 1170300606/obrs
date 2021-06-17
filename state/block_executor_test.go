package state

import (
	mempl "chainbft_demo/mempool"
	"chainbft_demo/types"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"os"
	"testing"
)

type cleanup func()

func newBLockExecutor() (BlockExecutor, cleanup) {
	testconfig := config.ResetTestRoot("state_test")
	return newBlockExecutorWithConfig(testconfig)
}

func newBlockExecutorWithConfig(config *config.Config) (BlockExecutor, cleanup) {
	logger := log.NewFilter(log.TestingLogger(), log.AllowDebug())
	listMempool := mempl.NewListMempool(config.Mempool)
	listMempool.SetLogger(logger)
	blockexec := NewBlockExecutor(listMempool)
	blockexec.SetLogger(logger)

	return blockexec, func() {
		os.RemoveAll(config.RootDir)
	}
}

func generateGenesis(chainID string) (*types.Block, State) {
	genblock := types.MakeGenesisBlock("state_test", []byte("signature"))
	genstate := MakeGenesisState("state_test", types.LtimeZero, genblock, nil, nil, nil)

	return genblock, genstate
}

// TestCreateProposal 测试打包的正确性
func TestCreateProposal(t *testing.T) {
	_, genstate := generateGenesis("state_test")
	testconfig := config.ResetTestRoot("state_test")
	logger := log.NewFilter(log.TestingLogger(), log.AllowDebug())
	listMempool := mempl.NewListMempool(testconfig.Mempool)
	listMempool.SetLogger(logger)
	blockExec := NewBlockExecutor(listMempool)
	blockExec.SetLogger(logger)

	txs := []types.Tx{}
	for i := 0; i < 10; i++ {
		tx := []byte(fmt.Sprintf("tx=%v", i))
		txs = append(txs, tx)
		err := listMempool.CheckTx(tx, mempl.TxInfo{
			SenderID: mempl.UnknownPeerID,
		})
		assert.NoError(t, err, "add %vth tx into mempool failed.", i)
	}

	var proposal *types.Proposal

	assert.NotPanics(t, func() {
		proposal = blockExec.CreateProposal(genstate, types.LTime(1))
	}, "create proposal failed.")

	assert.Equal(t, 10, len(proposal.Txs), "proposal should contain 10 txs, but got %v", len(proposal.Txs))

	// 理论顺序打包
	for i := 0; i < 10; i++ {
		assert.Equal(t, txs[i], proposal.Txs[i], "%vth tx non-equal", i)
	}
}

func TestApplyBlock(t *testing.T) {
	genblock, genstate := generateGenesis("state_test")
	txs := []types.Tx{}

	for i := 0; i < 10; i++ {
		tx := []byte(fmt.Sprintf("tx=%v", i))
		txs = append(txs, tx)

	}
	block := types.MakeBlock(txs)
	block.Fill("state_test", types.LTime(1), types.PrecommitBlock, genblock.BlockHash, genstate.Validators.Hash(), nil)

	blockExec, cleanup := newBLockExecutor()
	defer cleanup()
	var newState State
	var err error
	assert.NotPanics(t, func() {
		newState, err = blockExec.ApplyBlock(genstate, block)
		assert.NoError(t, err, "aplly block failed")
	}, "apply block panic.")

	// 检查blockSet有无更新
	assert.Equal(t, 1, newState.UnCommitBlocks.Size())
	assert.Equal(t, block, newState.UnCommitBlocks.Blocks()[0])

	// 检查blocktree有无更新提案
	queryBlock, err := newState.BlockTree.QueryBlockByHash(block.BlockHash)
	assert.NoError(t, err)
	assert.NotNil(t, block, queryBlock)
	t.Log(newState)
}

package state

import (
	"chainbft_demo/crypto/bls"
	threshold2 "chainbft_demo/crypto/threshold"
	mempl "chainbft_demo/mempool"
	"chainbft_demo/privval"
	"chainbft_demo/types"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"os"
	"testing"
	"time"
)

type cleanup func()

// 生成指定数量的validator，
// 返回相等数量的私钥、验证者集合、公共公钥
func newPrivAndValSet(count int) ([]types.PrivValidator, *types.ValidatorSet, *types.Validator) {
	pub_priv := bls.GenTestPrivKey(int64(100))
	pub_val := types.NewValidator(pub_priv.PubKey())
	poly := threshold2.Master(pub_priv, 3, 1000)

	privs := []types.PrivValidator{}
	vallist := []*types.Validator{}

	for i := 0; i < count; i++ {
		priv, err := poly.GetValue(int64(i + 1))
		if err != nil {
			panic(err)
		}
		prival := privval.NewFilePV(priv, "")

		val := types.NewValidator(priv.PubKey())
		vallist = append(vallist, val)
		privs = append(privs, prival)
	}

	vals := types.NewValidatorSet(vallist)

	return privs, vals, pub_val
}

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
	genblock := types.MakeGenesisBlock("state_test", time.Now())
	_, vals, pub_val := newPrivAndValSet(4)
	_, val := vals.GetByIndex(0)

	genstate := MakeGenesisState("state_test", types.LtimeZero, genblock, val, pub_val, vals)

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
	block.Fill("state_test", types.LTime(1), types.PrecommitBlock, genblock.BlockHash, genstate.Validators.Hash(), genstate.Validators.Hash())
	block.Signature = []byte("singature")
	block.Hash()
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

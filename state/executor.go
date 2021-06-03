package state

import (
	"chainbft_demo/mempool"
	"chainbft_demo/store"
	"chainbft_demo/types"
	"github.com/tendermint/tendermint/libs/log"
)

type BlockExecutor interface {
	// CreateProposal 从mempool按照交易到达的顺序打包交易
	// 需要注意打包的交易不能和pre-commit区块的任意一个交易有冲突
	CreateProposal([]*types.Block) *types.Proposal

	// Apply一个指定的区块，如果提交成功后state发生变化，返回新的state
	ApplyBlock(block *types.Block) (State, error)

	SetLogger(logger log.Logger)
}

func NewBlockExec(mempool mempool.Mempool) BlockExecutor {
	blockexec := &blockExcutor{
		mempool: mempool,
	}

	return blockexec
}

type blockExcutor struct {
	mempool mempool.Mempool

	db store.Store

	logger log.Logger
}

// SetLogger implements BlockExecutor
func (exec *blockExcutor) SetLogger(logger log.Logger) {
	exec.logger = logger
}

// ApplyBlock implements BlockExecutor
func (exec *blockExcutor) ApplyBlock(block *types.Block) (State, error) {
	// TODO apply上一轮slot的proposal，同时会尝试更新pre-commit区块
	return State{}, nil
}

func (exec *blockExcutor) CreateProposal(pendingBlocks []*types.Block) *types.Proposal {

	// reap all tx from mempool
	// 需要变更mempool的接口，reap时候要选择没有冲突的交易
	_ = exec.mempool.ReapMaxTxs(-1)
	return nil
}

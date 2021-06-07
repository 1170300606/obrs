package state

import (
	"chainbft_demo/mempool"
	"chainbft_demo/store"
	"chainbft_demo/types"
	"github.com/tendermint/tendermint/libs/log"
	"time"
)

type BlockExecutor interface {
	// CreateProposal 从mempool按照交易到达的顺序打包交易
	// 需要注意打包的交易不能和pre-commit区块的任意一个交易有冲突
	CreateProposal(State, types.LTime) *types.Proposal

	// Apply一个指定的区块，如果提交成功后state发生变化，返回新的state
	ApplyBlock(state State, block *types.Block) (State, error)

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
// apply上一轮slot的proposal，同时会尝试更新pre-commit区块
func (exec *blockExcutor) ApplyBlock(state State, block *types.Block) (State, error) {
	// 首先验证区块是否合法，不合法直接返回愿状态
	if err := exec.validateBlock(state, block); err != nil {
		return state, ErrInvalidBlock(err)
	}

	// 决定哪些区块可以提交
	toCommitBlock := state.decideCommitBlocks(block)

	// 正式提交交易，在这里可能不止提交一个交易，如果提交成功后还要负责删除mempool中的交易
	state, err := exec.Commit(state, toCommitBlock)
	if err != nil {
		return state, err
	}

	// 更新state的状态
	_ = state.BlockTree.AddBlocks(block.LastBlockHash, block)
	state.LastBlockTime = time.Now()
	state.LastBlockHash = block.BlockHash
	state.LastBlockSlot = block.Slot

	return state, err
}

// 按照list的顺序提交多个区块，是否要保证原子性待定
// 如果提交成功后更新mempool
func (exec *blockExcutor) Commit(state State, toCommitblocks []*types.Block) (State, error) {
	toRemovesTxs := types.Txs{}
	// 假设都能全部提交成功
	for idx, block := range toCommitblocks {
		exec.logger.Debug("commit block", "state", state, "idx", idx)
		for _, tx := range block.Txs {
			// commit tx
			exec.logger.Debug(
				"commit tx",
				"tx", tx)
		}
		// TODO 检查交易执行的状态 如果有一个区块交易执行失败，直接结束本轮提交，是否需要rollback待定
		block.BlockState = types.CommiitedBlock
		//block.ResultHash = newhash
		toRemovesTxs.Append(block.Txs)
	}

	// 提交成功后更新mempool，首先加锁
	exec.mempool.Lock()
	exec.mempool.Update(0, toRemovesTxs)
	exec.mempool.Unlock()

	newState := state.Copy()
	// 更新precommit blocks
	newState.CommitBlocks(toCommitblocks)
	// TODO 更新提交后的merkle root
	newState.LastResultsHash = newState.LastResultsHash

	return newState, nil
}

// 从mempool中打包交易
func (exec *blockExcutor) CreateProposal(state State, nextSlot types.LTime) *types.Proposal {
	// reap all tx from mempool
	// 需要变更mempool的接口，reap时候要选择没有冲突的交易
	txs := exec.mempool.ReapMaxTxs(-1)
	block := types.MakeBlock(state.ChainID, nextSlot, txs)
	return &types.Proposal{
		ChainID: state.ChainID,
		Slot:    nextSlot,
		Block:   block,
	}
}

// 根绝当前的state验证一个区块是否合法
// TODO
func (exec *blockExcutor) validateBlock(state State, block *types.Block) error {
	// 先检验区块基本的信息是否正确
	if err := block.ValidteBasic(); err != nil {
		return err
	}

	// TODO 检验block是否符合state

	return nil
}

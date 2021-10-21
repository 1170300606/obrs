package state

import (
	"bytes"
	mempl "chainbft_demo/mempool"
	"chainbft_demo/store"
	"chainbft_demo/types"
	"errors"
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

func NewBlockExecutor(mempool mempl.Mempool) BlockExecutor {
	blockexec := &blockExcutor{
		mempool: mempool,
	}

	return blockexec
}

type blockExcutor struct {
	mempool mempl.Mempool

	db store.Store

	logger log.Logger
}

// SetLogger implements BlockExecutor
func (exec *blockExcutor) SetLogger(logger log.Logger) {
	exec.logger = logger
}

// ApplyBlock implements BlockExecutor
// apply上一轮slot的proposal，同时会尝试更新pre-commit区块
func (exec *blockExcutor) ApplyBlock(state State, proposal *types.Block) (State, error) {
	// 首先验证区块是否合法，不合法直接返回愿状态
	if err := exec.validateBlock(state, proposal); err != nil {
		return state, ErrInvalidBlock(err)
	}

	// 首先将这轮slot收到的block里面的交易在mempool中变更状态
	exec.mempool.Lock()
	exec.logger.Debug("prepare to locks proposal txs")
	if err := exec.mempool.LockTxs(proposal.Txs); err != nil {
		exec.logger.Error("Lock txs in mempool failed.", "raason", err)
	}
	exec.mempool.Unlock()

	// 根据proposal的投票情况更新blockSet
	state.UnCommitBlocks.AddBlock(proposal)

	// 决定哪些区块可以提交
	toCommitBlock := state.decideCommitBlocks(proposal)
	exec.logger.Debug("prepare to commit blocks", "size", len(toCommitBlock), "toCommitBlocks", toCommitBlock)

	// 正式提交交易，在这里可能不止提交一个交易，如果提交成功后还要负责删除mempool中的交易
	state, err := exec.Commit(state, toCommitBlock)
	if err != nil {
		return state, err
	}

	// 更新state的状态
	_ = state.BlockTree.AddBlocks(proposal.LastBlockHash, proposal)
	state.LastBlockTime = time.Now()
	if len(toCommitBlock) > 0 {
		// 有提交的区块 更新state的slot
		lastBlock := toCommitBlock[len(toCommitBlock)-1]
		state.LastBlockSlot = lastBlock.Slot
		state.LastBlockHash = lastBlock.BlockHash
		state.LastResultsHash = lastBlock.ResultHash
		state.LastBlockTime = lastBlock.Ctime
		state.LastCommitedBlock = lastBlock
	}

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
			tx.Hash()
			//exec.logger.Debug(
			//	"commit tx",
			//	"tx", tx)
		}
		// TODO 检查交易执行的状态 如果有一个区块交易执行失败，直接结束本轮提交，是否需要rollback待定
		block.BlockState = types.CommittedBlock
		//block.ResultHash = newhash
		toRemovesTxs.Append(block.Txs)
	}

	// 提交成功后更新mempool，首先加锁
	exec.mempool.Lock()
	if err := exec.mempool.Update(0, toRemovesTxs); err != nil {
		exec.logger.Error("Update tx in mempool failed.", "reason", err)
	}
	exec.mempool.Unlock()

	newState := state.Copy()
	// 更新precommit blocks&suspected blocks
	exec.logger.Debug("to remove blocks.", "blocks", toCommitblocks, "size", newState.UnCommitBlocks.Size())
	newState.CommitBlocks(toCommitblocks)
	exec.logger.Debug("after remove.", "size", newState.UnCommitBlocks.Size())

	// TODO 更新提交后的merkle root
	newState.LastResultsHash = newState.LastResultsHash

	// 将Last字段更新为最后一个提交的区块的信息
	if len(toCommitblocks) > 0 {
		newState.LastBlockHash = toCommitblocks[len(toCommitblocks)-1].BlockHash
	}

	return newState, nil
}

// 从mempool中打包交易
func (exec *blockExcutor) CreateProposal(state State, curSlot types.LTime) *types.Proposal {
	// step 1 根据state中的信息，选出下一轮应该follow哪个区块
	parentBlock, precommitBlocks := state.NewBranch()

	// step 2 reap all tx from mempool
	// reap时候要选择没有冲突的交易
	txs := exec.mempool.ReapMaxTxs(-1)
	block := types.MakeBlock(txs)

	// step 3将区块头填补完整
	block.Fill(
		state.ChainID, curSlot,
		types.ProposalBlock,
		parentBlock.BlockHash,
		state.Validator.Address, state.Validators.Hash(),
		time.Now(),
	)
	block.Hash()

	// step 4 生成evidence
	block.Evidences = []types.Quorum{}
	for _, pblock := range precommitBlocks {
		block.Evidences = append(block.Evidences, pblock.VoteQuorum.Copy())
	}

	return &types.Proposal{
		Block: block,
	}
}

// 根绝当前的state验证一个区块是否合法
func (exec *blockExcutor) validateBlock(state State, block *types.Block) error {
	// 先检验区块基本的信息是否正确
	if err := block.ValidteBasic(); err != nil {
		return err
	}

	if !bytes.Equal(state.Validators.Hash(), block.ValidatorsHash) {
		// 验证者集合不一致
		return errors.New("block has different validator set")
	}

	return nil
}

package state

import (
	"bytes"
	mempl "chainbft_demo/mempool"
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

type BlockExecutorOption func(excutor *blockExecutor)

func NewBlockExecutor(mempool mempl.Mempool, options ...BlockExecutorOption) BlockExecutor {
	blockexec := &blockExecutor{
		mempool: mempool,
	}

	for _, option := range options {
		option(blockexec)
	}

	return blockexec
}

func SetBlockExecutorDB(inner Store) BlockExecutorOption {
	return func(excutor *blockExecutor) {
		excutor.stateDB = inner
	}
}

type blockExecutor struct {
	mempool mempl.Mempool

	stateDB Store

	logger log.Logger
}

// SetLogger implements BlockExecutor
func (exec *blockExecutor) SetLogger(logger log.Logger) {
	exec.logger = logger
}

// ApplyBlock implements BlockExecutor
// apply上一轮slot的proposal，同时会尝试更新pre-commit区块
func (exec *blockExecutor) ApplyBlock(state State, proposal *types.Block) (State, error) {
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
	state, _, err := exec.Commit(state, toCommitBlock)
	if err != nil {
		return state, err
	}

	return state, err
}

// TODO 当有多个可提交区块的时候，如何处理？
// 函数的语义：在当前state下，尽力提交所有可以提交的区块，如果提交成功后更新mempool
func (exec *blockExecutor) Commit(state State, toCommitblocks []*types.Block) (newState State, lastCommittedBlock *types.Block, err error) {
	newState = state
	for idx, block := range toCommitblocks {
		// TODO 当上一轮的区块提交失败时，是否要继续提交？
		exec.logger.Debug("commit block", "state", state, "idx", idx)

		// step 1 提交到状态数据库
		resulthash, err := exec.stateDB.CommitBlock(state, block)
		if err != nil {
			exec.logger.Error("commit block failed.", "err", err, "block", block.Hash())
			continue
		}

		// step 2 更新mempool，删除交易
		exec.mempool.Lock()

		if err := exec.mempool.Update(0, block.Txs); err != nil {
			exec.logger.Error("Update tx in mempool failed.", "reason", err)
			exec.mempool.Unlock()
			continue
		}
		exec.mempool.Unlock()

		// step 3 更新state
		tmpState := newState.Copy()
		tmpState.CommitBlock(block)
		tmpState.LastResultsHash = resulthash
		tmpState.LastBlockSlot = block.Slot
		tmpState.LastBlockHash = block.BlockHash
		tmpState.LastBlockTime = block.ProposalTime
		tmpState.LastCommitedBlock = block

		newState = tmpState
		lastCommittedBlock = block
	}

	return
}

// 从mempool中打包交易
func (exec *blockExecutor) CreateProposal(state State, curSlot types.LTime) *types.Proposal {
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
func (exec *blockExecutor) validateBlock(state State, block *types.Block) error {
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

package state

import (
	"bytes"
	"chainbft_demo/types"
	tmtype "github.com/tendermint/tendermint/types"
	"time"
)

func MakeGenesisState(
	chainID string,
	InitialSlot types.LTime,
	genesisBlock *types.Block,
) State {
	state := NewState(chainID, InitialSlot)
	state.BlockTree = types.NewBlockTree(genesisBlock)
	return state
}

func NewState(
	chainID string,
	InitialSlot types.LTime) State {
	return State{
		ChainID:        chainID,
		InitialSlot:    InitialSlot,
		Validators:     tmtype.NewValidatorSet([]*tmtype.Validator{}),
		UnCommitBlocks: types.NewBlockSet(),
	}

}

// 有限确定状态机的一个状态节点
// 节点里应维护所有和区块相关的信息
// 理论上一次共识完成后一定会使状态节点发生
type State struct {
	// 初始设定值 const value
	ChainID     string
	InitialSlot types.LTime // 初始Slot
	Validators  *tmtype.ValidatorSet

	// 最后提交的区块的信息
	LastBlockSlot types.LTime
	LastBlockHash []byte
	LastBlockTime time.Time // 提交的时间 - 物理时间

	// uncommitted blocks
	// 查询操作的比重会很大 - 能在PreCommitBlocks快速找到blockhash对应的区块
	UnCommitBlocks *types.BlockSet

	// block tree - 所有收到非error区块组织形成的树，根节点一定是genesis block
	BlockTree *types.BlockTree

	// 最后提交区块的结果集的hash or merkle root？
	LastResultsHash []byte
}

// 返回当前state的拷贝副本，deepcopy
func (state *State) Copy() State {
	newState := State{
		ChainID:         state.ChainID,
		InitialSlot:     state.InitialSlot,
		LastBlockSlot:   state.LastBlockSlot,
		LastBlockHash:   make([]byte, len(state.LastBlockHash)),
		LastBlockTime:   state.LastBlockTime,
		UnCommitBlocks:  state.UnCommitBlocks,
		BlockTree:       state.BlockTree,
		LastResultsHash: make([]byte, len(state.LastResultsHash)),
	}

	copy(newState.LastBlockHash, state.LastBlockHash)
	copy(newState.LastResultsHash, state.LastResultsHash)

	return newState
}

// NewBranch 遵循正常扩展分支的扩展逻辑，返回一个新的区块应该follow的区块 - 一般返回最长/深的区块
// 返回应该follow的区块，以及这个区块所在路径上所precommit的区块list
// 在收到新的区块以前，重复调用保持幂等性
func (state *State) NewBranch() (*types.Block, []*types.Block) {
	b := state.BlockTree.GetLatestBlock()
	precommitBlocks := state.BlockTree.GetBlockByFilter(b.Hash(), func(block *types.Block) bool {
		if block.BlockState == types.PrecommitBlock {
			return true
		}
		return false
	})

	return b, precommitBlocks
}

// IsMatch 判断一个提案是否符合提案规则
func (state *State) IsMatch(proposal *types.Proposal) bool {
	b := state.BlockTree.GetLatestBlock()

	return bytes.Equal(b.Hash(), proposal.LastBlockHash)
}

func (state *State) CommitBlocks(blocks []*types.Block) {
	state.UnCommitBlocks.RemoveBlocks(blocks)
}

// decideCommitBlocks 在当前状态，根据新的block给出可以提交的区块
// 要为每个可以提交的区块生成commit
func (state *State) decideCommitBlocks(block *types.Block) []*types.Block {
	toCommitBlocks := []*types.Block{}

	blocks := state.UnCommitBlocks.Blocks()
	idx := -1
	// 从后往前找到第一个可以提交的区块
	for i := len(blocks) - 1; i >= 0; i-- {
		if blocks[i].Commit == nil {
			continue
		}
		if blocks[i].Commit.IsReady() == true {
			idx = i
			break
		}
	}

	if idx != -1 {
		// 最后一个可以提交的区块开始，前面的区块都可以提交
		toCommitBlocks = append(toCommitBlocks, blocks[0:idx+1]...)
	}

	return toCommitBlocks
}

// UpdateState 每次收到新的提案，不论提案的正确与否，都将block内的support quorum更新到对应的区块上
func (state *State) UpdateState(block *types.Block) {
	// 如果evidence quorum不为空，更新以往到区块的信息
	if block.Evidences != nil {
		for _, evidence := range block.Evidences {
			// TODO 首先检验evidence的正确性 - 签名的正确性

			blockhash := evidence.BlockHash
			block := state.UnCommitBlocks.QueryBlockByHash(blockhash)
			if block == nil {
				continue
			}

			if block.Commit == nil {
				block.Commit = &types.Commit{}
			}

			if block.BlockState == types.CommiitedBlock {
				// 已经提交的区块
				continue
			}

			block.VoteQuorum = evidence
			// 更新blockstate
			if evidence.Type == types.SupportQuorum {
				block.BlockState = types.PrecommitBlock
			} else if evidence.Type == types.AgainstQuorum {
				block.BlockState = types.ErrorBlock
			}
		}
	}
}

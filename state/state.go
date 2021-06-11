package state

import (
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
		ChainID:         chainID,
		InitialSlot:     InitialSlot,
		Validators:      tmtype.NewValidatorSet([]*tmtype.Validator{}),
		PreCommitBlocks: types.NewBlockSet(),
		SuspectBlocks:   types.NewBlockSet(),
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
	PreCommitBlocks *types.BlockSet
	SuspectBlocks   *types.BlockSet

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
		PreCommitBlocks: state.PreCommitBlocks,
		SuspectBlocks:   state.SuspectBlocks,
		BlockTree:       state.BlockTree,
		LastResultsHash: make([]byte, len(state.LastResultsHash)),
	}

	copy(newState.LastBlockHash, state.LastBlockHash)
	copy(newState.LastResultsHash, state.LastResultsHash)

	return newState
}

// 遵循正常扩展分支的扩展逻辑，返回一个新的区块应该follow的区块 - 一般返回最长/深的区块
// TODO 在收到新的区块以前，重复调用保持幂等性
func (state *State) NewBranch() *types.Block {
	return &types.Block{
		Header:    types.Header{},
		Data:      types.Data{},
		Quorum:    types.Quorum{},
		Evidences: types.Quorum{},
		Commit:    nil,
	}
}

func (state *State) CommitBlocks(blocks []*types.Block) {
	state.PreCommitBlocks.RemoveBlocks(blocks)
	state.SuspectBlocks.RemoveBlocks(blocks)
}

// 在当前状态，根据新的block给出可以提交的区块 TODO
// 要为每个可以提交的区块生成commit
func (state *State) decideCommitBlocks(block *types.Block) []*types.Block {
	return []*types.Block{}
}

// 每收到一个区块尝试更新区块到合适的位置
func (state *State) UpdateState(block *types.Block) {
	if block.BlockState == types.SuspectBlock {
		state.SuspectBlocks.AddBlock(block)
	} else if block.BlockState == types.PrecommitBlock {
		state.PreCommitBlocks.AddBlock(block)
	}

	// TODO如果evidence quorum不为空，更新以往到区块的信息
	if !block.Evidences.IsEmpty() {

	}
}

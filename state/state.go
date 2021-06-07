package state

import (
	"chainbft_demo/types"
	"time"
)

func NewState(
	chainID string,
	InitialSlot types.LTime) State {
	return State{
		ChainID:     chainID,
		InitialSlot: InitialSlot,
	}
}

// 有限确定状态机的一个状态节点
// 节点里应维护所有和区块相关的信息
// 理论上一次共识完成后一定会使状态节点发生
type State struct {
	// 初始设定值 const value
	ChainID     string
	InitialSlot types.LTime // 初始Slot

	// 最后提交的区块的信息
	LastBlockSlot types.LTime
	LastBlockHash []byte
	LastBlockTime time.Time // 提交的时间 - 物理时间

	// uncommitted blocks
	// TODO 查询操作的比重会很大 - 能在PreCommitBlocks快速找到blockhash对应的区块
	PreCommitBlocks []*types.Block
	SuspectBlocks   []*types.Block

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
		PreCommitBlocks: make([]*types.Block, len(state.PreCommitBlocks)),
		SuspectBlocks:   make([]*types.Block, len(state.SuspectBlocks)),
		BlockTree:       &types.BlockTree{},
		LastResultsHash: make([]byte, len(state.LastResultsHash)),
	}

	copy(newState.LastBlockHash, state.LastBlockHash)

	return newState
}

// 遵循正常扩展分支的扩展逻辑，返回一个新的区块应该follow的区块
func (state *State) NewBranch() *types.Block {
	return nil
}

func (state *State) CommitBlocks(blocks []*types.Block) {
	for _, _ = range blocks {
		// TODO  把block从state中的PreCommitBlocks和SuspectBlocks移除
	}
}

// 在当前状态，根据新的block给出可以提交的区块
func (state *State) decideCommitBlocks(block *types.Block) []*types.Block {
	return state.PreCommitBlocks
}

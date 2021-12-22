package state

import (
	"bytes"
	"chainbft_demo/types"
	"time"
)

func MakeGenesisState(
	chainID string,
	InitialSlot types.LTime,
	genesisBlock *types.Block,
	val, pub_val *types.Validator,
	vals *types.ValidatorSet,
) State {
	state := NewState(chainID, InitialSlot, val, pub_val, vals)
	state.BlockTree = types.NewObrsBlockTree(genesisBlock)
	return state
}

func NewState(
	chainID string,
	InitialSlot types.LTime,
	val, pub_val *types.Validator,
	vals *types.ValidatorSet,
) State {
	return State{
		ChainID:     chainID,
		InitialSlot: InitialSlot,
		PubVal:      pub_val,
		Validator:   val,
		Validators:  vals,
		//UnCommitBlocks: types.NewBlockSet(),
	}
}

// 有限确定状态机的一个状态节点
// 节点里应维护所有和区块相关的信息
// 理论上一次共识完成后一定会使状态节点发生
type State struct {
	// 初始设定值 const value
	ChainID     string
	InitialSlot types.LTime // 初始Slot

	PubVal     *types.Validator // 原始公钥，hardcode到genesis block中，用来验证quorum的签名
	Validator  *types.Validator // 节点自己的验证者信息
	Validators *types.ValidatorSet

	// 最后提交的区块的信息
	LastBlockSlot     types.LTime
	LastCommitedBlock *types.Block
	LastBlockHash     []byte
	LastBlockTime     time.Time // 提交的时间 - 物理时间

	// uncommitted blocks
	// 查询操作的比重会很大 - 能在PreCommitBlocks快速找到blockhash对应的区块
	// UnCommitBlocks *types.BlockSet
	// UnCommitBlocks *types.BlockSet //删除该结构

	// block tree - 所有收到非error区块组织形成的树，根节点一定是genesis block
	BlockTree *types.ObrsBlockTree //替换成obrs结构

	// 最后提交区块的结果集的hash or merkle root？
	LastResultsHash []byte
}

// 返回当前state的拷贝副本，deepcopy
func (state *State) Copy() State {
	newState := State{
		ChainID:       state.ChainID,
		InitialSlot:   state.InitialSlot,
		LastBlockSlot: state.LastBlockSlot,
		LastBlockHash: make([]byte, len(state.LastBlockHash)),
		LastBlockTime: state.LastBlockTime,
		//UnCommitBlocks:  state.UnCommitBlocks,
		BlockTree:       state.BlockTree,
		PubVal:          state.PubVal,
		Validator:       state.Validator,
		Validators:      state.Validators,
		LastResultsHash: make([]byte, len(state.LastResultsHash)),
	}

	copy(newState.LastBlockHash, state.LastBlockHash)
	copy(newState.LastResultsHash, state.LastResultsHash)

	return newState
}

// NewBranch 遵循正常扩展分支的扩展逻辑，返回一个新的区块应该follow的区块 - 一般返回最长/深的区块
// 返回应该follow的区块
// 在收到新的区块以前，重复调用保持幂等性
func (state *State) NewBranch() *types.Block {
	b := state.BlockTree.GetLatestBlock()
	return b
}

// IsMatch 判断一个提案是否符合提案规则
func (state *State) IsMatch(proposal *types.Proposal) bool {
	b := state.BlockTree.GetLatestBlock()

	return bytes.Equal(b.Hash(), proposal.LastBlockHash)
}

// Commit一个区块，把该区块从UnCommitBLocks中移除，放入到BlockTree中
func (state *State) CommitBlock(block *types.Block) {
	state.BlockTree.Commit(block.BlockHash)
}

func (state *State) CommitBlocks(blocks []*types.Block) {
	for i := range blocks {
		state.CommitBlock(blocks[i])
	}
}

// decideCommitBlocks 在当前状态，根据新的block给出可以提交的区块
// 要为每个可以提交的区块生成commit
func (state *State) decideCommitBlocks(block *types.Block) []*types.Block {
	//toCommitBlocks := []*types.Block{}
	toCommitBlocks := state.BlockTree.GetBlockCanCommit()
	return toCommitBlocks
}

func (state *State) GenesisBlock() *types.Block {
	return state.BlockTree.GetRoot()
}

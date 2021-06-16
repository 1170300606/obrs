package state

import (
	"chainbft_demo/types"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewBranch(t *testing.T) {
	chainID := "state_test"
	genBlock := types.MakeGenesisBlock(chainID, []byte("signature"))
	genState := MakeGenesisState(chainID, types.LtimeZero, genBlock)

	treeCase := []struct {
		slot       int64
		previdx    int
		blockState types.BlockState
		expected   int //加入该区块前应该follow哪个区块
	}{
		{1, 0, types.SuspectBlock, 0},
		{2, 1, types.PrecommitBlock, 1},
		{3, 0, types.PrecommitBlock, 2},
		{4, 1, types.PrecommitBlock, 3},
		{5, 4, types.SuspectBlock, 4},
		{6, 2, types.SuspectBlock, 5},
	}

	blocks := make([]*types.Block, len(treeCase)+1, len(treeCase)+1)
	blocks[0] = genBlock
	// 实例化树
	for idx, node := range treeCase {
		actual, _ := genState.NewBranch()
		assert.Equal(t, blocks[node.expected], actual)

		b := types.MakeBlock([]types.Tx{[]byte(fmt.Sprintf("%v", idx))})
		b.Fill(chainID,
			types.LTime(node.slot),
			node.blockState,
			blocks[node.previdx].Hash(),
			[]byte(""))
		blocks[node.slot] = b
		b.Hash()
		assert.NotNil(t, b.LastBlockHash)
		assert.NotNil(t, b.BlockHash)
		genState.BlockTree.AddBlocks(b.LastBlockHash, b)
	}

	// 应该返回blocks[5]
	actual, blist := genState.NewBranch()
	assert.Equal(t, 1, len(blist))
	assert.Equal(t, blocks[5], actual)

	// 如果最后blocks[5]变为最新的precommit区块 应该在该区块后面追加
	blocks[5].BlockState = types.PrecommitBlock
	actual, blist = genState.NewBranch()
	assert.Equal(t, 2, len(blist))
	assert.Equal(t, blocks[5], actual)

	// 如果block[4] block[5]都提交了 下一轮应该返回block[3]
	blocks[1].BlockState = types.CommiitedBlock
	blocks[4].BlockState = types.CommiitedBlock
	blocks[5].BlockState = types.CommiitedBlock
	actual, blist = genState.NewBranch()
	assert.Equal(t, 1, len(blist))
	assert.Equal(t, blocks[3], actual)

	// 如果block[3]提交 应该返回block[6]
	blocks[3].BlockState = types.CommiitedBlock
	actual, blist = genState.NewBranch()
	assert.Equal(t, 1, len(blist))
	assert.Equal(t, blocks[6], actual)
}

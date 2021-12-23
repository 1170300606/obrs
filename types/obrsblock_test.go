package types

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// 生成一个obrstree
// 返回生成的树
func newObrsBlockTree() *ObrsBlockTree {
	chainID := "CONSENSUS_TEST"
	geneBlock := MakeGenesisBlock(chainID, time.Now())
	geneBlock.fillHeader()
	fmt.Println(geneBlock.BlockHash)
	tree := NewObrsBlockTree(geneBlock)
	return tree
}

//生成一个空区块
func newblock() *Block {
	block := &Block{}
	return block
}

// 测试AddBlcok函数
// 向root节点添加一个
func TestAddBlockTest(t *testing.T) {
	tree := newObrsBlockTree()
	data := newblock()
	tree.AddBlocks(tree.root.data.BlockHash, data)
	fmt.Println(tree.root.data.BlockHash.String())
	if len(tree.root.children) > 0 {
		assert.Equal(t, tree.root.children[0].data, data)
	} else {
		assert.Error(t, errors.New("wrong"))
	}
}

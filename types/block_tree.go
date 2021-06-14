package types

import (
	"bytes"
	"errors"
	"sync"
)

var (
	ErrDuplicatedBlock = errors.New("Duplicated block data in Block Tree")
	ErrNoQueryBlock    = errors.New("No such Block queried by hash value")
)

func NewBlockTree(genBlock *Block) *BlockTree {
	root := &treeNode{
		prev:     nil,
		children: nil,
		data:     genBlock,
	}
	return &BlockTree{
		root:     root,
		lastNode: root,
	}
}

// block组成的多叉树
type BlockTree struct {
	mtx            sync.RWMutex
	root, lastNode *treeNode
}

type treeNode struct {
	mtx      sync.RWMutex
	prev     *treeNode
	children []*treeNode
	data     *Block
}

// 在树中插入一个节点，根据hash确定父节点
// 如果父节点为空或反复插入同样的节点数据返回error
func (tree *BlockTree) AddBlocks(parentHash []byte, data *Block) error {
	parent, err := tree.queryNodeByHash(parentHash)
	if err != nil {
		return ErrNoQueryBlock
	}

	// 查看是否已经有节点在树中
	if n, err := tree.queryNodeByHash(data.BlockHash); err == nil && n != nil {
		return ErrDuplicatedBlock
	}

	newNode := &treeNode{
		prev:     parent,
		children: make([]*treeNode, 1),
		data:     data,
	}
	tree.lastNode = newNode
	tree.mtx.Lock()
	defer tree.mtx.Unlock()
	parent.mtx.Lock()
	defer parent.mtx.Unlock()
	parent.children = append(parent.children, newNode)

	return nil
}

// 广搜查找
func (tree *BlockTree) queryNodeByHash(hash []byte) (*treeNode, error) {
	tree.mtx.RLock()
	defer tree.mtx.RUnlock()

	if tree.root == nil {
		return nil, ErrNoQueryBlock
	}

	queue := []*treeNode{tree.root}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if bytes.Equal(hash, cur.data.BlockHash) {
			return cur, nil
		}
		queue = append(queue, cur.children...)
	}

	return nil, ErrNoQueryBlock
}

func (tree *BlockTree) QueryBlockByHash(hash []byte) (*Block, error) {
	tnode, err := tree.queryNodeByHash(hash)
	if tnode != nil {
		return tnode.data, err
	}
	return nil, err
}

// TODO 找到树最新的区块 - 定义参见协议细节
func (tree *BlockTree) GetLatestBlock() (*Block, error) {
	return tree.lastNode.data, nil
}

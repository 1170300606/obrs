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

type FilterFunc func(block *Block) bool

func NewBlockTree(genBlock *Block) *BlockTree {
	root := &treeNode{
		parent:   nil,
		children: []*treeNode{},
		data:     genBlock,
	}
	return &BlockTree{
		root:     root,
		lastNode: root,
		size:     1,
	}
}

// block组成的多叉树
type BlockTree struct {
	mtx            sync.RWMutex
	size           int
	root, lastNode *treeNode
}

type treeNode struct {
	mtx      sync.RWMutex
	parent   *treeNode
	children []*treeNode
	data     *Block
}

// 在树中插入一个节点，根据hash确定父节点
// 如果父节点为空或反复插入同样的节点数据返回error
func (tree *BlockTree) AddBlocks(parentHash []byte, data *Block) error {
	tree.mtx.Lock()
	defer tree.mtx.Unlock()

	parent, err := tree.queryNodeByHash(parentHash)
	if err != nil {
		return ErrNoQueryBlock
	}

	// 查看是否已经有节点在树中
	if n, err := tree.queryNodeByHash(data.BlockHash); err == nil && n != nil {
		return ErrDuplicatedBlock
	}

	newNode := &treeNode{
		parent:   parent,
		children: []*treeNode{},
		data:     data,
	}
	if data.Slot.Greater(tree.lastNode.data.Slot) {
		tree.lastNode = newNode
	}
	tree.lastNode = newNode
	parent.mtx.Lock()
	defer parent.mtx.Unlock()
	parent.children = append(parent.children, newNode)
	tree.size += 1
	return nil
}

// QueryBlockByHash根据区块hash查找区块
func (tree *BlockTree) QueryBlockByHash(hash []byte) (*Block, error) {
	tnode, err := tree.queryNodeByHash(hash)
	if tnode != nil {
		return tnode.data, err
	}
	return nil, err
}

// GetBlockByFilter 返回到指定hash节点的路径上所有满足filter函数且未提交的区块list
func (tree *BlockTree) GetBlockByFilter(hash []byte, filter FilterFunc) []*Block {
	tree.mtx.RLock()
	defer tree.mtx.RUnlock()
	res := []*Block{}

	endBlock, err := tree.queryNodeByHash(hash)
	if err != nil {
		return res
	}

	for cur := endBlock; cur != nil && cur.data.BlockState != CommiitedBlock; cur = cur.parent {
		if filter(cur.data) {
			res = append(res, cur.data)
		}
	}

	return res
}

// 找到树最新的区块 - 定义参见协议细节
// 当有precommit block时，返回precommit下面slot最新的区块
// 如果没有不存在precommit 则返回最新的slot
func (tree *BlockTree) GetLatestBlock() *Block {
	tree.mtx.RLock()
	defer tree.mtx.RUnlock()

	// 记录最新一个precommit block
	LastPrecommitNode := tree.root
	// 记录最新一个precommit block后代(包括自己)中最新的区块
	LastSlotBlock := tree.root.data

	queue := []*treeNode{tree.root}

	// 层次遍历找到一个最大slot的pre-commit block
	for len(queue) > 0 {
		cur := queue[0]
		if cur == nil {
			continue
		}

		curBlock := cur.data
		if curBlock.BlockState == PrecommitBlock && curBlock.Slot.Greater(LastPrecommitNode.data.Slot) {
			LastPrecommitNode = cur
			// precommit
			LastSlotBlock = curBlock
		} else if cur.data.Slot.Greater(LastSlotBlock.Slot) && LastPrecommitNode.isAncestor(cur) {
			// 避免判断自己是自己的祖先这样的判断
			// 一个区块是最后一个precommit block的后代，且比较新
			LastSlotBlock = curBlock
		}

		queue = queue[1:]
		queue = append(queue, cur.children...)
	}

	return LastSlotBlock
}

func (tree *BlockTree) Size() int {
	return tree.size
}

// isAncestor 判断tnode是否是other的祖先节点
// NOTE 调用者负责加锁
func (tnode *treeNode) isAncestor(other *treeNode) bool {
	for cur := other.parent; cur != nil; cur = cur.parent {
		if cur == tnode {
			// 节点以指针的形式保存，直接比较
			return true
		}
	}
	return false
}

// 广搜查找
// NOTE caller负责加锁解锁
func (tree *BlockTree) queryNodeByHash(hash []byte) (*treeNode, error) {

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

		if cur.children != nil && len(cur.children) > 0 {
			queue = append(queue, cur.children...)
		}
	}

	return nil, ErrNoQueryBlock
}

func (tree *BlockTree) GetRoot() *Block {
	return tree.root.data
}

// ForEach 以层级遍历的顺序，对所有节点执行lambda函数
func (tree *BlockTree) ForEach(lambda func(block *Block)) {
	tree.mtx.RLock()
	defer tree.mtx.RUnlock()
	queue := []*treeNode{tree.root}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if len(cur.children) > 0 {
			queue = append(queue, cur.children...)
		}
		lambda(cur.data)
	}

}

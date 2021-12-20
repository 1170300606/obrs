package types

import (
	"bytes"
	"sync"
)

//为了分层考虑，所有的大写字母开头的函数，返回值不要出现树中定义的数据结构
var t = 2 //假设t的大小为2

func NewObrsBlockTree(genBlock *Block) *ObrsBlockTree {
	root := &obrsTreeNode{
		parent:   nil,
		children: []*obrsTreeNode{},
		data:     genBlock,
		height:   1,
		//commited: false,
	}
	return &ObrsBlockTree{
		root:     root,
		lastNode: root,
		size:     1,
	}
}

type ObrsBlockTree struct {
	mtx            sync.RWMutex
	size           int
	root, lastNode *obrsTreeNode
}

type obrsTreeNode struct {
	mtx      sync.RWMutex
	parent   *obrsTreeNode
	children []*obrsTreeNode
	data     *Block
	height   int
	//commited bool //表示该节点是否被提交到数据库，如果已经被提交则显示true，否则是false
}

// 在树中插入一个节点，根据hash确定父节点
// @parentHash 是父节点的hash，用于寻找父节点
// @data是新加入节点的区块数据部分
// 如果父节点为空或反复插入同样的节点数据返回error
func (tree *ObrsBlockTree) AddBlocks(parentHash []byte, data *Block) error {
	parent, err := tree.queryNodeByHash(parentHash)
	if err != nil {
		return ErrNoQueryBlock
	}

	// 查看是否已经有节点在树中
	if n, err := tree.queryNodeByHash(data.BlockHash); err == nil && n != nil {
		return ErrDuplicatedBlock
	}

	newNode := &obrsTreeNode{
		parent:   parent,
		children: []*obrsTreeNode{},
		data:     data,
		height:   (parent.height + 1),
		//commited: false,
	}
	parent.mtx.Lock()
	defer parent.mtx.Unlock()
	parent.children = append(parent.children, newNode)
	if newNode.height > tree.lastNode.height {
		tree.lastNode = newNode //如果当前链的长度大于现存主链的长度，切换主链
	}
	tree.size += 1
	return nil
}

// 广搜查找
// @hash,节点hash,用于查找节点
// NOTE caller负责加锁解锁
// 返回查找到的节点，以及一个error用来表示查找成功与否
func (tree *ObrsBlockTree) queryNodeByHash(hash []byte) (*obrsTreeNode, error) {

	if tree.root == nil {
		return nil, ErrNoQueryBlock
	}

	queue := []*obrsTreeNode{tree.root}

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

// 提交节点
// @filter ???
// 从当前主链先前跳过pending节点后,将剩下的未提交的节点提交
// 返回一个[],用于储存所有需要提交的节点
func (tree *ObrsBlockTree) GetBlockByFilter(filter FilterFunc) []*Block {
	tree.mtx.RLock()
	defer tree.mtx.RUnlock()
	res := []*Block{}

	endBlock := tree.lastNode
	for i := 0; i < 2*t+1; i++ { //向前跳过2T+1个节点
		endBlock = endBlock.parent
		if endBlock == nil {
			return res //如果到头了就提前退出
		}
	}
	for cur := endBlock; cur != nil && cur.data.BlockState != CommittedBlock; cur = cur.parent {
		if filter(cur.data) {
			res = append(res, cur.data)
		}
	}
	return res
}

// 返回最新区块
// 返回lastnode的数据阶段
func (tree *ObrsBlockTree) GetLatestBlock() *Block {
	return tree.lastNode.data
}

// 返回tree大小
// 返回tree.size
func (tree *ObrsBlockTree) Size() int {
	return tree.size
}

// 返回tree根节点
// 返回tree.root.data
func (tree *ObrsBlockTree) GetRoot() *Block {
	return tree.root.data
}

package types

import (
	"bytes"
	"sync"
)

// BlockSet 用来表示区块的一个集合，里面的数据会频繁更新，读写均衡
// 正常情况集合内不会维护太多数据
type BlockSet struct {
	mtx    sync.RWMutex
	blocks []*Block
}

func NewBlockSet() *BlockSet {
	return &BlockSet{
		blocks: []*Block{},
	}
}

func (bs *BlockSet) QueryBlockByHash(hash []byte) *Block {
	for _, block := range bs.blocks {
		if bytes.Equal(block.BlockHash, hash) {
			return block
		}
	}
	return nil
}

func (bs *BlockSet) AddBlock(b *Block) {
	bs.mtx.Lock()
	defer bs.mtx.Unlock()
	bs.blocks = append(bs.blocks, b)
}

func (bs *BlockSet) AddBlocks(blocks ...*Block) {
	bs.mtx.Lock()
	defer bs.mtx.Unlock()
	bs.blocks = append(bs.blocks, blocks...)
}

// 将blocks从BlockSet中删除
func (bs *BlockSet) RemoveBlocks(blocks []*Block) {
	bs.mtx.Lock()
	bs.mtx.Unlock()

	queryMap := make(map[*Block]struct{})

	for _, block := range blocks {
		queryMap[block] = struct{}{}
	}

	newBlocks := []*Block{}
	for _, block := range bs.blocks {
		if _, ok := queryMap[block]; !ok {
			newBlocks = append(newBlocks, block)
		}
	}

	bs.blocks = newBlocks
}

func (bs *BlockSet) Size() int {
	return len(bs.blocks)
}

func (bs *BlockSet) Blocks() []*Block {
	return bs.blocks
}

func (bs *BlockSet) ForEach(lambda func(*Block)) {
	bs.mtx.RLock()
	defer bs.mtx.RUnlock()
	for _, block := range bs.blocks {
		lambda(block)
	}
}

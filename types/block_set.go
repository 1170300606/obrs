package types

import "bytes"

// BlockSet 用来表示区块的一个集合，里面的数据会频繁更新，读写均衡
// 正常情况集合内不会维护太多数据
type BlockSet struct {
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
	bs.blocks = append(bs.blocks, b)
}

func (bs *BlockSet) AddBlocks(blocks ...*Block) {
	bs.blocks = append(bs.blocks, blocks...)
}

// 将blocks从BlockSet中删除
func (bs *BlockSet) RemoveBlocks(blocks []*Block) {
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

package types

import "bytes"

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

// TODO 将blocks从BlockSet中删除
func (bs *BlockSet) RemoveBlocks(blocks []*Block) {}

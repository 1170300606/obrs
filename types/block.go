package types

import (
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	"sync"
)

type BlockState uint8

const (
	ErrorBlock     = BlockState(1)  // 错误的区块，收到against-quorum的区块
	SuspectBlock   = BlockState(2)  // 没有收到任意quorum的区块
	PrecommitBlock = BlockState(3)  // supprot-quorum的区块
	CommiitedBlock = BlockState(20) // 处于PrecommitBlock的区块有suppror-quorum的后代区块
)

func (state BlockState) String() string {
	switch state {
	case ErrorBlock:
		return "ErrorBlock"
	case SuspectBlock:
		return "SuspectBlock"
	case PrecommitBlock:
		return "PrecommitBlock"
	case CommiitedBlock:
		return "CommiitedBlock"
	default:
		return "UnkownTypeBlock"
	}
}

// local blockchain维护的区块的基本单位
type Block struct {
	mtx      sync.Mutex
	Header   `json:"header""`
	Data     `json:"data"`
	Quorum   `json:"quorum"` // 当前区块收到的投票合法集合
	Evidence *Commit         `json:"evidence"` // 区块能够提交的证据 - 即proposer所有pre-commit的区块的support-quorum
}

// 检验一个block是否合法 - 这里的合法指的是没有明确的错误
func (block *Block) ValidteBasic() error {
	block.mtx.Lock()
	defer block.mtx.Unlock()

	return nil
}

// 填补各种hash value
// TODO
func (block *Block) Fill() {

}

type Header struct {
	// 基本的区块信息
	ChainID    string     `json:"chain_id"`
	Slot       LTime      `json:"slot"`
	BlockState BlockState `json:"block_state"`

	// 数据hash
	LastBlockHash  tmbytes.HexBytes `json:last_block_hash`   // 上一个区块的信息
	TxsHash        tmbytes.HexBytes `json:"txs_hash"`        // transactions
	ValidatorsHash tmbytes.HexBytes `json:"validators_hash"` // 提交当前区块时，共识组内的所有验证者的hash
	ResultHash     tmbytes.HexBytes `json:"result_hash"`     // 执行完transaction的结果hash

	BlockHash tmbytes.HexBytes `json:block_hash` // 当前区块的hash
}

type Data struct {
	Txs Txs `json:"txs"` // transcations
}

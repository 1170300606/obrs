package types

import (
	"github.com/tendermint/tendermint/crypto/merkle"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	"sync"
)

type BlockState uint8

const (
	DefaultBlock   = BlockState(0)  // 空白区块 该轮slot尚未收到任何提案
	ProposalBlock  = BlockState(1)  // 提案状态 尚未收到任何投票的区块，即提案
	ErrorBlock     = BlockState(2)  // 错误的区块，收到against-quorum的区块
	SuspectBlock   = BlockState(3)  // 没有收到任意quorum的区块
	PrecommitBlock = BlockState(4)  // supprot-quorum的区块
	CommiitedBlock = BlockState(20) // 处于PrecommitBlock的区块有suppror-quorum的后代区块
)

func (state BlockState) String() string {
	switch state {
	case DefaultBlock:
		return "DefaultBlock"
	case ProposalBlock:
		return "ProposalBlock"
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
	mtx        sync.Mutex
	Header     `json:"header""`
	Data       `json:"data"`
	VoteQuorum Quorum   `json:"quorum"`    // 当前区块收到的投票合法集合
	Evidences  []Quorum `json:"evidences"` //  指向前面区块的support-quorum
	Commit     *Commit  `json:"commit"`    // 区块能够提交的证据 - 即proposer所有pre-commit的区块的support-quorum
}

// 检验一个block是否合法 - 这里的合法指的是没有明确的错误
func (block *Block) ValidteBasic() error {
	block.mtx.Lock()
	defer block.mtx.Unlock()

	return nil
}

// 填补各种hash value
func (b *Block) fillHeader() {
	if b.TxsHash == nil {
		b.TxsHash = b.Data.Hash()
	}
}

func (b *Block) Hash() tmbytes.HexBytes {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	b.fillHeader()

	return b.Header.Hash()
}

type Header struct {
	// 基本的区块信息
	ChainID    string     `json:"chain_id"`
	Slot       LTime      `json:"slot"`
	BlockState BlockState `json:"block_state"` // 不参与hash的计算

	// 数据hash
	LastBlockHash  tmbytes.HexBytes `json:last_block_hash`   // 上一个区块的信息
	TxsHash        tmbytes.HexBytes `json:"txs_hash"`        // transactions
	ValidatorsHash tmbytes.HexBytes `json:"validators_hash"` // 提交当前区块时，共识组内的所有验证者的hash
	ResultHash     tmbytes.HexBytes `json:"result_hash"`     // 执行完transaction的结果hash TOREMOVE 这个值无法确定会导致hash发生变化，如果不参与hash的计算那么该值无任何意义

	BlockHash tmbytes.HexBytes `json:block_hash` // 当前区块的hash
}

func (h *Header) Fill(
	chainID string,
	Slot LTime,
	state BlockState,
	LastBlockHash []byte,
	validatorsHash []byte) {
	h.ChainID = chainID
	h.Slot = Slot
	h.BlockState = state
	h.LastBlockHash = LastBlockHash
	h.ValidatorsHash = validatorsHash
}

func (h *Header) Hash() tmbytes.HexBytes {
	if h == nil {
		return nil
	}
	if h.BlockHash == nil {
		h.BlockHash = merkle.HashFromByteSlices([][]byte{
			[]byte(h.ChainID),
			h.Slot.Hash(),
			h.LastBlockHash,
			h.TxsHash,
			h.ValidatorsHash,
		})
	}
	return h.BlockHash
}

type Data struct {
	Txs  Txs    `json:"txs"` // transcations
	hash []byte // temp value
}

func (d *Data) Hash() tmbytes.HexBytes {
	if d == nil {
		return (Txs{}).Hash()
	}
	if d.hash == nil {
		d.hash = d.Txs.Hash()
	}
	return d.hash
}

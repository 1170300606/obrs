package types

import (
	"errors"
	"github.com/tendermint/tendermint/crypto/merkle"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	"sync"
	"time"
)

type BlockState uint8

const (
	// block status
	DefaultBlock   = BlockState(0)  // 空白区块 该轮slot尚未收到任何提案
	ProposalBlock  = BlockState(1)  // 提案状态 尚未收到任何投票的区块，即提案
	ErrorBlock     = BlockState(2)  // 错误的区块，收到against-quorum的区块
	SuspectBlock   = BlockState(3)  // 没有收到任意quorum的区块
	PrecommitBlock = BlockState(4)  // supprot-quorum的区块
	CommittedBlock = BlockState(20) // 处于PrecommitBlock的区块有suppror-quorum的后代区块

	// time label
	BlockProposalTime  = "proposal_time"
	BlockPrecommitTime = "precommit_time"
	BlockCommitTime    = "commit_time"
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
	case CommittedBlock:
		return "CommittedBlock"
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

	if block.BlockState == ErrorBlock {
		return errors.New("block state is error")
	}

	if block.BlockHash == nil || len(block.BlockHash) == 0 {
		return errors.New("block had no blockhash")
	}

	if block.Signature == nil || len(block.Signature) == 0 {
		return errors.New("block had no signature")
	}

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
	ChainID       string     `json:"chain_id"`
	Slot          LTime      `json:"slot"`
	SlotStartTime time.Time  `json:"slot_start_time"` // 临时方案，添加一个切换到新slot的时间，来多个节点之间调整校正
	BlockState    BlockState `json:"block_state"`     // 不参与hash的计算
	ProposalTime  time.Time  `json:"proposal_time"`   // 区块产生的时间，如果是创世块的话，那么改时间则是系统开始运转的时间

	// 数据hash
	LastBlockHash  tmbytes.HexBytes `json:last_block_hash`   // 上一个区块的信息
	TxsHash        tmbytes.HexBytes `json:"txs_hash"`        // transactions
	ValidatorAddr  Address          `json:"validator_addr"`  // 提案者地址
	ValidatorsHash tmbytes.HexBytes `json:"validators_hash"` // 提交当前区块时，共识组内的所有验证者的hash
	ResultHash     tmbytes.HexBytes `json:"result_hash"`     // 执行完transaction的结果hash TOREMOVE 这个值无法确定会导致hash发生变化，如果不参与hash的计算那么该值无任何意义

	BlockHash tmbytes.HexBytes `json:"block_hash"` // 当前区块的hash
	Signature tmbytes.HexBytes `json:"signature"`  // 区块的签名，sign {chainID}{slot}{LastBlockHash}{TxsHash}{ValidatorsHash}{ResultHash}

	// timestamp，调用computeTime之前，保存的是对应状态的时间戳
	BlockPrecommitTime int64 `json:"precommit_time"` // 达成precommit状态的耗时
	BlockCommitTime    int64 `json:"commit_time"`    // 达成precommit状态的耗时
}

func (h *Header) Fill(
	chainID string,
	Slot LTime,
	state BlockState,
	LastBlockHash []byte,
	valdatorAddr []byte,
	validatorsHash []byte,
	slotStartTime time.Time,
) {
	h.ChainID = chainID
	h.Slot = Slot
	h.BlockState = state
	h.LastBlockHash = LastBlockHash
	h.ValidatorAddr = valdatorAddr
	h.ValidatorsHash = validatorsHash
	h.SlotStartTime = slotStartTime
}

func (h *Header) Hash() tmbytes.HexBytes {
	if h == nil {
		return nil
	}
	startTimeHash, _ := h.SlotStartTime.MarshalBinary()
	if h.BlockHash == nil {
		h.BlockHash = merkle.HashFromByteSlices([][]byte{
			[]byte(h.ChainID),
			h.Slot.Hash(),
			h.LastBlockHash,
			h.TxsHash,
			h.ValidatorsHash,
			startTimeHash,
		})
	}
	return h.BlockHash
}

func (h *Header) CalculateTime() {
	precommitTime := h.BlockPrecommitTime - h.ProposalTime.UnixNano()
	commitTime := h.BlockCommitTime - h.BlockPrecommitTime
	h.BlockPrecommitTime = precommitTime
	h.BlockCommitTime = commitTime
}

func (h *Header) MarkTime(s string, t int64) {
	if s == BlockCommitTime {
		h.BlockCommitTime = t
	} else if s == BlockPrecommitTime {
		h.BlockPrecommitTime = t
	}
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

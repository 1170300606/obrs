package types

import (
	"errors"
	"github.com/tendermint/tendermint/crypto/merkle"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	"sync"
	"time"
)

// local blockchain维护的区块的基本单位
type Block struct {
	mtx    sync.Mutex
	Header `json:"header""`
	Data   `json:"data"`
	//VoteQuorum Quorum   `json:"quorum"`    // 当前区块收到的投票合法集合
	//Evidences  []Quorum `json:"evidences"` //  指向前面区块的support-quorum
	Commit *Commit `json:"commit"` // 区块能够提交的证据 - 即proposer所有pre-commit的区块的support-quorum
}

// 检验一个block是否合法 - 这里的合法指的是没有明确的错误
func (block *Block) ValidteBasic() error {
	block.mtx.Lock()
	defer block.mtx.Unlock()

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
	ChainID       string    `json:"chain_id"`
	Slot          LTime     `json:"slot"`
	SlotStartTime time.Time `json:"slot_start_time"` // 临时方案，添加一个切换到新slot的时间，来多个节点之间调整校正
	//BlockState    BlockState `json:"block_state"`     // 不参与hash的计算
	ProposalTime time.Time `json:"proposal_time"` // 区块产生的时间，如果是创世块的话，那么改时间则是系统开始运转的时间

	// 数据hash
	LastBlockHash  tmbytes.HexBytes `json:last_block_hash`   // 上一个区块的信息
	TxsHash        tmbytes.HexBytes `json:"txs_hash"`        // transactions
	ValidatorAddr  Address          `json:"validator_addr"`  // 提案者地址
	ValidatorsHash tmbytes.HexBytes `json:"validators_hash"` // 提交当前区块时，共识组内的所有验证者的hash
	ResultHash     tmbytes.HexBytes `json:"result_hash"`     // 执行完transaction的结果hash TOREMOVE 这个值无法确定会导致hash发生变化，如果不参与hash的计算那么该值无任何意义

	BlockHash tmbytes.HexBytes `json:"block_hash"` // 当前区块的hash
	Signature tmbytes.HexBytes `json:"signature"`  // 区块的签名，sign {chainID}{slot}{LastBlockHash}{TxsHash}{ValidatorsHash}{ResultHash}

	// timestamp，保存的是对应状态的时间戳
	BlockPrecommitTime int64 `json:"precommit_timestamp"` // 达成precommit状态的时间戳
	BlockCommitTime    int64 `json:"commit_timestamp"`    // 达成commit状态的时间戳

	TimePrecommit int64 `json:"precommit_time"` // 达成precommit状态的耗时
	TimeCommit    int64 `json:"commit_time"`    // 达成commit状态的耗时
	TimeConsensus int64 `json:"consensus_time"` // 达成commit的总耗时
}

func (h *Header) Fill(
	chainID string,
	Slot LTime,
	//state BlockState,
	LastBlockHash []byte,
	valdatorAddr []byte,
	validatorsHash []byte,
	slotStartTime time.Time,
) {
	h.ChainID = chainID
	h.Slot = Slot
	//h.BlockState = state
	h.LastBlockHash = LastBlockHash
	h.ValidatorAddr = valdatorAddr
	h.ValidatorsHash = validatorsHash
	h.SlotStartTime = slotStartTime
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

func (h *Header) CalculateTime() {
	h.TimePrecommit = h.BlockPrecommitTime - h.ProposalTime.UnixNano()
	h.TimeCommit = h.BlockCommitTime - h.BlockPrecommitTime
	h.TimeConsensus = h.BlockCommitTime - h.ProposalTime.UnixNano()
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

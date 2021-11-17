package types

import (
	multisig "chainbft_demo/crypto/threshold"
	"chainbft_demo/types"
	"errors"
	"fmt"
)

var (
	ErrDuplicateVote = errors.New("duplicate vote")
)

func MakeSlotVoteSet() *SlotVoteSet {
	return &SlotVoteSet{
		slotVoteSet: make(map[types.LTime]*voteSet),
	}
}

type SlotVoteSet struct {
	slotVoteSet map[types.LTime](*voteSet)
}

// AddVote 将vote添加到相应的slot下面
func (slotvs *SlotVoteSet) AddVote(vote *types.Vote) error {
	slot := vote.Slot
	vs, exsit := slotvs.slotVoteSet[slot]
	if !exsit {
		vs = NewVoteSet(vote.BlockHash)
		slotvs.slotVoteSet[slot] = vs
	}
	return vs.AddVote(vote)
}

// GetVotesBySlot 根据slot返回对应的voteset，如果对应的slot没有voteset，则返回空的voteset
func (slotvs *SlotVoteSet) GetVotesBySlot(slot types.LTime) *voteSet {
	vs, exist := slotvs.slotVoteSet[slot]
	if !exist {
		return nil
	}
	return vs
}

// Size 返回slot对应的voteset的投票总数
func (slotvs *SlotVoteSet) Size(slot types.LTime) int {
	vs, exist := slotvs.slotVoteSet[slot]
	if !exist {
		return 0
	}
	return len(vs.votes)
}

func NewVoteSet(hash []byte) *voteSet {
	return &voteSet{
		blockHash: hash,
		votes:     []*types.Vote{},
	}
}

// voteSet 一个slot内的投票集合，暂时假定收到的投票都是同一个区块
type voteSet struct {
	blockHash []byte
	votes     []*types.Vote
}

func (vs *voteSet) AddVote(vote *types.Vote) error {
	for _, v := range vs.votes {
		if v.Equal(vote) {
			return ErrDuplicateVote
		}
	}
	vs.votes = append(vs.votes, vote)
	return nil
}

func (vs *voteSet) GetVotes() []*types.Vote {
	return vs.votes
}

// TryGenQuorum 尝试根据vote生成quorum - 还原出聚合签名，选出大多数
func (vs *voteSet) TryGenQuorum(threshold int) types.Quorum {
	supportVotes := []*types.Vote{}
	againstVotes := []*types.Vote{}
	for _, vote := range vs.votes {
		if vote.Type == types.AgainstVote {
			againstVotes = append(againstVotes, vote)
		} else if vote.Type == types.SupportVote {
			supportVotes = append(supportVotes, vote)
		}
	}
	var signature []byte

	var majorityVotes []*types.Vote
	quorumType := types.EmptyQuorum

	if len(supportVotes) >= threshold {
		majorityVotes = supportVotes
		// 有足够的赞成票
		quorumType = types.SupportQuorum
	} else if len(againstVotes) >= threshold {
		majorityVotes = againstVotes
		quorumType = types.AgainstQuorum
	} else {
		majorityVotes = nil
		quorumType = types.EmptyQuorum
	}

	// 门限签名还原出原始签名
	if majorityVotes != nil {
		sigs, ids := getSignsAndIdsFromVotes(majorityVotes)
		var err error
		signature, err = multisig.SignatureRecovery(threshold, sigs, ids)
		fmt.Println(err)
		if err != nil {
			quorumType = types.EmptyQuorum
			signature = nil
		}
	}

	return types.Quorum{
		BlockHash: vs.blockHash,
		Type:      quorumType,
		Signature: signature,
	}
}

func getSignsAndIdsFromVotes(votes []*types.Vote) ([][]byte, []int64) {
	sigs := make([][]byte, 0, len(votes))
	ids := make([]int64, 0, len(votes))

	for _, vote := range votes {
		ids = append(ids, int64(vote.ValidatorIndex+1))
		sigs = append(sigs, vote.Signature)
	}

	return sigs, ids
}

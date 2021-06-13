package types

import (
	"chainbft_demo/types"
	"errors"
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
		vs = NewVoteSet()
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

func NewVoteSet() *voteSet {
	return &voteSet{
		votes: []*types.Vote{},
	}
}

type voteSet struct {
	votes []*types.Vote
}

func (vs *voteSet) AddVote(vote *types.Vote) error {
	for _, v := range vs.votes {
		if v.Equal(vote) {
			return ErrDuplicateVote
		}
	}
	vs.votes = append(vs.votes)
	return nil
}

func (vs *voteSet) GetVotes() []*types.Vote {
	return vs.votes
}

// TODO TryGenQuorum 尝试根据vote生成quorum - 还原出聚合签名，选出大多数
func (vs *voteSet) TryGenQuorum() types.Quorum {

	signature := []byte("")
	return types.Quorum{
		BlockHash: nil,
		Type:      types.SupportQuorum,
		Signature: signature,
	}
}

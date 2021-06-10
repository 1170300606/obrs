package types

import (
	"chainbft_demo/types"
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
func (slotvs *SlotVoteSet) AddVote(vote *types.Vote) {
	slot := vote.Slot
	vs, exsit := slotvs.slotVoteSet[slot]
	if !exsit {
		vs = NewVoteSet()
		slotvs.slotVoteSet[slot] = vs
	}
	vs.AddVote(vote)
}

// GetVotesBySlot 根据slot返回对应的voteset，如果对应的slot没有voteset，则返回空的voteset
func (slotvs *SlotVoteSet) GetVotesBySlot(slot types.LTime) []*types.Vote {
	vs, exist := slotvs.slotVoteSet[slot]
	if !exist {
		return []*types.Vote{}
	}
	return vs.GetVotes()
}

func NewVoteSet() *voteSet {
	return &voteSet{
		votes: []*types.Vote{},
	}
}

type voteSet struct {
	votes []*types.Vote
}

func (vs *voteSet) AddVote(vote *types.Vote) {
	vs.votes = append(vs.votes)
}

func (vs *voteSet) GetVotes() []*types.Vote {
	return vs.votes
}

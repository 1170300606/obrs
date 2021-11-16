package consensus

import (
	"chainbft_demo/types"
	jsoniter "github.com/json-iterator/go"
	"time"
)

// TODO update metric
func newConsensusMetric() *consensusMetric {
	return &consensusMetric{
		Slot:                   0,
		SlotStartTime:          time.Time{},
		LastRevisedSlot:        -1,
		IsWorking:              false,
		RecieveProposal:        false,
		CommittedInCurrentSlot: false,
		RoundStatus:            "",
		IsProposer:             false,
		ProposerAddress:        "",
	}
}

type consensusMetric struct {
	Slot            int64     `json:"current_slot"`
	SlotStartTime   time.Time `json:"slot_start_time"`
	LastRevisedSlot int64     `json:"last_revised_slot"`

	IsWorking              bool   `json:"is_working"`
	RecieveProposal        bool   `json:"receiveProposal"`
	CommittedInCurrentSlot bool   `json:"committed_in_current_slot`
	RoundStatus            string `json:"current_round_status"`

	IsProposer      bool   `json:"is_proposer"`
	ProposerAddress string `json:"proposer_address"`
}

func (cm *consensusMetric) JSONString() string {
	s, _ := jsoniter.MarshalToString(cm)
	return s
}

func (cm *consensusMetric) MarkSlot(slot types.LTime) {
	cm.Slot = slot.Int64()
}

func (cm *consensusMetric) MarkSlotStartTime(t time.Time) {
	cm.SlotStartTime = t
}

func (cm *consensusMetric) MarkRevisedSlot(slot types.LTime) {
	cm.LastRevisedSlot = slot.Int64()
}

func (cm *consensusMetric) MarkIsWorking(v bool) {
	cm.IsWorking = v
}

func (cm *consensusMetric) MarkRecieveProposal(v bool) {
	cm.RecieveProposal = v
}

func (cm *consensusMetric) MarkBlockStatus(v string) {
	cm.RoundStatus = v
}

func (cm *consensusMetric) MarkIsProposer(v bool) {
	cm.IsProposer = v
}

func (cm *consensusMetric) MarkProposerAddr(v string) {
	cm.ProposerAddress = v
}

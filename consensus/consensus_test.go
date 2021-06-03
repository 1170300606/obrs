package consensus

import (
	bkstate "chainbft_demo/state"
	"chainbft_demo/types"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"os"
	"testing"
	"time"
)

type cleanup func()

func newConsensusState() (*ConsensusState, cleanup) {
	conR := NewReactor()

	cs, clean := newConsensusStateWithConfig(config.ResetTestRoot("consensus_test"))
	SetReactor(conR)
	return cs, clean
}

func newConsensusStateWithConfig(config *config.Config) (*ConsensusState, cleanup) {
	state := bkstate.NewState()
	cs := NewConsensusState(config.Consensus, nil, nil, state)

	cs.SetLogger(log.NewFilter(log.TestingLogger(), log.AllowDebug()))
	return cs, func() {
		os.RemoveAll(config.RootDir)
	}
}

func TestNewDefaultConsensusState(t *testing.T) {
	cs, clean := newConsensusState()
	defer clean()
	cs.OnStart()
	cs.enterNewSlot(types.LtimeZero) //手动触发
	time.Sleep(20 * time.Second)
	cs.OnStop()
}

func TestEnterAlppyInWrongWay(t *testing.T) {
	cs, clean := newConsensusState()
	defer clean()
	cs.OnStart()

	assert.Panics(t, func() {
		cs.enterApply()
	}) //手动触发

	cs.OnStop()
}

func TestEnterProposeInWrongWay(t *testing.T) {
	cs, clean := newConsensusState()
	defer clean()
	cs.OnStart()

	assert.Panics(t, func() {
		cs.enterPropose()
	}) //手动触发

	cs.OnStop()
}

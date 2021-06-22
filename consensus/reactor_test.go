package consensus

import (
	cstypes "chainbft_demo/consensus/types"
	"chainbft_demo/types"
	"github.com/stretchr/testify/assert"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
	"testing"
	"time"
)

// connect N consensus reactors through N switches
func makeAndConnectReactors(config *cfg.Config, logger log.Logger, n int, privs []types.PrivValidator, vals *types.ValidatorSet, pub_val *types.Validator) []*Reactor {
	reactors := make([]*Reactor, n)
	for i := 0; i < n; i++ {
		_, val := vals.GetByIndex(int32(i))
		cs, clean := newConsensusState(privs[i], val, pub_val, vals)
		defer clean()

		reactors[i] = NewReactor(cs) // so we dont start the consensus states
		reactors[i].SetLogger(logger.With("validator", i))
	}

	p2p.MakeConnectedSwitches(config.P2P, n, func(i int, s *p2p.Switch) *p2p.Switch {
		s.AddReactor("CONSENSUS", reactors[i])
		return s
	}, p2p.Connect2Switches)
	return reactors
}

// TestProposalWithSupportQuorum 测试一个节点负责提案，并且这个提案能被4个同意，最后检查提案是否处于precommit状态
// 肩擦好
func TestProposalWithSupportQuorum(t *testing.T) {
	count := 4
	config := cfg.ResetTestRoot("consensus_test")
	logger := log.NewFilter(log.TestingLogger(), log.AllowDebug())

	privs, vals, pub_val := newPrivAndValSet(count)
	reactors := makeAndConnectReactors(config, logger, count, privs, vals, pub_val)

	// reactor[0] 负责提案
	proposal := reactors[0].consensus.defaultProposal()

	assert.NotNil(t, proposal, "提案不应该为空")

	// 给通信一定时间后检查提案有没有收到
	time.Sleep(2 * time.Second)
	for i := 0; i < count; i++ {
		cs := reactors[i].consensus
		assert.NotNil(t, cs.Proposal, "#{%v}的proposal不应该为空", i)
		assert.Equalf(t, proposal.Hash(), cs.Proposal.Hash(), "#{%v}的proposal和leader不一样", i)
	}

	// 手动进入apply step
	for i := 0; i < count; i++ {
		cs := reactors[i].consensus
		cs.updateStep(cstypes.RoundStepSlot)
		cs.enterApply()
		assert.Equalf(t, types.PrecommitBlock, cs.Proposal.BlockState, "#{%v}的proposal应该位于precommit", i)
	}

}

package consensus

import (
	cstypes "chainbft_demo/consensus/types"
	"chainbft_demo/crypto/bls"
	threshold2 "chainbft_demo/crypto/threshold"
	mempl "chainbft_demo/mempool"
	"chainbft_demo/privval"
	bkstate "chainbft_demo/state"
	"chainbft_demo/types"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
	"os"
	"testing"
	"time"
)

type cleanup func()
type memplFunc func(mempool mempl.Mempool)

func newBlockExecutorWithConfig(config *config.Config, mempool mempl.Mempool, logger log.Logger) (bkstate.BlockExecutor, cleanup) {
	blockexec := bkstate.NewBlockExecutor(mempool)
	blockexec.SetLogger(logger)

	return blockexec, func() {
		os.RemoveAll(config.RootDir)
	}
}

func newConsensusState(
	prival types.PrivValidator,
	val, pub_val *types.Validator, vals *types.ValidatorSet,
	memplfunc ...memplFunc,
) (*ConsensusState, cleanup) {
	logger := log.NewFilter(log.TestingLogger(), log.AllowDebug())

	cs, clean := newConsensusStateWithConfig(config.ResetTestRoot("consensus_test"), logger, prival, val, pub_val, vals, memplfunc...)
	conR := NewReactor(cs)
	conR.SetLogger(logger)

	return cs, clean
}

func newConsensusStateWithConfig(
	config *config.Config, logger log.Logger,
	prival types.PrivValidator,
	val, pub_val *types.Validator, vals *types.ValidatorSet,
	memplfunc ...memplFunc,
) (*ConsensusState, cleanup) {
	chainID := "CONSENSUS_TEST"
	geneBlock := types.MakeGenesisBlock(chainID)

	state := bkstate.MakeGenesisState(chainID, types.LtimeZero, geneBlock, val, pub_val, vals)

	mempool := mempl.NewListMempool(config.Mempool)
	mempool.SetLogger(logger)
	blockExec, _ := newBlockExecutorWithConfig(config, mempool, logger)

	cs := NewDefaultConsensusState(config.Consensus, types.LtimeZero, prival, vals, blockExec, nil, state)

	cs.SetLogger(logger)

	for _, f := range memplfunc {
		f(mempool)
	}

	return cs, func() {
		os.RemoveAll(config.RootDir)
	}
}

// 生成指定数量的validator，
// 返回相等数量的私钥、验证者集合、公共公钥
func newPrivAndValSet(count int) ([]types.PrivValidator, *types.ValidatorSet, *types.Validator) {
	pub_priv := bls.GenTestPrivKey(int64(100))
	pub_val := types.NewValidator(pub_priv.PubKey())
	poly := threshold2.Master(pub_priv, 3, 1000)

	privs := []types.PrivValidator{}
	vallist := []*types.Validator{}

	for i := 0; i < count; i++ {
		priv, err := poly.GetValue(int64(i + 1))
		if err != nil {
			panic(err)
		}
		prival := privval.NewFilePV(priv, "")

		val := types.NewValidator(priv.PubKey())
		vallist = append(vallist, val)
		privs = append(privs, prival)
	}

	vals := types.NewValidatorSet(vallist)

	return privs, vals, pub_val
}

func TestNewDefaultConsensusState(t *testing.T) {
	priv, vals, pub_val := newPrivAndValSet(4)
	_, val := vals.GetByIndex(0)
	cs, clean := newConsensusState(priv[0], val, pub_val, vals)
	defer clean()
	cs.OnStart()
	cs.slotClock.OnStart()
	cs.enterNewSlot(types.LtimeZero) //手动触发
	time.Sleep(20 * time.Second)
	cs.OnStop()
}

func TestEnterAlppyInWrongWay(t *testing.T) {
	priv, vals, pub_val := newPrivAndValSet(4)
	_, val := vals.GetByIndex(0)
	cs, clean := newConsensusState(priv[0], val, pub_val, vals)
	defer clean()
	cs.OnStart()
	cs.slotClock.OnStart()

	assert.Panics(t, func() {
		cs.enterApply()
	}) //手动触发

	cs.OnStop()
}

func TestEnterProposeInWrongWay(t *testing.T) {
	priv, vals, pub_val := newPrivAndValSet(4)
	_, val := vals.GetByIndex(0)
	cs, clean := newConsensusState(priv[0], val, pub_val, vals)
	defer clean()
	cs.OnStart()
	cs.slotClock.OnStart()

	assert.Panics(t, func() {
		cs.enterPropose()
	}) //手动触发

	cs.OnStop()
}

// TestProposal 测试consensus能否正确的通过blockexec生成新的提案,同时测试下setproposal的正确性
// NOTE: proposal的正确性由blockexec保证
func TestProposal(t *testing.T) {
	priv, vals, pub_val := newPrivAndValSet(4)
	_, val := vals.GetByIndex(0)
	txs := []types.Tx{}
	cs, clean := newConsensusState(priv[0], val, pub_val, vals, func(mem mempl.Mempool) {
		// 默认创建10条交易
		for i := 0; i < 10; i++ {
			tx := []byte(fmt.Sprintf("tx=%v", i))
			txs = append(txs, tx)
			err := mem.CheckTx(tx, mempl.TxInfo{
				SenderID: mempl.UnknownPeerID,
			})
			assert.NoError(t, err, "add %vth tx into mempool failed.", i)
		}
	})

	defer clean()
	cs.OnStart()
	cs.slotClock.OnStart()

	// 手动进入proposal step，验证是否可以运行和正确打包
	cs.updateStep(cstypes.RoundStepApply)

	var proposal *types.Proposal

	assert.NotPanics(t, func() {
		proposal = cs.defaultProposal()
	}, "propose failed.")

	// 暂停一会保证setproposal正常执行
	time.Sleep(1 * time.Second)

	assert.Equal(t, proposal, cs.Proposal, "proposal不一致，setProposal失败")
}

// TestApply 测试consensus能否正确通过blockExec执行提交操作，consensus需要保证quorum的正确生成
// NOTE: state更新的正确性由blockexec保证
func TestApplyWithSuspectProposal(t *testing.T) {
	priv, vals, pub_val := newPrivAndValSet(4)
	_, val := vals.GetByIndex(0)
	txs := []types.Tx{}
	cs, clean := newConsensusState(priv[0], val, pub_val, vals, func(mem mempl.Mempool) {
		// 默认创建10条交易

		for i := 0; i < 10; i++ {
			tx := []byte(fmt.Sprintf("tx=%v", i))
			txs = append(txs, tx)
			err := mem.CheckTx(tx, mempl.TxInfo{
				SenderID: mempl.UnknownPeerID,
			})
			assert.NoError(t, err, "add %vth tx into mempool failed.", i)
		}

	})

	defer clean()
	cs.OnStart()

	// 手动设置提案
	cs.updateStep(cstypes.RoundStepApply)
	proposal := cs.defaultProposal()
	cs.defaultSetProposal(proposal)

	// 手动enterNewSlot
	cs.updateStep(cstypes.RoundStepSlot)

	prevState := cs.state
	assert.NotPanics(t, func() {
		cs.enterApply()
	})

	assert.Equal(t, prevState, cs.lastState)
	assert.NotEqual(t, prevState, cs.state)
	assert.Equal(t, types.SuspectBlock, proposal.BlockState) //  没有投票，该提案为suspect block
}

// TestApplyWithSuppotQuorum 测试consensus apply动作，设置的提案有足够的supportVote
func TestApplyWithSuppotQuorum(t *testing.T) {
	privs, vals, pub_val := newPrivAndValSet(4)
	_, val := vals.GetByIndex(0)
	txs := []types.Tx{}

	cs, clean := newConsensusState(privs[0], val, pub_val, vals, func(mem mempl.Mempool) {
		// 默认创建10条交易
		for i := 0; i < 10; i++ {
			tx := []byte(fmt.Sprintf("tx=%v", i))
			txs = append(txs, tx)
			err := mem.CheckTx(tx, mempl.TxInfo{
				SenderID: mempl.UnknownPeerID,
			})
			assert.NoError(t, err, "add %vth tx into mempool failed.", i)
		}
	})

	defer clean()
	cs.OnStart()

	// 手动进入proposal step，验证是否可以运行和正确打包
	cs.updateStep(cstypes.RoundStepApply)

	var proposal *types.Proposal

	assert.NotPanics(t, func() {
		proposal = cs.defaultProposal()
	}, "propose failed.")

	// sleep一会保证投票全部处理
	time.Sleep(1 * time.Second)

	// proposal的第一轮投票，应该全部添加成功
	for i := 1; i < len(privs); i++ {
		priv := privs[i]
		addr, _ := vals.GetByIndex(int32(i))
		vote := &types.Vote{
			Slot:             0,
			BlockHash:        proposal.Hash(),
			Type:             types.SupportVote,
			Timestamp:        time.Now(),
			ValidatorAddress: addr,
			ValidatorIndex:   int32(i),
			Signature:        nil,
		}

		err := priv.SignVote(cs.state.ChainID, vote)
		assert.NoError(t, err, "sign vote failed.", "index", i, "error", err)
		cs.peerMsgQueue <- msgInfo{&VoteMessage{vote}, p2p.ID(fmt.Sprintf("%v", i))} // 模拟收到vote
	}

	// sleep一会保证投票全部处理
	time.Sleep(1 * time.Second)

	// 手动enterNewSlot
	cs.updateStep(cstypes.RoundStepSlot)

	prevState := cs.state
	assert.NotPanics(t, func() {
		cs.enterApply()
	})

	assert.NotEqual(t, prevState, cs.state)
	assert.NotNil(t, proposal.VoteQuorum, "proposal should have vote quorum")
	assert.Equal(t, proposal.VoteQuorum.Type, types.SupportQuorum,
		"proposal's voteQuorum should be supprot, but got %v",
		proposal.VoteQuorum.Type.String(),
	)
	assert.Equal(t, types.PrecommitBlock, proposal.BlockState) //  有足够的投票，该提案为precommit block
}

// TestTryAddVote 测试能否添加投票，以及能否防止重复投票
func TestTryAddVote(t *testing.T) {
	privs, vals, pub_val := newPrivAndValSet(4)
	_, val := vals.GetByIndex(0)
	cs, clean := newConsensusState(privs[0], val, pub_val, vals)

	defer clean()
	cs.OnStart()

	// 手动进入proposal step，验证是否可以运行和正确打包
	cs.updateStep(cstypes.RoundStepApply)

	var proposal *types.Proposal

	assert.NotPanics(t, func() {
		proposal = cs.defaultProposal()
	}, "propose failed.")

	// proposal的第一轮投票，应该全部添加成功
	for i := 1; i < len(privs); i++ {
		priv := privs[i]
		addr, _ := vals.GetByIndex(int32(i))
		vote := &types.Vote{
			Slot:             0,
			BlockHash:        proposal.Hash(),
			Type:             types.SupportVote,
			Timestamp:        time.Now(),
			ValidatorAddress: addr,
			ValidatorIndex:   int32(i),
			Signature:        nil,
		}

		err := priv.SignVote(cs.state.ChainID, vote)
		assert.NoError(t, err, "sign vote failed.", "index", i, "error", err)
		cs.peerMsgQueue <- msgInfo{&VoteMessage{vote}, p2p.ID(fmt.Sprintf("%v", i))} // 模拟收到vote
	}

	// 给足够的时间处理投票
	time.Sleep(1 * time.Second)
	assert.Equalf(t, 4, cs.VoteSet.Size(cs.CurSlot), "should receive 4 votes, but got %v", cs.VoteSet.Size(cs.CurSlot))

	for i := 1; i < len(privs); i++ {
		priv := privs[i]
		addr, _ := vals.GetByIndex(int32(i))
		vote := &types.Vote{
			Slot:             0,
			BlockHash:        proposal.Hash(),
			Type:             types.SupportVote,
			Timestamp:        time.Now(),
			ValidatorAddress: addr,
			ValidatorIndex:   int32(i),
			Signature:        nil,
		}

		err := priv.SignVote(cs.state.ChainID, vote)
		assert.NoError(t, err, "sign vote failed.", "index", i, "error", err)
		cs.peerMsgQueue <- msgInfo{&VoteMessage{vote}, p2p.ID(fmt.Sprintf("%v", i))} // 模拟收到vote
	}

	// 给足够的时间处理投票
	time.Sleep(1 * time.Second)
	assert.Equalf(t, 4, cs.VoteSet.Size(cs.CurSlot), "should receive 4 votes, but got %v", cs.VoteSet.Size(cs.CurSlot))
}

// TestApplyPrecommitBlock 测试，提交一个收到足够投票的提案
func TestApplyPrecommitBlock(t *testing.T) {
	privs, vals, pub_val := newPrivAndValSet(4)
	_, val := vals.GetByIndex(0)
	cs, clean := newConsensusState(privs[0], val, pub_val, vals)

	defer clean()
	cs.OnStart()

	// 手动进入proposal step，验证是否可以运行和正确打包
	cs.updateStep(cstypes.RoundStepApply)

	var proposal *types.Proposal

	assert.NotPanics(t, func() {
		proposal = cs.defaultProposal()
	}, "propose failed.")

	// proposal的第一轮投票，应该全部添加成功
	for i := 1; i < len(privs); i++ {
		priv := privs[i]
		addr, _ := vals.GetByIndex(int32(i))
		vote := &types.Vote{
			Slot:             0,
			BlockHash:        proposal.Hash(),
			Type:             types.SupportVote,
			Timestamp:        time.Now(),
			ValidatorAddress: addr,
			ValidatorIndex:   int32(i),
			Signature:        nil,
		}

		err := priv.SignVote(cs.state.ChainID, vote)
		assert.NoError(t, err, "sign vote failed.", "index", i, "error", err)
		cs.peerMsgQueue <- msgInfo{&VoteMessage{vote}, p2p.ID(fmt.Sprintf("%v", i))} // 模拟收到vote
	}

	// 给足够的时间处理投票
	time.Sleep(1 * time.Second)

	cs.updateStep(cstypes.RoundStepSlot)
	cs.enterApply()

	assert.Equal(t, 1, cs.state.UnCommitBlocks.Size(), "提案应该")
	newBlock := cs.state.BlockTree.GetLatestBlock()
	assert.Equal(t, proposal.Hash(), newBlock.Hash(), "提案应该为新的扩展分支")

	// 模拟生成第二个区块，evidence应该不为空
	proposal2 := cs.blockExec.CreateProposal(cs.state, types.LTime(1))
	addr2, _ := vals.GetByIndex(1)
	proposal2.ValidatorAddr = addr2
	assert.NoError(t, privs[1].SignProposal(cs.state.ChainID, proposal2), "生成proposal2的签名有问题")

	assert.Equal(t, 1, len(proposal2.Evidences), "proposal2应该包含proposal的support-quorum")

	// 测试提交第二个proposal 然后第一个proposal应该可以提交
	cs.updateSlot(types.LTime(1)) // 模拟更新slot
	cs.decideProposer()
	cs.setProposal(proposal2)

	assert.NotEqual(t, proposal2, proposal)

	// proposal的第一轮投票，应该全部添加成功
	for i := 0; i < len(privs); i++ {
		priv := privs[i]
		addr, _ := vals.GetByIndex(int32(i))
		vote := &types.Vote{
			Slot:             1,
			BlockHash:        proposal2.Hash(),
			Type:             types.SupportVote,
			Timestamp:        time.Now(),
			ValidatorAddress: addr,
			ValidatorIndex:   int32(i),
			Signature:        nil,
		}

		err := priv.SignVote(cs.state.ChainID, vote)
		assert.NoError(t, err, "sign vote failed.", "index", i, "error", err)
		cs.peerMsgQueue <- msgInfo{&VoteMessage{vote}, p2p.ID(fmt.Sprintf("%v", i))} // 模拟收到vote
	}

	// 给足够的时间处理投票
	time.Sleep(1 * time.Second)

	cs.updateStep(cstypes.RoundStepSlot)
	cs.enterApply()

	assert.Equal(t, 1, cs.state.UnCommitBlocks.Size(), "proposal应该提交")
	newBlock = cs.state.BlockTree.GetLatestBlock()
	assert.Equal(t, proposal2.Hash(), newBlock.Hash(), "提案应该为新的扩展分支")
}

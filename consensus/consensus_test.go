package consensus

import (
	cstypes "chainbft_demo/consensus/types"
	"chainbft_demo/crypto/bls"
	threshold2 "chainbft_demo/crypto/threshold"
	mempl "chainbft_demo/mempool"
	"chainbft_demo/privval"
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
	geneBlock := types.MakeGenesisBlock(chainID, time.Now())

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

func TestProposal(t *testing.T) {
	priv, vals, pub_val := newPrivAndValSet(4)
	_, val := vals.GetByIndex(0)
	txs := []types.Tx{}
	cs, clean := newConsensusState(priv[0], val, pub_val, vals, func(mem mempl.Mempool) {
		// 默认创建10条交易
		for i := 0; i < 10; i++ {
			//tx_ := []byte(fmt.Sprintf("tx=%v", i))
			tx := types.NewTx()
			txs = append(txs, tx)
			err := mem.CheckTx(tx, mempl.TxInfo{
				SenderID: mempl.UnknownPeerID,
			})
			assert.NoError(t, err, "add %vth tx into mempool failed.", i)
		}
	})

	defer clean()
	defer clean()
	cs.OnStart()
	cs.slotClock.OnStart()
	cs.updateStep(cstypes.RoundStepApply)

	var proposal *types.Proposal

	assert.NotPanics(t, func() {
		proposal = cs.defaultProposal()
	}, "propose failed.")

	// 暂停一会保证setproposal正常执行
	time.Sleep(1 * time.Second)

	assert.Equal(t, proposal, cs.Proposal, "proposal不一致，setProposal失败")
}

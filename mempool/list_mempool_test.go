package mempool

import (
	"chainbft_demo/types"
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"os"
	"testing"
)

type cleanupFunc func()

// ----- utility func -----

func newMempool() (*ListMempool, cleanupFunc) {
	return newMempoolWithConfig(cfg.ResetTestRoot("mempool_test"))
}

func newMempoolWithConfig(config *cfg.Config) (*ListMempool, cleanupFunc) {
	mempool := NewListMempool(config.Mempool)
	mempool.SetLogger(log.TestingLogger())
	return mempool, func() { os.RemoveAll(config.RootDir) }
}

// 随机生成一些交易，并对其checktx
func checkTxs(t *testing.T, mempool Mempool, count int, peerID uint16) types.Txs {
	txs := make(types.Txs, count)
	txinfo := TxInfo{
		SenderID: peerID,
	}
	for i := 0; i < count; i++ {
		txByte := make([]byte, 20)
		txs[i] = types.NormalTx(txByte)
		_, err := rand.Read(txByte)
		if err != nil {
			t.Error(err)
		}
		if err := mempool.CheckTx(txs[i], txinfo); err != nil {
			t.Fatalf("checkTx failed: %v while checking #%d tx", err, i)
		}
	}

	return txs
}

// ----- tests -----

func TestBasicMempool(t *testing.T) {
	mem, cleanup := newMempool()
	defer cleanup()

	test_Flush(t, mem)
	test_CheckTx(t, mem)
}

func test_Flush(t *testing.T, mem Mempool) {
	txs := checkTxs(t, mem, 1, UnknownPeerID)
	assert.Equal(t, 1, mem.Size())
	assert.Equal(t, int64(20), mem.TxsBytes())

	mem.Flush()
	assert.Equal(t, 0, mem.Size())
	assert.Equal(t, int64(0), mem.TxsBytes())

	_ = mem.CheckTx(txs[0], TxInfo{SenderID: UnknownPeerID})
	mem.Flush()
}

func test_CheckTx(t *testing.T, mem Mempool) {
	// 测试交易不能重复check & add
	// 暂时无法测试，没有应用cache
	//txs := checkTxs(t, mem, 1, UnknownPeerID)
	//err := mem.CheckTx(txs[0], TxInfo{SenderID: UnknownPeerID})
	//assert.NotNil(t, err, "the same tx can add to mempool.")
	//mem.Flush()

	{
		// 测试过大的交易不能加入到mempool中
		//tx := make([]byte, 1024*1024+1)
		//rand.Read(tx)
		//err := mem.CheckTx(tx, TxInfo{SenderID: UnknownPeerID})
		//if assert.Error(t, err){
		//	assert.Equal(ErrTxTooLarge{maxTxSize, testCase.len},)
		//}
	}

	tests := []struct {
		numTxsToCreate  int
		expectedTxNum   int
		expectedTxBytes int64
	}{
		{0, 0, 0},
		{1, 1, 20},
		{10, 10, 200},
	}

	for index, test := range tests {
		checkTxs(t, mem, test.numTxsToCreate, UnknownPeerID)
		assert.Equal(t, test.expectedTxNum, mem.Size(),
			"[memNum] Got %d, expected %d tc #%d",
			mem.Size(), test.expectedTxNum, index)
		assert.Equal(t, test.expectedTxBytes, mem.TxsBytes(),
			"[memBytes] Got %d, expected %d tc #%d",
			mem.TxsBytes(), test.expectedTxNum, index)
		mem.Flush()
	}
}

func TestReapTxs(t *testing.T) {
	mem, cleanup := newMempool()
	defer cleanup()

	// 确保生成的tx参数符合预设
	checkTxs(t, mem, 1, UnknownPeerID)
	tx := mem.TxsFront().Value.(*mempoolTx)
	require.Equal(t, 20, tx.tx.ComputeSize(), "len(tx) != 20 bytes")
	mem.Flush() // 清空mempool，开始测试

	tests := []struct {
		numTxsToCreate int
		maxBytes       int64
		expectedNumTxs int
	}{
		{20, -1, 20},
		{20, 400, 20},
		{20, 0, 0},
		{20, 150, 7},
		{20, 10, 0},
		{20, 200, 10},
	}

	for index, test := range tests {
		checkTxs(t, mem, test.numTxsToCreate, UnknownPeerID)
		txsFromReap := mem.ReapTxs(test.maxBytes)
		assert.Equal(t, test.expectedNumTxs, len(txsFromReap),
			"Got %v tx, expected %d, tc #%d",
			len(txsFromReap), test.expectedNumTxs, index)
		mem.Flush()
	}

}

func TestUpdate(t *testing.T) {
	mem, cleanup := newMempool()
	defer cleanup()

	//// Adds valid txs to the cache
	//{
	//	err := mem.Update(1, []types.Tx{[]byte{0x01}})
	//	require.NoError(t, err)
	//	err = mem.CheckTx([]byte{0x01}, TxInfo{})
	//	assert.NoError(t, err, "checkTxs not nil")
	//
	//}

	// 2. Removes valid txs from the mempool
	{
		err := mem.CheckTx(types.NormalTx{0x02}, TxInfo{})
		require.NoError(t, err)
		err = mem.Update(1, []types.Tx{types.NormalTx{0x02}})
		require.NoError(t, err)
		assert.Zero(t, mem.Size())
	}
}

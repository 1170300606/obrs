package store

import (
	"chainbft_demo/types"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/libs/log"
	tmdb "github.com/tendermint/tm-db"
	"math/rand"
	"testing"
)

func initaccount(db tmdb.DB, name string, c, s int) int {
	accountKey := GenKey(TableAccount, name)
	customID := Int2byte(rand.Int())
	db.Set(accountKey, customID)
	db.Set(GenKey(TableChecking, customID), Int2byte(c))
	db.Set(GenKey(TableSaving, customID), Int2byte(s))
	return Byte2int(customID)
}

func TestDbOpration(t *testing.T) {
	testStore := NewKVStore("test", "/tmp/chainbft/testdb", log.TestingLogger())
	testdb := testStore.kvDB
	name := "test2"

	customID := initaccount(testdb, name, 1000, 2000)
	t.Log(customID)

	s, err := testStore.getSavingBal(customID)
	assert.Nil(t, err)
	assert.Equal(t, 2000, s)

	c, err := testStore.getCheckingBal(customID)
	assert.Nil(t, err)
	assert.Equal(t, 1000, c)

	ite, _ := testdb.Iterator(nil, nil)
	for ite.Valid() {
		t.Log(string(ite.Key()), Byte2int(ite.Value()))
		ite.Next()
	}
}

func TestUtils(t *testing.T) {
	assert.Equal(t, 10, Byte2int(Int2byte(10)))
	t.Log(string(GenKey(TableAccount, "TOMMO")))
	t.Log(string(GenKey(TableSaving, 100)))
	t.Log(string(GenKey(TableChecking, 200)))
}

func TestSBDepositChecking(t *testing.T) {
	username := "Tom"
	testdb := NewKVStore("test", "/tmp/chainbft/testdb", log.TestingLogger())
	customID := initaccount(testdb.kvDB, username, 100, 10)

	tx := &types.Tx{
		TxType: types.SBDepositCheckingTx,
		Args:   []string{username, "1000"},
	}
	batch := testdb.kvDB.NewBatch()
	before, err := testdb.getCheckingBal(customID)

	assert.Equal(t, 100, before, "checking balance != 100")
	err = testdb.applySmallBank(batch, tx)
	batch.WriteSync()
	batch.Close()
	assert.Nil(t, err, "commit tx failed. err: ", err)
	after, err := testdb.getCheckingBal(customID)
	assert.Equal(t, 1100, after, "checking balance != 1100")
}

func TestSBTransationSaving(t *testing.T) {
	username := "saving"
	testdb := NewKVStore("test", "/tmp/chainbft/testdb", log.TestingLogger())
	customID := initaccount(testdb.kvDB, username, 100, 10)

	tx := &types.Tx{
		TxType: types.SBTransactionSavingTx,
		Args:   []string{username, "90"},
	}
	batch := testdb.kvDB.NewBatch()
	before, err := testdb.getSavingBal(customID)

	assert.Equal(t, 10, before, "saving balance != 100")
	err = testdb.applySmallBank(batch, tx)
	batch.WriteSync()
	batch.Close()
	assert.Nil(t, err, "commit tx failed. err: ", err)
	after, err := testdb.getSavingBal(customID)
	assert.Equal(t, 100, after, "saving balance != 1100")
}

func TestSBAmalgamate(t *testing.T) {
	username1 := "Tom"
	username2 := "Alice"
	testdb := NewKVStore("test", "/tmp/chainbft/testdb", log.TestingLogger())
	customID1 := initaccount(testdb.kvDB, username1, 100, 90)
	customID2 := initaccount(testdb.kvDB, username2, 10, 10)

	tx := &types.Tx{
		TxType: types.SBAmalgamateTx,
		Args:   []string{username1, username2},
	}

	batch := testdb.kvDB.NewBatch()
	before, err := testdb.getCheckingBal(customID2)

	assert.Equal(t, 10, before, "checking balance != 10")
	err = testdb.applySmallBank(batch, tx)
	batch.WriteSync()
	batch.Close()
	assert.Nil(t, err, "commit tx failed. err: ", err)
	after, err := testdb.getCheckingBal(customID2)
	assert.Equal(t, 200, after, "checking balance != 200")

	v, err := testdb.getSavingBal(customID1)
	assert.Equal(t, 0, v)
	v, err = testdb.getCheckingBal(customID1)
	assert.Equal(t, 0, v)
}

func TestWriteCheck(t *testing.T) {
	username := "Tom"
	testdb := NewKVStore("test", "/tmp/chainbft/testdb", log.TestingLogger())
	customID := initaccount(testdb.kvDB, username, 100, 90)

	{
		// total <= v
		tx := &types.Tx{
			TxType: types.SBWriteCheckingTx,
			Args:   []string{username, "80"},
		}

		batch := testdb.kvDB.NewBatch()
		before, err := testdb.getCheckingBal(customID)

		assert.Equal(t, 100, before, "checking balance != 100")
		err = testdb.applySmallBank(batch, tx)
		batch.WriteSync()
		batch.Close()
		assert.Nil(t, err, "commit tx failed. err: ", err)
		after, err := testdb.getCheckingBal(customID)
		assert.Equal(t, 20, after, "checking balance != 20")
	}

	{
		// total > v
		tx := &types.Tx{
			TxType: types.SBWriteCheckingTx,
			Args:   []string{username, "111"},
		}

		batch := testdb.kvDB.NewBatch()
		before, err := testdb.getCheckingBal(customID)

		assert.Equal(t, 20, before, "checking balance != 20")
		err = testdb.applySmallBank(batch, tx)
		batch.WriteSync()
		batch.Close()
		assert.Nil(t, err, "commit tx failed. err: ", err)
		after, err := testdb.getCheckingBal(customID)
		assert.Equal(t, -92, after, "checking balance != -92")
	}
}

package store

import (
	"bytes"
	"chainbft_demo/state"
	"chainbft_demo/types"
	"errors"
	"fmt"
	"github.com/tendermint/tendermint/libs/log"
	tmdb "github.com/tendermint/tm-db"
	leveldb "github.com/tendermint/tm-db/goleveldb"
	"math"
	"math/rand"
	"strconv"
	"sync/atomic"
)

const (
	tableAccount  = "accout"
	tableSaving   = "saving"
	tableChecking = "checking"
)

func NewKVStore(name, dir string, logger log.Logger) *KVStore {
	levelDB, err := leveldb.NewDB(name, dir)
	if err != nil {
		return nil
	}
	return NewKVStoreWithDB(levelDB, logger)
}

//
func NewKVStoreWithDB(kvdb tmdb.DB, logger log.Logger) *KVStore {
	return &KVStore{kvDB: kvdb, logger: logger, increasingID: rand.Int31n(math.MaxInt32)}
}

type KVStore struct {
	kvDB tmdb.DB

	logger log.Logger

	increasingID int32
}

// TODO
func (kv *KVStore) SaveState(s state.State) error {
	return nil
}

// implement SmallBank
// table definition：
// account table； key=account_{name}; value=string(int, string), which mean the customID
// saving table: key=saving_{customID}; value=string(int, string)
// checking table: key=checking_{customID}; value=string(int, string)
func (kv *KVStore) CommitBlock(s state.State, block *types.Block) ([]byte, error) {
	var batch tmdb.Batch = nil
	defer func() {
		if batch != nil {
			batch.Close()
		}
	}()

	batch = kv.kvDB.NewBatch()
	for _, tx := range block.Txs {
		err := kv.applySmallBank(batch, tx)
		if err != nil {
			kv.logger.Error("exec tx failed.", "err", err)
			kv.logger.Debug("exec tx failed.", "tx", tx, "err", err)
		}
	}
	batch.Write()
	if err := batch.Close(); err != nil {
		return nil, err
	}
	batch = nil
	// TODO generate result hash
	return []byte("result hash"), nil
}

func (kv *KVStore) applySmallBank(batch tmdb.Batch, tx types.Tx) error {
	sbtx := tx.(*types.SmallBankTx)
	switch sbtx.TxType {
	case types.SBBalanceTx:
		// query, continue
		break
	case types.SBDepositCheckingTx:
		name := sbtx.Args[0]
		value, err := strconv.Atoi(sbtx.Args[1])
		if err != nil {
			return err
		}

		customID, err := kv.getCustomID(name)
		if err != nil {
			return err
		}
		preBal, err := kv.getCheckingBal(customID)
		if err != nil {
			return err
		}
		return batch.Set(genKey(tableChecking, customID), int2byte(preBal+value))

	case types.SBTransactionSavingTx:
		name := sbtx.Args[0]
		value, err := strconv.Atoi(sbtx.Args[1])
		if err != nil {
			return err
		}
		customID, err := kv.getCustomID(name)
		if err != nil {
			return err
		}
		preBal, err := kv.getSavingBal(customID)
		if err != nil {
			return err
		}
		return batch.Set(genKey(tableSaving, customID), int2byte(preBal+value))

	case types.SBAmalgamateTx:
		n1 := sbtx.Args[0]
		n2 := sbtx.Args[1]

		customID1, err := kv.getCustomID(n1)
		if err != nil {
			return err
		}
		customID2, err := kv.getCustomID(n2)
		if err != nil {
			return err
		}
		c1Total, err := kv.getTotalBal(customID1)
		if err != nil {
			return err
		}

		preBal, err := kv.getCheckingBal(customID2)
		if err != nil {
			return err
		}

		batch.Set(genKey(tableChecking, customID2), int2byte(preBal+c1Total))

		batch.Set(genKey(tableSaving, customID1), int2byte(0))
		batch.Set(genKey(tableChecking, customID1), int2byte(0))

		return nil
	case types.SBWriteCheckingTx:
		name := sbtx.Args[0]
		value, err := strconv.Atoi(sbtx.Args[1])
		if err != nil {
			return err
		}
		customID, err := kv.getCustomID(name)
		if err != nil {
			return err
		}
		total, err := kv.getTotalBal(customID)
		if err != nil {
			return err
		}

		preBal, err := kv.getCheckingBal(customID)
		if err != nil {
			return err
		}

		if total <= value {
			preBal -= (value + 1)
		} else {
			preBal -= value
		}

		return batch.Set(genKey(tableChecking, customID), int2byte(preBal))

	default:
		return errors.New(string("wrong small bank tx type(" + sbtx.TxType + ")"))
	}
	return nil
}

func (kv *KVStore) getCustomID(name string) (int, error) {
	accountKey := genKey(tableAccount, name)
	customID, err := kv.kvDB.Get(accountKey)
	if err != nil {
		return -1, err
	}
	return byte2int(customID), nil
}

func (kv *KVStore) getCheckingBal(customID int) (int, error) {
	checkingkey := genKey(tableChecking, customID)
	bal, err := kv.kvDB.Get(checkingkey)
	if err != nil {
		return -1, err
	}
	return byte2int(bal), nil
}

func (kv *KVStore) getSavingBal(customID int) (int, error) {
	savingKey := genKey(tableSaving, customID)
	bal, err := kv.kvDB.Get(savingKey)
	if err != nil {
		return -1, err
	}
	return byte2int(bal), nil
}

func (kv *KVStore) getTotalBal(customID int) (int, error) {
	savingbal, err := kv.getSavingBal(customID)
	if err != nil {
		return 0, err
	}

	checkingbal, err := kv.getCheckingBal(customID)
	if err != nil {
		return 0, err
	}

	return savingbal + checkingbal, nil
}

func genKey(table string, primaryKey interface{}) []byte {
	buffer := new(bytes.Buffer)
	buffer.WriteString(table)
	switch primaryKey.(type) {
	case int:
		buffer.WriteString(strconv.Itoa(primaryKey.(int)))
	case string:
		buffer.WriteString(primaryKey.(string))
	case []byte:
		buffer.Write(primaryKey.([]byte))
	default:
		fmt.Println("adfasdfas", primaryKey)
	}
	return buffer.Bytes()
}

func byte2int(src []byte) int {
	v, _ := strconv.Atoi(string(src))
	return v
}

func int2byte(src int) []byte {
	return []byte(strconv.Itoa(src))
}

func (kv *KVStore) GetDB() tmdb.DB {
	return kv.kvDB
}

func (kv *KVStore) InitAccount(name string, saving int, checking int) error {
	accountKey := genKey(tableAccount, name)
	atomic.AddInt32(&kv.increasingID, 1)
	customID := int2byte(int(kv.increasingID - 1))
	kv.kvDB.Set(accountKey, customID)
	kv.kvDB.Set(genKey(tableChecking, customID), int2byte(checking))
	kv.kvDB.Set(genKey(tableSaving, customID), int2byte(saving))
	return nil
}

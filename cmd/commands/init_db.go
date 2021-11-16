package commands

import (
	"chainbft_demo/store"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	tmdb "github.com/tendermint/tm-db"
	leveldb "github.com/tendermint/tm-db/goleveldb"
)

var (
	accountSum int
	dbdir      string
)

func init() {
	InitDBCmd.Flags().IntVar(&accountSum, "account-sum", 100, "small bank account sum")
	InitDBCmd.Flags().StringVar(&dbdir, "dir", "/tmp/chainbft", "levelDB dir")
	InitDBCmd.MarkFlagRequired("account-sum")
}

var InitDBCmd = &cobra.Command{
	Use:     "init-db",
	Aliases: []string{"init_db", "initdb"},
	Short:   "initiate a test small bank database",
	RunE:    initDB,
}

func initAccount(db tmdb.DB, name string, id, saving, checking int) {
	accountKey := store.GenKey(store.TableAccount, name)
	customID := store.Int2byte(id)
	db.Set(accountKey, customID)
	db.Set(store.GenKey(store.TableChecking, customID), store.Int2byte(checking))
	db.Set(store.GenKey(store.TableSaving, customID), store.Int2byte(saving))
}

func initDB(cmd *cobra.Command, args []string) error {
	levelDB, err := leveldb.NewDB("smallbank", dbdir)
	if err != nil {
		return err
	}
	if accountSum < 0 {
		return errors.New("account sum must > 0")
	}

	for i := 0; i < accountSum; i++ {
		initAccount(levelDB, fmt.Sprintf("username%v", i+1), i+1, 200, 200)
	}
	//test(levelDB)
	levelDB.Close()
	return nil
}
func test(db tmdb.DB) {
	ite, _ := db.Iterator(nil, nil)
	for ite.Valid() {
		fmt.Println(string(ite.Key()), ite.Value())
		ite.Next()
	}

}

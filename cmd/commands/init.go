package commands

import (
	"chainbft_demo/crypto/bls"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"chainbft_demo/privval"
	"chainbft_demo/types"

	cfg "github.com/tendermint/tendermint/config"
	tmos "github.com/tendermint/tendermint/libs/os"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	"github.com/tendermint/tendermint/p2p"

	tmtime "github.com/tendermint/tendermint/types/time"
)

// InitFilesCmd initialises a fresh Tendermint Core instance.
var InitFilesCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Tendermint",
	RunE:  initFiles,
}

func init() {
}

func initFiles(cmd *cobra.Command, args []string) error {
	return initFilesWithConfig(config)
}

func initFilesWithConfig(config *cfg.Config) error {
	// private validator
	privValKeyFile := config.PrivValidatorKeyFile()

	var pv *privval.FilePV
	if tmos.FileExists(privValKeyFile) {
		pv = privval.LoadFilePV(privValKeyFile)
		logger.Info("Found private validator", "keyFile", privValKeyFile)
	} else {
		pv = privval.GenFilePVWithSeedAndIdx(privValKeyFile, thres, idx, seed)
		pv.Save()
		logger.Info("Generated private validator", "keyFile", privValKeyFile)
	}

	nodeKeyFile := config.NodeKeyFile()
	if tmos.FileExists(nodeKeyFile) {
		logger.Info("Found node key", "path", nodeKeyFile)
	} else {
		if _, err := p2p.LoadOrGenNodeKey(nodeKeyFile); err != nil {
			return err
		}
		logger.Info("Generated node key", "path", nodeKeyFile)
	}

	// public primary key
	pub_val := bls.GenPrimaryKeyWithSeed(seed)

	// genesis file
	genFile := config.GenesisFile()
	if tmos.FileExists(genFile) {
		logger.Info("Found genesis file", "path", genFile)
	} else {
		chainID := fmt.Sprintf("test-chain-%v", tmrand.Str(6))
		genBlock := types.MakeGenesisBlock(chainID, time.Now())
		quorum := types.Quorum{
			SLot:      0,
			BlockHash: genBlock.Hash(),
			Type:      types.SupportQuorum,
			Signature: []byte(""),
		}
		genDoc := types.GenesisDoc{
			ChainID:     fmt.Sprintf(chainID),
			GenesisTime: tmtime.Now(),
			InitialSlot: 0,
			PubValidator: types.GenesisValidator{
				Address: pub_val.Address(),
				PubKey:  pub_val,
				Name:    "",
			},
			SupportQuorum: quorum,
		}
		pubKey, err := pv.GetPubKey()
		if err != nil {
			return fmt.Errorf("can't get pubkey: %w", err)
		}
		genDoc.Validators = []types.GenesisValidator{{
			Address: pubKey.Address(),
			PubKey:  pubKey,
		}}

		if err := genDoc.SaveAs(genFile); err != nil {
			return err
		}
		logger.Info("Generated genesis file", "path", genFile)
	}

	return nil
}

package main

import (
	"fmt"
	cfg "github.com/tendermint/tendermint/config"
	"os"
	"path/filepath"
	//"chainbft_demo/types"
	cmd "chainbft_demo/cmd/commands"
	nm "chainbft_demo/node"
	"github.com/tendermint/tendermint/libs/cli"
)

func main() {
	//types.()

	cfg.DefaultTendermintDir = ".chain_bft"
	rootCmd := cmd.RootCmd

	rootCmd.AddCommand(
		//cmd.InitFilesCmd,
		cli.NewCompletionCmd(rootCmd, true),
	)

	// NOTE:
	// Users wishing to:
	//	* Use an external signer for their validators
	//	* Supply an in-proc abci app
	//	* Supply a genesis doc file from another source
	//	* Provide their own DB implementation
	// can copy this file and use something other than the
	// DefaultNewNode function
	nodeFunc := nm.DefaultNewNode

	// Create & start node
	rootCmd.AddCommand(
		cmd.GenNodeKeyCmd,
		cmd.GenValidatorCmd,
		cmd.ShowNodeIDCmd,
		cmd.ShowValidatorCmd,
		cmd.GenGenesisCmd,
		cmd.InitDBCmd,
		cmd.NewRunNodeCmd(nodeFunc),
	)
	cmd := cli.PrepareBaseCmd(rootCmd, "TM", os.ExpandEnv(filepath.Join("$HOME", cfg.DefaultTendermintDir)))

	if err := cmd.Execute(); err != nil {
		fmt.Println("error")
		panic(err)
	}

}

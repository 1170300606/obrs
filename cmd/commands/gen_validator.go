package commands

import (
	"fmt"
	tmos "github.com/tendermint/tendermint/libs/os"

	"chainbft_demo/privval"
	"github.com/spf13/cobra"
)

// GenValidatorCmd生成共识验证者的公私钥对
var GenValidatorCmd = &cobra.Command{
	Use:     "gen-validator",
	Aliases: []string{"gen_validator"},
	Args:    cobra.ArbitraryArgs,
	Short:   "Generate new validator keypair",
	PreRun:  deprecateSnakeCase,
	Run:     genValidator,
}

func init() {
	GenValidatorCmd.Flags().Int64Var(&seed, "seed", 1, "随机数种子，影响primary private key的生成")
	GenValidatorCmd.MarkFlagRequired("seed")
	GenValidatorCmd.Flags().Int64Var(&idx, "idx", 1, "共识节点的编号，影响节点private key的生成")
	GenValidatorCmd.MarkFlagRequired("idx")
	GenValidatorCmd.Flags().IntVar(&thres, "thres", 3, "门限签名阈值")
	GenValidatorCmd.MarkFlagRequired("thres")
}

func genValidator(cmd *cobra.Command, args []string) {
	privValKeyFile := config.PrivValidatorKeyFile()
	if tmos.FileExists(privValKeyFile) {
		logger.Info("Found private validator", "keyFile", privValKeyFile)
		return
	}

	pv := privval.GenFilePVWithSeedAndIdx(privValKeyFile, thres, idx, seed)
	jsbz, err := json.Marshal(pv)
	if err != nil {
		panic(err)
	}
	pv.Save()

	fmt.Printf(`%v
`, string(jsbz))
}

package commands

import (
	"chainbft_demo/crypto/bls"
	"chainbft_demo/crypto/threshold"
	"chainbft_demo/types"
	"fmt"
	"github.com/spf13/cobra"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	tmos "github.com/tendermint/tendermint/libs/os"
	tmtime "github.com/tendermint/tendermint/types/time"
	"time"
)

var GenGenesisCmd = &cobra.Command{
	Use:     "gen-genesis-block",
	Aliases: []string{"gen_genesis"},
	Short:   "Generate a genesis block for cluster",
	RunE:    genGenesisFile,
}

func init() {
	GenGenesisCmd.Flags().StringVar(&chainID, "chainID", "test-Chain", "链名，不指定则使用test-Chain")

	GenGenesisCmd.Flags().Int64Var(&seed, "seed", 1, "用来生成集群密钥的种子")
	GenGenesisCmd.MarkFlagRequired("seed")
	GenGenesisCmd.Flags().IntVar(&thres, "thres", 3, "门限签名阈值数")
	GenGenesisCmd.MarkFlagRequired("thres")
	GenGenesisCmd.Flags().IntVar(&cluster_count, "cluster-count", 1, "用来生成集群密钥的种子")
	GenGenesisCmd.MarkFlagRequired("cluster_count")
}

func genGenesisFile(cmd *cobra.Command, args []string) error {
	// genesis file
	genFile := config.GenesisFile()
	if tmos.FileExists(genFile) {
		logger.Info("Found genesis file", "path", genFile, ". exit.")
		return nil
	}

	primary_priv := bls.GenPrivKeyWithSeed(seed)
	primary_pub := primary_priv.PubKey()
	poly := threshold.Master(primary_priv, thres, seed)

	// 为每一个验证者生成公钥
	val_list := make([]types.GenesisValidator, cluster_count)
	for id := 1; id <= cluster_count; id++ { // 从1开始编号
		priv, err := poly.GetValue(int64(id))
		if err != nil {
			logger.Error(fmt.Sprintf("生成第%v个验证者的公钥失败", id), "err", err)
		}
		pub := priv.PubKey()

		val_list[id-1] = types.GenesisValidator{
			Address: pub.Address(),
			PubKey:  pub,
			Name:    fmt.Sprintf("validator-%v", id),
		}
	}

	genBlock := types.MakeGenesisBlock(chainID, time.Now())

	signature, err := primary_priv.Sign(types.ProposalSignBytes(chainID, &types.Proposal{genBlock}))
	if err != nil {
		logger.Error("生成quorum的签名出错", "err", err)
	}
	quorum := types.Quorum{
		SLot:      0,
		BlockHash: genBlock.Hash(),
		Type:      types.SupportQuorum,
		Signature: signature,
	}

	genDoc := types.GenesisDoc{
		ChainID:     fmt.Sprintf(chainID),
		GenesisTime: tmtime.Now(),
		InitialSlot: 0,
		Validators:  val_list,
		PubValidator: types.GenesisValidator{
			Address: primary_pub.Address(),
			PubKey:  primary_pub,
			Name:    "cluster-primary",
		},
		SupportQuorum: quorum,
		AppHash:       tmbytes.HexBytes("app hash"),
	}

	if err := genDoc.SaveAs(genFile); err != nil {
		return err
	}
	logger.Info("Generated genesis file", "path", genFile)

	return nil
}

package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	tmos "github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/p2p"
)

// GenNodeKeyCmd 生成节点用来通信连接用的公私钥，
// 同时还要负责生成节点的nodeID
var GenNodeKeyCmd = &cobra.Command{
	Use:     "gen-node-key",
	Aliases: []string{"gen_node_key"},
	Short:   "Generate a node key for this node and print its ID",
	PreRun:  deprecateSnakeCase,
	RunE:    genNodeKey,
}

func genNodeKey(cmd *cobra.Command, args []string) error {
	nodeKeyFile := config.NodeKeyFile()
	if tmos.FileExists(nodeKeyFile) {
		return fmt.Errorf("node key at %s already exists", nodeKeyFile)
	}

	nodeKey, err := p2p.LoadOrGenNodeKey(nodeKeyFile)
	if err != nil {
		return err
	}
	fmt.Println(nodeKey.ID())
	return nil
}

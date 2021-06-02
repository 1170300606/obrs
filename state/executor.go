package state

import (
	"chainbft_demo/types"
	"github.com/tendermint/tendermint/libs/log"
)

type BlockExecutor interface {
	CreateProposal() *types.Proposal

	CommitBlock(block *types.Block) error

	SetLogger(logger log.Logger)
}

package state

import "chainbft_demo/types"

type BlockExecutor interface {
	CreateProposal() *types.Proposal

	CommitBlock(block *types.Block) error
}

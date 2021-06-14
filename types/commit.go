package types

// Commit表示一个区块可以提交的证据，满足下面两个条件之一
// - 后代区块满足提交条件
// - 自己有support-quorum，同时后代也有support-quorum
type Commit struct {
	ParentCommit *Commit

	LocalQuorum *Quorum
	Witness     *Block
}

func (c *Commit) SetParentCommit(pc *Commit) {
	c.ParentCommit = pc
}

func (c *Commit) SetQuorum(q *Quorum) {
	c.LocalQuorum = q
}

func (c *Commit) SetWitness(w *Block) {
	c.Witness = w
}

// IsReady 检查一个区块的commit是否满足提交条件
func (c *Commit) IsReady() bool {
	root := c
	if c.ParentCommit != nil {
		root = c.ParentCommit
	}

	// 尝试检查自己是否有support-quorum和提供证据的区块处于pre-commit的block
	if root.LocalQuorum != nil && root.Witness != nil {
		return false
	}

	// 自己拥有support quorum
	if root.LocalQuorum.Type != SupportQuorum {
		return false
	}

	// 见证者已经提交或者处于precommit状态
	if root.Witness.BlockState == PrecommitBlock || root.Witness.BlockState == CommiitedBlock {
		return false
	}
	return true
}

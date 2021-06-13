package types

// Commit表示一个区块可以提交的证据，满足下面两个条件之一
// - 后代区块满足提交条件
// - 自己有support-quorum，同时后代也有support-quorum
type Commit struct {
	Child       *Commit
	SelfQuorum  *Quorum
	ChildQuorum *Quorum
}

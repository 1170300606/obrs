package state

func NewState() State {
	return State{}
}

// 有限确定状态机的一个状态节点
// 节点里应维护所有和区块相关的信息
// 理论上一次共识完成后一定会使状态节点发生
type State struct {
}

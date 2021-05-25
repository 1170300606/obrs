package node

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tendermint/tendermint/p2p"
)

func NewNodeInfo(ID p2p.ID, laddr string) (p2p.NodeInfo, error) {
	laddr = removeProtocolIfDefined(laddr)
	idAddr := fmt.Sprintf("%s@%s", ID, laddr)

	addr, err := p2p.NewNetAddressString(idAddr)
	if err != nil {
		return nil, err
	}
	return NodeInfo{
		Addr:    addr,
		Version: "1.0",
	}, nil

}

type NodeInfo struct {
	Addr *p2p.NetAddress

	Version string
}

func (info NodeInfo) ID() p2p.ID {
	return info.Addr.ID
}

func (info NodeInfo) NetAddress() (*p2p.NetAddress, error) {
	if info.Addr != nil {
		return info.Addr, nil
	}
	return nil, errors.New("node address is empty")
}

func (info NodeInfo) Validate() error {
	if info.Addr == nil {
		return errors.New("node address is empty")
	}

	if len(info.Version) > 0 && (strings.Trim(info.Version, "\t ") == "") {
		return fmt.Errorf("info.Version must be valid ASCII text without tabs, but got &v", info.Version)
	}

	return nil
}


// TODO p2p模块与tendermint的nodeinfo绑定紧密
func (info NodeInfo) CompatibleWith(otherInfo p2p.NodeInfo) error {
	//other, ok := otherInfo.(NodeInfo)
	//if !ok {
	//	return fmt.Errorf("wrong NodeInfo type. Expected DefaultNodeInfo, got %v", reflect.TypeOf(otherInfo))
	//}
	//
	//if other.Version != info.Version {
	//	return fmt.Errorf("wrong NodeInfo Version. Expected %v, but got %v", info.Version, other.Version)
	//}

	return nil
}

func removeProtocolIfDefined(addr string) string {
	if strings.Contains(addr, "://") {
		return strings.Split(addr, "://")[1]
	}
	return addr

}

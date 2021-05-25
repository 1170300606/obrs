package node

import (
	"chainbft_demo/consensus"
	"fmt"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/service"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/p2p/conn"
	"github.com/tendermint/tendermint/version"
	"strings"
)

type Provider func(*cfg.Config, log.Logger) (*Node, error)

type Node struct {
	service.BaseService

	// config
	config *cfg.Config

	// network
	transport *p2p.MultiplexTransport
	sw        *p2p.Switch // p2p connections
	//addrBook    pex.AddrBook // known peers
	nodeInfo p2p.NodeInfo
	nodeKey  *p2p.NodeKey // our node privkey
	//isListening bool

	// service
	testReactor *consensus.Reactor
}

type Option func(*Node)

func DefaultNewNode(config *cfg.Config, logger log.Logger) (*Node, error) {
	nodeKey, _ := p2p.LoadOrGenNodeKey(config.NodeKeyFile())

	return NewNode(config, nodeKey, logger)
}

func createTransport(
	nodeInfo p2p.NodeInfo,
	nodeKey *p2p.NodeKey,
) *p2p.MultiplexTransport {
	var (
		mConnConfig = conn.DefaultMConnConfig()
		transport   = p2p.NewMultiplexTransport(nodeInfo, *nodeKey, mConnConfig)
	)

	// Limit the number of incoming connections.
	//max := config.P2P.MaxNumInboundPeers + len(splitAndTrimEmpty(config.P2P.UnconditionalPeerIDs, ",", " "))
	//p2p.MultiplexTransportMaxIncomingConnections(max)(transport)

	return transport
}

func createSwitch(config *cfg.Config,
	transport p2p.Transport,
	consensusReactor *consensus.Reactor,
	nodeInfo p2p.NodeInfo,
	nodeKey *p2p.NodeKey,
	p2pLogger log.Logger) *p2p.Switch {

	sw := p2p.NewSwitch(
		config.P2P,
		transport,
	)
	sw.SetLogger(p2pLogger)
	sw.AddReactor("CONSENSUS", consensusReactor)

	sw.SetNodeInfo(nodeInfo)
	sw.SetNodeKey(nodeKey)

	p2pLogger.Info("P2P Node ID", "ID", nodeKey.ID(), "file", config.NodeKeyFile())
	return sw
}

func makeNodeInfo(
	config *cfg.Config,
	nodeKey *p2p.NodeKey,
) (p2p.NodeInfo, error) {
	txIndexerStatus := "off"

	nodeInfo := p2p.DefaultNodeInfo{
		ProtocolVersion: p2p.NewProtocolVersion(
			8, // global
			11,
			0,
		),
		DefaultNodeID: nodeKey.ID(),
		Network:       "test-chain-xNfEDp",
		Version:       version.TMCoreSemVer,
		Channels: []byte{
			consensus.TestChannel,
		},
		Moniker: config.Moniker,
		Other: p2p.DefaultNodeInfoOther{
			TxIndex:    txIndexerStatus,
			RPCAddress: config.RPC.ListenAddress,
		},
	}

	lAddr := config.P2P.ExternalAddress

	if lAddr == "" {
		lAddr = config.P2P.ListenAddress
	}

	nodeInfo.ListenAddr = lAddr

	err := nodeInfo.Validate()
	return nodeInfo, err
}

func NewNode(config *cfg.Config, nodekey *p2p.NodeKey, logger log.Logger, options ...Option) (*Node, error) {
	// create test reactor
	testReactor := consensus.NewReactor()
	testReactor.SetLogger(logger)
	testReactor.SetId(nodekey.ID())

	p2pLogger := logger.With("module", "p2p")

	// setup node identity
	//nodeinfo, err := NewNodeInfo(nodekey.ID(), config.P2P.ListenAddress)
	nodeinfo, err := makeNodeInfo(config, nodekey)
	if err != nil {
		return nil, err
	}

	// Setup Transport.
	transport := createTransport(nodeinfo, nodekey)

	// Setup Switch.
	sw := createSwitch(
		config, transport, testReactor, nodeinfo, nodekey, p2pLogger,
	)

	node := &Node{
		BaseService: service.BaseService{},
		config:      config,
		transport:   transport,
		sw:          sw,
		nodeInfo:    nodeinfo,
		nodeKey:     nodekey,
		testReactor: testReactor,
	}

	node.BaseService = *service.NewBaseService(logger, "Node", node)
	for _, option := range options {
		option(node)
	}

	return node, nil
}

func (n *Node) Switch() *p2p.Switch {
	return n.sw
}

func (n *Node) NodeInfo() p2p.NodeInfo {
	return n.nodeInfo
}

func (n *Node) OnStart() error {
	// start the transport
	addr, err := p2p.NewNetAddressString(p2p.IDAddressString(n.nodeKey.ID(), n.config.P2P.ListenAddress))
	if err != nil {
		return err
	}
	if err := n.transport.Listen(*addr); err != nil {
		return err
	}

	// start the Switch
	err = n.sw.Start()
	if err != nil {
		return err
	}

	// TODO 去连接其他节点
	n.Logger.Info("onstart", "peers", n.config.P2P.PersistentPeers)
	err = n.sw.DialPeersAsync(splitAndTrimEmpty(n.config.P2P.PersistentPeers, ",", " "))
	if err != nil {
		return fmt.Errorf("could not dial peers from persistent_peers field: %w", err)
	}

	return nil
}

func (n *Node) OnStop() {
	n.testReactor.OnStop()

	n.sw.Stop()

	n.transport.Close()

}

// splitAndTrimEmpty slices s into all subslices separated by sep and returns a
// slice of the string s with all leading and trailing Unicode code points
// contained in cutset removed. If sep is empty, SplitAndTrim splits after each
// UTF-8 sequence. First part is equivalent to strings.SplitN with a count of
// -1.  also filter out empty strings, only return non-empty strings.
func splitAndTrimEmpty(s, sep, cutset string) []string {
	if s == "" {
		return []string{}
	}

	spl := strings.Split(s, sep)
	nonEmptyStrings := make([]string, 0, len(spl))
	for i := 0; i < len(spl); i++ {
		element := strings.Trim(spl[i], cutset)
		if element != "" {
			nonEmptyStrings = append(nonEmptyStrings, element)
		}
	}
	return nonEmptyStrings
}

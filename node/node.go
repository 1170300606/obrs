package node

import (
	"chainbft_demo/consensus"
	"chainbft_demo/mempool"
	"chainbft_demo/rpc"
	"errors"
	"fmt"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/service"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/p2p/conn"
	"github.com/tendermint/tendermint/rpc/jsonrpc/server"
	"github.com/tendermint/tendermint/version"
	"net"
	"net/http"
	"strings"
)

func DefaultNewNode(config *cfg.Config, logger log.Logger) (*Node, error) {
	nodeKey, _ := p2p.LoadOrGenNodeKey(config.NodeKeyFile())

	return NewNode(config, nodeKey, logger)
}

func NewNode(config *cfg.Config, nodekey *p2p.NodeKey, logger log.Logger, options ...Option) (*Node, error) {
	// create services

	// create Consensus reactor
	conlogger := logger.With("module", "Consensus")
	conR := consensus.NewReactor()
	conR.SetLogger(conlogger)
	conR.SetId(nodekey.ID()) // TODO remove

	// create Mempool reactor
	memlogger := logger.With("module", "Mempool")
	mem := mempool.NewListMempool(config.Mempool, 0)
	memR := mempool.NewReactor(mem)
	memR.SetLogger(memlogger)

	// create p2p network
	p2pLogger := logger.With("module", "P2P")
	//setup node identity
	//nodeinfo, err := NewNodeInfo(nodekey.ID(), config.P2P.ListenAddress)
	nodeinfo, err := makeNodeInfo(config, nodekey)
	if err != nil {
		return nil, err
	}

	// Setup Transport.
	transport := createTransport(nodeinfo, nodekey)

	// Setup Switch.
	sw := createSwitch(
		config, transport,
		conR, memR,
		nodeinfo, nodekey, p2pLogger,
	)

	node := &Node{
		BaseService:      service.BaseService{},
		config:           config,
		transport:        transport,
		sw:               sw,
		nodeInfo:         nodeinfo,
		nodeKey:          nodekey,
		consensusReactor: conR,
		mempool:          mem,
		mempoolReactor:   memR,
	}

	node.BaseService = *service.NewBaseService(logger, "Node", node)
	for _, option := range options {
		option(node)
	}

	return node, nil
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
	conR *consensus.Reactor,
	memR *mempool.Reactor,
	nodeInfo p2p.NodeInfo,
	nodeKey *p2p.NodeKey,
	p2pLogger log.Logger) *p2p.Switch {

	sw := p2p.NewSwitch(
		config.P2P,
		transport,
	)
	sw.SetLogger(p2pLogger)

	sw.AddReactor("CONSENSUS", conR)
	sw.AddReactor("MEMPOOL", memR)

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
			mempool.MempoolChannel,
		}, // 必须在这里声明Channel才可以使用，为什么
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

// ----------------------------------
type Provider func(*cfg.Config, log.Logger) (*Node, error)

type Node struct {
	service.BaseService

	// config
	config *cfg.Config

	// network
	transport *p2p.MultiplexTransport
	sw        *p2p.Switch // p2p connections
	//addrBook    pex.AddrBook // known peers
	nodeInfo    p2p.NodeInfo
	nodeKey     *p2p.NodeKey // our node privkey
	isListening bool

	// service
	consensusReactor *consensus.Reactor
	mempool          mempool.Mempool
	mempoolReactor   *mempool.Reactor

	rpcListeners []net.Listener
}

type Option func(*Node)

func (n *Node) Switch() *p2p.Switch {
	return n.sw
}

func (n *Node) NodeInfo() p2p.NodeInfo {
	return n.nodeInfo
}

func (n *Node) OnStart() error {
	// start RPC server if listenAddr is not empty
	if n.config.RPC.ListenAddress != "" {
		RPClogger := n.Logger.With("module", "RPC")
		listeners, err := n.startRPC(n.mempool, RPClogger)
		if err != nil {
			return errors.New("start RPC server failed. reason: " + err.Error())
		}
		n.rpcListeners = listeners
	}

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

	n.isListening = true
	// 去连接其他节点
	n.Logger.Info("onstart", "peers", n.config.P2P.PersistentPeers)
	err = n.sw.DialPeersAsync(splitAndTrimEmpty(n.config.P2P.PersistentPeers, ",", " "))
	if err != nil {
		return fmt.Errorf("could not dial peers from persistent_peers field: %w", err)
	}

	return nil
}

func (n *Node) startRPC(mem mempool.Mempool, logger log.Logger) ([]net.Listener, error) {
	// setup rpc enviroment
	rpc.SetEnvironment(&rpc.Environment{Mempool: mem})

	config := server.DefaultConfig()

	listenAddrs := splitAndTrimEmpty(n.config.RPC.ListenAddress, ",", " ")

	listeners := make([]net.Listener, len(listenAddrs))

	for i, listenAddr := range listenAddrs {
		mux := http.NewServeMux()
		rpcLogger := n.Logger.With("module", "rpc-server")
		wmLogger := rpcLogger.With("protocol", "websocket")
		wm := server.NewWebsocketManager(
			rpc.Routes,
		)
		wm.SetLogger(wmLogger)
		mux.HandleFunc("/websocket", wm.WebsocketHandler)
		server.RegisterRPCFuncs(mux, rpc.Routes, rpcLogger)
		listener, err := server.Listen(
			listenAddr,
			config,
		)
		if err != nil {
			return nil, err
		}

		var rootHandler http.Handler = mux

		go func() {
			if err := server.Serve(
				listener,
				rootHandler,
				rpcLogger,
				config,
			); err != nil {
				n.Logger.Error("Error serving server", "err", err)
			}
		}()

		listeners[i] = listener
	}

	return listeners, nil
}

func (n *Node) OnStop() {
	n.consensusReactor.OnStop()

	n.sw.Stop()

	n.transport.Close()

	for _, l := range n.rpcListeners {
		if err := l.Close(); err != nil {
			n.Logger.Error("Error closing listener", "listener", l, "err", err)
		}
	}

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

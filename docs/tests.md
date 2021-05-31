# Tests Utility Function

## tendermint 

tendermint/config有生成用于测试的配置文件，但是validator只有一个
- tendermint/config/ResetTestRoot生成指定TestName的配置文件
- tendermint/config/ResetTestRootWithChainID生成指定TestName和ChainID的配置文件
> 测试流程先生成默认配置文件，然后再根据测试需要修改相应字段即可

-----

tendermint/p2p 
 - /mock/
   - Peer Peer接口mock实现，可以使用NewPeer创建一个指定Ip或者使用随机Ip的peer实例
   - Reactor Reactor接口Mock实现，可以使用NewReactor创建一个空转的Reactor
 - /test_utils.go 
   - AddrBookMock AddrBook接口的Mock实现，能简单保存ip
   - mockNodeInfo NodeInfo接口的Mock实现 - 能否在实际场景中使用该mock实现，目前并不需要tendermint.NodeInfo那么复杂的实现
   - CreateRandomPeer 创建一个随机地址的Peer实例。注意，这里返回的是真正实现的Peer的实例
   - CreateRoutableAddr 随机创建一个本机路由可达的地址
   - StartSwitches 启动一组switch
   - MakeSwitch 生成一个switch
   - Connect2Switches Switches全连接函数
   - MakeConnectedSwitches 返回n个启动好的switch，switch监听的端口由config指定。通过connect闭包指定switch的连接模式，如果是Connect2Switches则使用全连接
   

## ChainBFT

chainbft/mempool/list_mempool_test
 - newMempool生成默认测试配置文件的list mempool
 - newMempoolWithConfig生成指定配置的List Mempool
 
chainbft/mempool/mock/Mempool
 - 空的mempool实现。当其他模块测试不需要mempool功能时，可以使用该Mempool快速配置参数
 
chainbft/mempool/reactor_test
 - makeAndConnectReactors 生成n个Mempool，每个mempool加入到一个单独的Switch，Switch已经启动好
 - 

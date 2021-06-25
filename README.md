# chainBFT

链式BFT

## TODO

- mempool
    - mempoolTx引入状态 - 未打包交易、已打包交易但未提交、已提交交易（可以删除）。每次生成新proposal时，mempool从未打包的交易中，选出那些和已打包交易不冲突的交易返回
    - 打包时选择不和当前precommit区块冲突的交易
- 是否引入go-proto编码协议，目前使用std/json
    - 如果修改编码协议，要同步修改channeldesc的最大大小设置
- 分布式测试；初始化
    - 初始启动时，节点如何协商共同开启consensus
- consensus
    - reactor gossips：vote-gossip、block-gossip、slot-gossip
    - panic恢复 - 尽可能从Panic中恢复运行而不是结束协程，程序健壮性要求 后期工作 
    - 调试信息太少，增加日志内容
- 测试集
    - smallBank

## bug

- ctl+c 无法退出程序
- rpc发送交易有问题，需要思考如何点位
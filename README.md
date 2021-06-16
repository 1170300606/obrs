# chainBFT

链式BFT

## TODO

- 是否引入go-proto编码协议，目前使用std/json
    - 如果修改编码协议，要同步修改channeldesc的最大大小设置
- slot能否使用逻辑时钟设计协议
- 基础数据类型
    - block：ValidateBasic逻辑；
    - ~~block_set：存放block的集合，用来保存precommitblock、suspectblock。一般情况，使用者会频繁地更新数据数据，也有一定的读需求~~
    - ~~block_tree: 核心，用来存放blockhash构成的树形关系，在生成提案和验证区块时都需要该节点的参与；没有删除需求~~
    - ~~commit：是否必须待定，该结构为区块提交可以提交的证据；如何生成是一个问题~~
    - ~~quorum：明确的2t+1个投票合成的聚合签名，commit的核心组成成分~~
    - ~~slotVoteSet：在共识过程中用来临时保存收到的投票，投票以slot number为key聚集；每一轮的apply阶段来提取上一轮收到的所有vote~~
    - validatorSet&validator：用来保存验证者的身份信息和公钥信息，目前使用的是tendermint的设计，但是接口有冲突，后期需修改
- crypto
    - ~~引入BLS加密协议，目前使用的ed25519~~
    - 引入门限签名包，接口数据格式有变更
- consensus - doing
    - ~~reactor <=> consensus的消息传递方式：事件 or channe => channel l~~
    - ~~vote的相关逻辑 - 生成vote；收到vote以后添加到voteset；apply阶段提取votes尝试生成quorum~~
    - reactor gossips：vote-gossip、block-gossip、slot-gossip
    - panic恢复 - 尽可能从Panic中恢复运行而不是结束协程，程序健壮性要求 后期工作 
- blockstate
    - ~~决定哪些区块可以提交~~
    - 如何将内部数据库映射为merkle tree来存证
    - ~~如何根据当前的state，来验证一个区块是否合法~~
- mempool
    - mempoolTx引入状态 - 未打包交易、已打包交易但未提交、已提交交易（可以删除）。每次生成新proposal时，mempool从未打包的交易中，选出那些和已打包交易不冲突的交易返回
    - 新增接口LockTxs - 变更指定交易的状态，锁定
- *test：缺少test验证正确性
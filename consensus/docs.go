package consensus

//
//+-------------------------------------+
//v                                     |(Wait til `CommmitTime+timeoutCommit`)
//+-----------+                         +-----+-----+
//+----------> |  Propose  +--------------+          | NewHeight |
//|            +-----------+              |          +-----------+
//|                                       |                ^
//|(Else, after timeoutPrecommit)         v                |
//+-----+-----+                           +-----------+    |
//| Precommit |  <------------------------+  Prevote  |          |
//+-----+-----+                           +-----------+          |
//|(When +2/3 Precommits for block found)                  |
//v                                                        |
//+--------------------------------------------------------------------+
//|  Commit                                                            |
//|                                                                    |
//|  * Set CommitTime = now;                                           |
//|  * Wait for block, then stage/save/commit block;                   |
//+--------------------------------------------------------------------+

//ConsensusState - 共识状态机，负责共识逻辑的推进，main goroutine
//	- RoundState - 共识机内部的状态节点、该轮slot的区块、投票是否也保存在这里，因为是和该轮状态相关的东西
//	- State - 区块链的状态，维护除了error状态外的所有区块
//	- BlockExcutor - 负责执行可以提交的区块、或者和mempool打包区块
//  	- Store - 数据持久化
//		- Mempool - 交易缓存池
//	- PeerState - 保存邻居节点的状态，根据收到的消息来确定，consensus reactor负责创建PeerState，并且保存到Switch中

// Slot移动到Consensus包内，因为Slo需要生成Consensus需要的数据，但这些数据类型很多都是consensus内部的数据结构；其次slot并不需要单独一个reactor来通信，或者目前不需要

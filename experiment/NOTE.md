# 实验记录

## 11/18 大规模实验

唔，可算把整个实验流程跑起来了，通过实验发现一些问题。

一个是实现上的BUG：

- 网络采用广播算法，限制了测试规模，目前做到22个节点时就基本做不动了 - 邻居选择有问题 环形传播？
- 当节点规模变大时，节点很容易没有收到一个Precommit Block，造成整个链断了
- 聚合签名很不稳定，很容易就还原失败 - 
- 系统鲁棒性很差，在极限压测时，当系统因为网络抖动产生一个suspect block区块时，之后就很难再次恢复到正常的状态
- 网络层主动规避坏节点。如何判断一个节点是坏节点？


另外是实现过程中遇到的一些问题：

- 目前的配置修改不好用，是通过模块级别的全局变量设置的。只能尽量做到修改配置属性不用重新编译
- 影响TPS的因素有很多，block size、slot timeout。不妨固定slot timeout，可以通过txlatency判断是否到达系统的极限，如果txlatency刚好大于2*slot timeout，说明系统已经不能在两个slot内完成区块的commit，已经基本达到极限
- 如何通过日志判断节点能在正常共识？看两个：一个是看toCommitBlocks=[]不为空，另一个是看voteset is not empty，support vote size应该大于threshold，这样才能保证能够生成support Quorum

## 11/21

- 为什么总是偶数slot的区块suspect
- 共识失败的原因，大多数是没有收集足够多的投票，why？
- slot timeout和block size如何影响TPS，比如可能某个规模下有一个极限值，如何找打这个极限值？这个极限值应该是和slot timeout、blocksize无关。和原生PBFT的一个恒定加速比
- latency受slot timeout影响很大，latency和tps的关系？
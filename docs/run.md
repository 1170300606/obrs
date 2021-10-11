# HOW TO RUN

```shell script
# 生成创世块文件
./build/chain_bft gen-genesis-block --seed 100 --thres 3 --cluster-count 4

# 生成每个节点的配置文件
# 生成节点的node key文件
./build/chain_bft gen-node-key

# 生成节点的公私钥文件
./build/chain_bft gen-validator --idx 1 --seed 100 --thres 3
```
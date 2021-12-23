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


默认路径是~/.chain_bft,可以通过--home指定

1. 生成nodekey ./chain_bft gen-node-key
2. 生成validator-key ./chain_bft gen-validator --idx 1 --seed 1000 --thres 3
3. 生成创世文件 ./chain_bft gen-genesis-block --seed 100 --thres 3
4. 运行 ./chain_bft run

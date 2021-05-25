PROJECT="/Users/thegloves/workspace/chainbft_demo"

for i in {2..4} ; do
     nohup $PROJECT/build/bft start --home $PROJECT/test/node$i &
     sleep 1
done

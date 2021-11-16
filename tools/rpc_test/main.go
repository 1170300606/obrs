package main

import (
	"chainbft_demo/types"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	jsonrpc "github.com/tendermint/tendermint/rpc/jsonrpc/types"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	sendTimeout = 10 * time.Second
)

func connect(host string) (*websocket.Conn, *http.Response, error) {
	u := url.URL{Scheme: "ws", Host: host, Path: "/websocket"}
	return websocket.DefaultDialer.Dial(u.String(), nil)
}

func main() {
	c, resp, err := connect("127.0.0.1:26657")
	fmt.Println(resp, err)
	sbtx := types.SmallBankTx{
		TxType: types.SBTransactionSavingTx,
		Args:   []string{"Tom", "100"},
	}

	tx, _ := json.Marshal(sbtx)

	txHex := make([]byte, len(tx)*2)

	fmt.Println(len(tx))
	fmt.Println(hex.Encode(txHex, tx))

	paramsJSON, err := json.Marshal(map[string]interface{}{"tx": txHex})

	if err != nil {
		fmt.Printf("failed to encode params: %v\n", err)
		os.Exit(1)
	}
	rawParamsJSON := json.RawMessage(paramsJSON)

	c.SetWriteDeadline(time.Now().Add(sendTimeout))
	req := jsonrpc.RPCRequest{
		JSONRPC: "2.0",
		ID:      jsonrpc.JSONRPCStringID("tm-bench"),
		Method:  "broadcast_tx",
		Params:  rawParamsJSON,
	}
	fmt.Println(req)
	err = c.WriteJSON(req)

}

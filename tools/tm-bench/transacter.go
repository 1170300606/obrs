package main

import (
	"chainbft_demo/types"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	// it is ok to use math/rand here: we do not need a cryptographically secure random
	// number generator here and we can run the tests a bit faster
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"github.com/tendermint/tendermint/libs/log"
	jsonrpc "github.com/tendermint/tendermint/rpc/jsonrpc/types"
)

const (
	sendTimeout = 10 * time.Second
	// see https://github.com/tendermint/tendermint/blob/master/rpc/lib/server/handlers.go
	pingPeriod = (30 * 9 / 10) * time.Second
)

type transacter struct {
	Target            string
	Rate              int
	Connections       int
	BroadcastTxMethod string
	Accounts          int
	conns             []*websocket.Conn
	connsBroken       []bool
	startingWg        sync.WaitGroup
	endingWg          sync.WaitGroup
	stopped           bool

	logger log.Logger
}

func newTransacter(target string, connections, rate int, accounts int, broadcastTxMethod string) *transacter {
	return &transacter{
		Target:            target,
		Rate:              rate,
		Accounts:          accounts,
		Connections:       connections,
		BroadcastTxMethod: broadcastTxMethod,
		conns:             make([]*websocket.Conn, connections),
		connsBroken:       make([]bool, connections),
		logger:            log.NewNopLogger(),
	}
}

// SetLogger lets you set your own logger
func (t *transacter) SetLogger(l log.Logger) {
	t.logger = l
}

// Start opens N = `t.Connections` connections to the target and creates read
// and write goroutines for each connection.
func (t *transacter) Start() error {
	t.stopped = false

	rand.Seed(time.Now().Unix())

	for i := 0; i < t.Connections; i++ {
		c, _, err := connect(t.Target)
		if err != nil {
			return err
		}
		t.conns[i] = c
	}

	t.startingWg.Add(t.Connections)
	t.endingWg.Add(2 * t.Connections)
	for i := 0; i < t.Connections; i++ {
		go t.sendLoop(i)
		go t.receiveLoop(i)
	}

	t.startingWg.Wait()

	return nil
}

// Stop closes the connections.
func (t *transacter) Stop() {
	t.stopped = true
	t.endingWg.Wait()
	for _, c := range t.conns {
		c.Close()
	}
}

// receiveLoop reads messages from the connection (empty in case of
// `broadcast_tx_async`).
func (t *transacter) receiveLoop(connIndex int) {
	c := t.conns[connIndex]
	defer t.endingWg.Done()
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				t.logger.Error(
					fmt.Sprintf("failed to read response on conn %d", connIndex),
					"err",
					err,
				)
			}
			return
		}
		if t.stopped || t.connsBroken[connIndex] {
			return
		}
	}
}

// sendLoop generates transactions at a given rate.
func (t *transacter) sendLoop(connIndex int) {
	started := false
	// Close the starting waitgroup, in the event that this fails to start
	defer func() {
		if !started {
			t.startingWg.Done()
		}
	}()
	c := t.conns[connIndex]

	c.SetPingHandler(func(message string) error {
		err := c.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(sendTimeout))
		if err == websocket.ErrCloseSent {
			return nil
		} else if e, ok := err.(net.Error); ok && e.Temporary() {
			return nil
		}
		return err
	})

	logger := t.logger.With("addr", c.RemoteAddr())

	var txNumber = 0

	pingsTicker := time.NewTicker(pingPeriod)
	txsTicker := time.NewTicker(1 * time.Second)
	defer func() {
		pingsTicker.Stop()
		txsTicker.Stop()
		t.endingWg.Done()
	}()

	for {
		select {
		case <-txsTicker.C:
			startTime := time.Now()
			endTime := startTime.Add(time.Second)
			numTxSent := t.Rate
			if !started {
				t.startingWg.Done()
				started = true
			}

			now := time.Now()
			for i := 0; i < t.Rate; i++ {
				tx := generateTx(t.Accounts)
				txjson, _ := json.Marshal(tx)
				txHex := make([]byte, len(txjson)*2)
				hex.Encode(txHex, txjson)
				// update tx number of the tx, and the corresponding hex
				paramsJSON, err := json.Marshal(map[string]interface{}{"tx": txHex})
				if err != nil {
					fmt.Printf("failed to encode params: %v\n", err)
					os.Exit(1)
				}
				rawParamsJSON := json.RawMessage(paramsJSON)

				c.SetWriteDeadline(now.Add(sendTimeout))
				err = c.WriteJSON(jsonrpc.RPCRequest{
					JSONRPC: "2.0",
					ID:      jsonrpc.JSONRPCStringID("tm-bench"),
					Method:  t.BroadcastTxMethod,
					Params:  rawParamsJSON,
				})
				if err != nil {
					err = errors.Wrap(err,
						fmt.Sprintf("txs send failed on connection #%d", connIndex))
					t.connsBroken[connIndex] = true
					logger.Error(err.Error())
					return
				}

				// cache the time.Now() reads to save time.
				if i%5 == 0 {
					now = time.Now()
					if now.After(endTime) {
						// Plus one accounts for sending this tx
						numTxSent = i + 1
						break
					}
				}

				txNumber++
			}

			timeToSend := time.Since(startTime)
			logger.Info(fmt.Sprintf("sent %d transactions", numTxSent), "took", timeToSend)
			if timeToSend < 1*time.Second {
				sleepTime := time.Second - timeToSend
				logger.Debug(fmt.Sprintf("connection #%d is sleeping for %f seconds", connIndex, sleepTime.Seconds()))
				time.Sleep(sleepTime)
			}

		case <-pingsTicker.C:
			// go-rpc server closes the connection in the absence of pings
			c.SetWriteDeadline(time.Now().Add(sendTimeout))
			if err := c.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				err = errors.Wrap(err,
					fmt.Sprintf("failed to write ping message on conn #%d", connIndex))
				logger.Error(err.Error())
				t.connsBroken[connIndex] = true
			}
		}

		if t.stopped {
			// To cleanly close a connection, a client should send a close
			// frame and wait for the server to close the connection.
			c.SetWriteDeadline(time.Now().Add(sendTimeout))
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				err = errors.Wrap(err,
					fmt.Sprintf("failed to write close message on conn #%d", connIndex))
				logger.Error(err.Error())
				t.connsBroken[connIndex] = true
			}

			return
		}
	}
}

func connect(host string) (*websocket.Conn, *http.Response, error) {
	u := url.URL{Scheme: "ws", Host: host, Path: "/websocket"}
	return websocket.DefaultDialer.Dial(u.String(), nil)
}

func generateTx(accounts int) *types.Tx {
	tx := new(types.Tx)

	switch rand.Intn(4) {
	case 0:
		tx.TxType = types.SBTransactionSavingTx
		username := fmt.Sprintf("username%v", rand.Intn(accounts)+1)
		v := rand.Intn(200)
		tx.Args = []string{username, strconv.Itoa(v)}
	case 1:
		tx.TxType = types.SBWriteCheckingTx
		username := fmt.Sprintf("username%v", rand.Intn(accounts)+1)
		v := rand.Intn(200)
		tx.Args = []string{username, strconv.Itoa(v)}
	case 2:
		tx.TxType = types.SBAmalgamateTx
		username1 := fmt.Sprintf("username%v", rand.Intn(accounts)+1)
		var username2 = username1
		for username2 != username1 {
			username2 = fmt.Sprintf("username%v", rand.Intn(accounts)+1)
		}
		tx.Args = []string{username1, username2}
	case 3:
		tx.TxType = types.SBDepositCheckingTx
		username := fmt.Sprintf("username%v", rand.Intn(accounts)+1)
		v := rand.Intn(200)
		tx.Args = []string{username, strconv.Itoa(v)}
	}

	return tx
}

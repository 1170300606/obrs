package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/log/term"

	"github.com/tendermint/tendermint/libs/log"
	tmrpc "github.com/tendermint/tendermint/rpc/client"
)

var logger = log.NewNopLogger()

func main() {
	var durationInt, txsRate, connections int
	var verbose bool
	var accounts int
	var outputFormat, broadcastTxMethod string
	broadcastTxMethod = "broadcast_tx"

	flagSet := flag.NewFlagSet("tm-bench", flag.ExitOnError)
	flagSet.IntVar(&connections, "c", 1, "Connections to keep open per endpoint")
	flagSet.IntVar(&durationInt, "T", 10, "Exit after the specified amount of time in seconds")
	flagSet.IntVar(&txsRate, "r", 1000, "Txs per second to send in a connection")
	flagSet.IntVar(&accounts, "a", 1000, "small bank accounts sum")
	flagSet.StringVar(&outputFormat, "output-format", "plain", "Output format: plain or json")
	flagSet.BoolVar(&verbose, "v", false, "Verbose output")

	flagSet.Usage = func() {
		fmt.Println(`Tendermint blockchain benchmarking tool.

Usage:
	tm-bench [-c 1] [-T 10] [-r 1000] [-s 250] [endpoints] [-output-format <plain|json> [-broadcast-tx-method <async|sync|commit>]]

Examples:
	tm-bench localhost:26657`)
		fmt.Println("Flags:")
		flagSet.PrintDefaults()
	}

	flagSet.Parse(os.Args[1:])

	if flagSet.NArg() == 0 {
		flagSet.Usage()
		os.Exit(1)
	}

	if verbose {
		if outputFormat == "json" {
			printErrorAndExit("Verbose mode not supported with json output.")
		}
		// Color errors red
		colorFn := func(keyvals ...interface{}) term.FgBgColor {
			for i := 1; i < len(keyvals); i += 2 {
				if _, ok := keyvals[i].(error); ok {
					return term.FgBgColor{Fg: term.White, Bg: term.Red}
				}
			}
			return term.FgBgColor{}
		}
		logger = log.NewTMLoggerWithColorFn(log.NewSyncWriter(os.Stdout), colorFn)

		fmt.Printf("Running %ds test @ %s\n", durationInt, flagSet.Arg(0))
	}

	var (
		endpoints           = strings.Split(flagSet.Arg(0), ",")
		initialHeight int64 = 1
	)
	logger.Info("Latest block height", "h", initialHeight)

	transacters := startTransacters(
		endpoints,
		connections,
		txsRate,
		accounts,
		broadcastTxMethod,
	)

	// Wait until transacters have begun until we get the start time.
	timeStart := time.Now()
	logger.Info("Time last transacter started", "t", timeStart)

	duration := time.Duration(durationInt) * time.Second

	timeEnd := timeStart.Add(duration)
	logger.Info("End time for calculation", "t", timeEnd)

	<-time.After(duration)
	for i, t := range transacters {
		t.Stop()
		numCrashes := countCrashes(t.connsBroken)
		if numCrashes != 0 {
			fmt.Printf("%d connections crashed on transacter #%d\n", numCrashes, i)
		}
	}

	logger.Debug("Time all transacters stopped", "t", time.Now())

	stats, err := calculateStatistics(
		nil,
		initialHeight,
		timeStart,
		durationInt,
	)
	if err != nil {
		printErrorAndExit(err.Error())
	}

	printStatistics(stats, outputFormat)
}

// TODO
func latestBlockHeight(client tmrpc.Client) int64 {
	return 1
	//status, err := client.Status()
	//if err != nil {
	//	fmt.Fprintln(os.Stderr, err)
	//	os.Exit(1)
	//}
	//return status.SyncInfo.LatestBlockHeight
}

func countCrashes(crashes []bool) int {
	count := 0
	for i := 0; i < len(crashes); i++ {
		if crashes[i] {
			count++
		}
	}
	return count
}

func startTransacters(
	endpoints []string,
	connections,
	txsRate int,
	accounts int,
	broadcastTxMethod string,
) []*transacter {
	transacters := make([]*transacter, len(endpoints))

	wg := sync.WaitGroup{}
	wg.Add(len(endpoints))
	for i, e := range endpoints {
		t := newTransacter(e, connections, txsRate, accounts, broadcastTxMethod)
		t.SetLogger(logger)
		go func(i int) {
			defer wg.Done()
			if err := t.Start(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			transacters[i] = t
		}(i)
	}
	wg.Wait()

	return transacters
}

func printErrorAndExit(err string) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

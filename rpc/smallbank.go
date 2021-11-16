package rpc

import (
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
	"strconv"
)

func InitSmallBankAccount(ctx *rpctypes.Context, name, saving, checking string) error {
	s, err := strconv.Atoi(saving)
	if err != nil {
		return err
	}
	c, err := strconv.Atoi(checking)
	if err != nil {
		return err
	}
	return env.Store.InitAccount(name, s, c)
}

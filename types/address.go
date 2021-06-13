package types

import (
	"bytes"
	"github.com/tendermint/tendermint/crypto"
)

type Address crypto.Address

func GetAddress(key crypto.PubKey) Address {
	return Address(key.Address())
}

func (addr Address) Equal(other Address) bool {
	if addr == nil || other == nil {
		return false
	}
	return bytes.Equal(crypto.Address(addr), crypto.Address(other))
}

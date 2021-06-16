package types

import (
	"github.com/tendermint/tendermint/crypto"
)

type Address = crypto.Address

func GetAddress(key crypto.PubKey) Address {
	return Address(key.Address())
}

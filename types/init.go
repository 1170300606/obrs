package types

import tmjson "github.com/tendermint/tendermint/libs/json"

func init() {
	tmjson.RegisterType(SmallBankTx{}, "chainBFT/SmallBankTx")
}

// fork from github.com/tendermint/tendermint/types/validator.go
package types

import (
	"errors"
	"fmt"
	"github.com/tendermint/tendermint/crypto"
)

// Volatile state for each Validator
// NOTE: The ProposerPriority is not included in PrivVal.Hash();
// make sure to update that method if changes are made here
type Validator struct {
	Address Address       `json:"address"`
	PubKey  crypto.PubKey `json:"pub_key"`
}

// NewValidator returns a new validator with the given pubkey and voting power.
func NewValidator(pubKey crypto.PubKey) *Validator {
	return &Validator{
		Address: pubKey.Address(),
		PubKey:  pubKey,
	}
}

// ValidateBasic performs basic validation.
func (v *Validator) ValidateBasic() error {
	if v == nil {
		return errors.New("nil validator")
	}
	if v.PubKey == nil {
		return errors.New("validator does not have a public key")
	}

	if len(v.Address) != crypto.AddressSize {
		return fmt.Errorf("validator address is the wrong size: %v", v.Address)
	}

	return nil
}

// Creates a new copy of the validator so we can mutate ProposerPriority.
// Panics if the validator is nil.
func (v *Validator) Copy() *Validator {
	vCopy := *v
	return &vCopy
}

// String returns a string representation of String.
//
// 1. address
// 2. public key
// 3. voting power
// 4. proposer priorityvalida
func (v *Validator) String() string {
	if v == nil {
		return "nil-PrivVal"
	}
	return fmt.Sprintf("PrivVal{%v %v}",
		v.Address,
		v.PubKey)
}

// Bytes computes the unique encoding of a validator with a given voting power.
// These are the bytes that gets hashed in consensus. It excludes address
// as its redundant with the pubkey. This also excludes ProposerPriority
// which changes every round.
func (v *Validator) Bytes() []byte {

	pk, err := json.Marshal(v.PubKey)
	if err != nil {
		panic(err)
	}

	return pk
}

//----------------------------------------
// RandValidator

// RandValidator returns a randomized validator, useful for testing.
// UNSTABLE
func RandValidator() (*Validator, PrivValidator) {
	privVal := NewMockPV()

	pubKey, err := privVal.GetPubKey()
	if err != nil {
		panic(fmt.Errorf("could not retrieve pubkey %w", err))
	}
	val := NewValidator(pubKey)
	return val, privVal
}

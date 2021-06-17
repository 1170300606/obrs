// fork from github.com/tendermint/tendermint/types/validator_set.go
package types

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/tendermint/tendermint/crypto/merkle"
	"sort"
	"strings"
)

// ValidatorSet represent a set of *Validator at a given height.
//
// The validators can be fetched by address or index.
// The index is in order of .VotingPower, so the indices are fixed for all
// rounds of a given blockchain height - ie. the validators are sorted by their
// voting power (descending). Secondary index - .Address (ascending).
//
// On the other hand, the .ProposerPriority of each validator and the
// designated .GetProposer() of a set changes every round, upon calling
// .IncrementProposerPriority().
//
// NOTE: Not goroutine-safe.
// NOTE: All get/set to validators should copy the value for safety.
type ValidatorSet struct {
	// NOTE: persisted via reflect, must be exported.
	Validators []*Validator `json:"validators"`
}

// NewValidatorSet initializes a ValidatorSet by copying over the values from
// `valz`, a list of Validators. If valz is nil or empty, the new ValidatorSet
// will have an empty list of Validators.
//
// The addresses of validators in `valz` must be unique otherwise the function
// panics.
//
// Note the validator set size has an implied limit equal to that of the
// MaxVotesCount - commits by a validator set larger than this will fail
// validation.
func NewValidatorSet(valz []*Validator) *ValidatorSet {
	vals := &ValidatorSet{}
	vals.Validators = make([]*Validator, 0, len(valz))

	for _, val := range valz {
		vals.Validators = append(vals.Validators, val)
	}

	return vals
}

func (vals *ValidatorSet) ValidateBasic() error {
	if vals.IsNilOrEmpty() {
		return errors.New("validator set is nil or empty")
	}

	for idx, val := range vals.Validators {
		if err := val.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid validator #%d: %w", idx, err)
		}
	}

	return nil
}

// IsNilOrEmpty returns true if validator set is nil or empty.
func (vals *ValidatorSet) IsNilOrEmpty() bool {
	return vals == nil || len(vals.Validators) == 0
}

// Makes a copy of the validator list.
func validatorListCopy(valsList []*Validator) []*Validator {
	if valsList == nil {
		return nil
	}
	valsCopy := make([]*Validator, len(valsList))
	for i, val := range valsList {
		valsCopy[i] = val.Copy()
	}
	return valsCopy
}

// Copy each validator into a new ValidatorSet.
func (vals *ValidatorSet) Copy() *ValidatorSet {
	return &ValidatorSet{
		Validators: validatorListCopy(vals.Validators),
	}
}

// HasAddress returns true if address given is in the validator set, false -
// otherwise.
func (vals *ValidatorSet) HasAddress(address []byte) bool {
	for _, val := range vals.Validators {
		if bytes.Equal(val.Address, address) {
			return true
		}
	}
	return false
}

// GetByAddress returns an index of the validator with address and validator
// itself (copy) if found. Otherwise, -1 and nil are returned.
func (vals *ValidatorSet) GetByAddress(address []byte) (index int32, val *Validator) {
	for idx, val := range vals.Validators {
		if bytes.Equal(val.Address, address) {
			return int32(idx), val.Copy()
		}
	}
	return -1, nil
}

// GetByIndex returns the validator's address and validator itself (copy) by
// index.
// It returns nil values if index is less than 0 or greater or equal to
// len(ValidatorSet.Validators).
func (vals *ValidatorSet) GetByIndex(index int32) (address []byte, val *Validator) {
	if index < 0 || int(index) >= len(vals.Validators) {
		return nil, nil
	}
	val = vals.Validators[index]
	return val.Address, val.Copy()
}

// Size returns the length of the validator set.
func (vals *ValidatorSet) Size() int {
	return len(vals.Validators)
}

// GetProposer returns the current proposer. If the validator set is empty, nil
// is returned.
func (vals *ValidatorSet) GetProposer(current LTime) (proposer *Validator) {
	if len(vals.Validators) == 0 {
		return nil
	}
	idx := current.Mod(len(vals.Validators))

	return vals.Validators[idx].Copy()
}

// Hash returns the Merkle root hash build using validators (as leaves) in the
// set.
func (vals *ValidatorSet) Hash() []byte {
	bzs := make([][]byte, len(vals.Validators))
	for i, val := range vals.Validators {
		bzs[i] = val.Bytes()
	}
	return merkle.HashFromByteSlices(bzs)
}

// Iterate will run the given function over the set.
func (vals *ValidatorSet) Iterate(fn func(index int, val *Validator) bool) {
	for i, val := range vals.Validators {
		stop := fn(i, val.Copy())
		if stop {
			break
		}
	}
}

//-----------------

// IsErrNotEnoughVotingPowerSigned returns true if err is
// ErrNotEnoughVotingPowerSigned.
func IsErrNotEnoughVotingPowerSigned(err error) bool {
	return errors.As(err, &ErrNotEnoughVotingPowerSigned{})
}

// ErrNotEnoughVotingPowerSigned is returned when not enough validators signed
// a commit.
type ErrNotEnoughVotingPowerSigned struct {
	Got    int64
	Needed int64
}

func (e ErrNotEnoughVotingPowerSigned) Error() string {
	return fmt.Sprintf("invalid commit -- insufficient voting power: got %d, needed more than %d", e.Got, e.Needed)
}

//----------------

// String returns a string representation of ValidatorSet.
//
// See StringIndented.
func (vals *ValidatorSet) String() string {
	return vals.StringIndented("")
}

// StringIndented returns an intended String.
//
// See Validator#String.
func (vals *ValidatorSet) StringIndented(indent string) string {
	if vals == nil {
		return "nil-ValidatorSet"
	}
	var valStrings []string
	vals.Iterate(func(index int, val *Validator) bool {
		valStrings = append(valStrings, val.String())
		return false
	})
	return fmt.Sprintf(`ValidatorSet{
%s  Validators:
%s    %v
%s}`,
		indent,
		indent, strings.Join(valStrings, "\n"+indent+"    "),
		indent)

}

//----------------------------------------

// RandValidatorSet returns a randomized validator set (size: +numValidators+),
// where each validator has a voting power of +votingPower+.
//
// EXPOSED FOR TESTING.
func RandValidatorSet(numValidators int) (*ValidatorSet, []PrivValidator) {
	var (
		valz           = make([]*Validator, numValidators)
		privValidators = make([]PrivValidator, numValidators)
	)

	for i := 0; i < numValidators; i++ {
		val, privValidator := RandValidator()
		valz[i] = val
		privValidators[i] = privValidator
	}

	sort.Sort(PrivValidatorsByAddress(privValidators))

	return NewValidatorSet(valz), privValidators
}

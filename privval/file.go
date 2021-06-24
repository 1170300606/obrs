package privval

import (
	"chainbft_demo/crypto/bls"
	"chainbft_demo/crypto/threshold"
	"chainbft_demo/types"
	"fmt"
	"github.com/tendermint/tendermint/crypto"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmos "github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/libs/tempfile"
	"io/ioutil"
)

//-------------------------------------------------------------------------------

// FilePVKey stores the immutable part of PrivValidator.
type FilePVKey struct {
	Address types.Address  `json:"address"`
	PubKey  crypto.PubKey  `json:"pub_key"`
	PrivKey crypto.PrivKey `json:"priv_key"`

	filePath string
}

// Save persists the FilePVKey to its filePath.
func (pvKey FilePVKey) Save() {
	outFile := pvKey.filePath
	if outFile == "" {
		panic("cannot save PrivValidator key: filePath not set")
	}

	jsonBytes, err := tmjson.MarshalIndent(pvKey, "", "  ")
	if err != nil {
		panic(err)
	}
	err = tempfile.WriteFileAtomic(outFile, jsonBytes, 0600)
	if err != nil {
		panic(err)
	}

}

//-------------------------------------------------------------------------------

// FilePV implements PrivValidator using data persisted to disk
// to prevent double signing.
// NOTE: the directories containing pv.Key.filePath and pv.LastSignState.filePath must already exist.
// It includes the LastSignature and LastSignBytes so we don't lose the signature
// if the process crashes after signing but before the resulting consensus message is processed.
type FilePV struct {
	Key FilePVKey
}

// NewFilePV generates a new validator from the given key and paths.
func NewFilePV(privKey crypto.PrivKey, keyFilePath string) *FilePV {
	return &FilePV{
		Key: FilePVKey{
			Address:  privKey.PubKey().Address(),
			PubKey:   privKey.PubKey(),
			PrivKey:  privKey,
			filePath: keyFilePath,
		},
	}
}

func GenFilePVWithSeedAndIdx(keyFilePath string, threshold_val int, idx, seed int64) *FilePV {
	// 集群主私钥
	primary := bls.GenPrivKeyWithSeed(seed)

	// 根据主私钥生成的随机多项式 用来生成节点的私钥
	poly := threshold.Master(primary, threshold_val, seed)

	// 节点自己的私钥
	priv, err := poly.GetValue(idx)
	if err != nil {
		panic(err)
	}
	return NewFilePV(priv, keyFilePath)
}

// GenFilePV generates a new validator with randomly generated private key
// and sets the filePaths, but does not call Save().
func GenFilePV(keyFilePath string) *FilePV {
	return NewFilePV(bls.GenPrivKey(), keyFilePath)
}

// LoadFilePV loads a FilePV from the filePaths.  The FilePV handles double
// signing prevention by persisting data to the stateFilePath.  If either file path
// does not exist, the program will exit.
func LoadFilePV(keyFilePath string) *FilePV {
	return loadFilePV(keyFilePath)
}

// LoadFilePVEmptyState loads a FilePV from the given keyFilePath, with an empty LastSignState.
// If the keyFilePath does not exist, the program will exit.
func LoadFilePVEmptyState(keyFilePath string) *FilePV {
	return loadFilePV(keyFilePath)
}

// If loadState is true, we load from the stateFilePath. Otherwise, we use an empty LastSignState.
func loadFilePV(keyFilePath string) *FilePV {
	keyJSONBytes, err := ioutil.ReadFile(keyFilePath)
	if err != nil {
		tmos.Exit(err.Error())
	}
	pvKey := FilePVKey{}
	err = tmjson.Unmarshal(keyJSONBytes, &pvKey)
	if err != nil {
		tmos.Exit(fmt.Sprintf("Error reading PrivValidator key from %v: %v\n", keyFilePath, err))
	}

	// overwrite pubkey and address for convenience
	pvKey.PubKey = pvKey.PrivKey.PubKey()
	pvKey.Address = pvKey.PubKey.Address()
	pvKey.filePath = keyFilePath

	return &FilePV{
		Key: pvKey,
	}
}

// LoadOrGenFilePV loads a FilePV from the given filePaths
// or else generates a new one and saves it to the filePaths.
func LoadOrGenFilePV(keyFilePath, stateFilePath string) *FilePV {
	var pv *FilePV
	if tmos.FileExists(keyFilePath) {
		pv = LoadFilePV(keyFilePath)
	} else {
		pv = GenFilePV(keyFilePath)
		pv.Save()
	}
	return pv
}

// GetAddress returns the address of the validator.
// Implements PrivValidator.
func (pv *FilePV) GetAddress() types.Address {
	return pv.Key.Address
}

// GetPubKey returns the public key of the validator.
// Implements PrivValidator.
func (pv *FilePV) GetPubKey() (crypto.PubKey, error) {
	return pv.Key.PubKey, nil
}

// SignVote signs a canonical representation of the vote, along with the
// chainID. Implements PrivValidator.
func (pv *FilePV) SignVote(chainID string, vote *types.Vote) error {
	if err := pv.signVote(chainID, vote); err != nil {
		return fmt.Errorf("error signing vote: %v", err)
	}
	return nil
}

// SignProposal signs a canonical representation of the proposal, along with
// the chainID. Implements PrivValidator.
func (pv *FilePV) SignProposal(chainID string, proposal *types.Proposal) error {
	if err := pv.signProposal(chainID, proposal); err != nil {
		return fmt.Errorf("error signing proposal: %v", err)
	}
	return nil
}

// Save persists the FilePV to disk.
func (pv *FilePV) Save() {
	pv.Key.Save()
}

// Reset resets all fields in the FilePV.
// NOTE: Unsafe!
func (pv *FilePV) Reset() {
	pv.Save()
}

// String returns a string representation of the FilePV.
func (pv *FilePV) String() string {
	return fmt.Sprintf(
		"PrivValidator{%v}",
		pv.GetAddress(),
	)
}

//------------------------------------------------------------------------------------

// signVote checks if the vote is good to sign and sets the vote signature.
// It may need to set the timestamp as well if the vote is otherwise the same as
// a previously signed vote (ie. we crashed after signing but before the vote hit the WAL).
func (pv *FilePV) signVote(chainID string, vote *types.Vote) error {

	signBytes := types.VoteSignBytes(chainID, vote)

	// It passed the checks. Sign the vote
	sig, err := pv.Key.PrivKey.Sign(signBytes)
	if err != nil {
		return err
	}
	vote.Signature = sig
	return nil
}

// signProposal checks if the proposal is good to sign and sets the proposal signature.
// It may need to set the timestamp as well if the proposal is otherwise the same as
// a previously signed proposal ie. we crashed after signing but before the proposal hit the WAL).
func (pv *FilePV) signProposal(chainID string, proposal *types.Proposal) error {

	signBytes := types.ProposalSignBytes(chainID, proposal)

	// It passed the checks. Sign the proposal
	sig, err := pv.Key.PrivKey.Sign(signBytes)
	if err != nil {
		return err
	}

	proposal.Signature = sig
	return nil
}

package bls

import (
	"bytes"
	"crypto/subtle"
	"errors"
	"fmt"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	"go.dedis.ch/kyber/v3/sign/bls"
	"go.dedis.ch/kyber/v3/util/random"
	"io"
	//"golang.org/x/exp/rand"
	"math/rand"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmjson "github.com/tendermint/tendermint/libs/json"
)

//-------------------------------------

var _ crypto.PrivKey = PrivKey{}

var (
	ErrBLSSignature       = errors.New("Invalid BLS signature")
	ErrWrongBLSPriv       = errors.New("bls private key is wrong")
	ErrAggregateSignature = errors.New("Error in BLS aggregate process")
	bn256_suite           = bn256.NewSuite()
)

const (
	PrivKeyName = "chainBFT/PrivKeyBLS"
	PubKeyName  = "chainBFT/PubKeyBLS"
	// PubKeySize is is the size, in bytes, of public keys as used in this package.
	PubKeySize = 32
	// PrivateKeySize is the size, in bytes, of private keys as used in this package.
	PrivateKeySize = 64
	// Size of an Edwards25519 signature. Namely the size of a compressed
	// Edwards25519 point, and a field element. Both of which are 32 bytes.
	SignatureSize = 64
	// SeedSize is the size, in bytes, of private key seeds. These are the
	// private key representations used by RFC 8032.
	SeedSize = 32

	KeyType = "bls"
)

func init() {
	tmjson.RegisterType(PubKey{}, PubKeyName)
	tmjson.RegisterType(PrivKey{}, PrivKeyName)
}

// PrivKey implements crypto.PrivKey.
type PrivKey []byte

// Bytes returns the privkey byte format.
func (privKey PrivKey) Bytes() []byte {
	return []byte(privKey)
}

// Sign produces a signature on the provided message.
func (privKey PrivKey) Sign(msg []byte) ([]byte, error) {
	priv := bn256_suite.G2().Scalar().One()

	// 将[]byte转换为kyber的私钥类型
	if err := priv.UnmarshalBinary(privKey); err != nil {
		return nil, fmt.Errorf("bls private key is wrong, err=%w", err)
	}
	if sig, err := bls.Sign(bn256_suite, priv, msg); err != nil {
		return nil, fmt.Errorf("genearte signature failed. err=%w", err)
	} else {
		return sig, nil
	}
}

// PubKey gets the corresponding public key from the private key.
func (privKey PrivKey) PubKey() crypto.PubKey {
	pubkeyBytes := make([]byte, len(privKey))
	copy(pubkeyBytes, privKey)
	priv := bn256_suite.G2().Scalar().One()
	if err := priv.UnmarshalBinary(pubkeyBytes); err != nil {
		return nil
	}
	pub := bn256_suite.G2().Point().Mul(priv, nil)
	pubBytes, _ := pub.MarshalBinary()
	return PubKey(pubBytes)
}

// Equals - you probably don't need to use this.
// Runs in constant time based on length of the keys.
func (privKey PrivKey) Equals(other crypto.PrivKey) bool {
	if otherEd, ok := other.(PrivKey); ok {
		return subtle.ConstantTimeCompare(privKey[:], otherEd[:]) == 1
	}

	return false
}

func (privKey PrivKey) Type() string {
	return KeyType
}

// ToKyber 将私钥转换为kyber的私钥类型
func (privKey PrivKey) ToKyber() (kyber.Scalar, error) {
	x := bn256_suite.G2().Scalar().One()
	err := x.UnmarshalBinary(privKey)
	if err != nil {
		return nil, err
	}
	return x, nil
}

// GenPrivKey generates a new BLS private key.
// It uses OS randomness in conjunction with the current global random seed
// in tendermint/libs/common to generate the private key.
func GenPrivKey() PrivKey {
	return genPrivKey(crypto.CReader())
}

// GenPrivKeyWithSeed 临时函数，使用指定的seed生成bls私钥
func GenPrivKeyWithSeed(seed int64) PrivKey {
	rander := rand.New(rand.NewSource(seed))
	return genPrivKey(rander)
}

// genPrivKey generates a new ed25519 private key using the provided reader.
func genPrivKey(rander io.Reader) PrivKey {
	cipher1 := random.New(rander)

	priv, _ := bls.NewKeyPair(bn256_suite, cipher1)
	data, _ := priv.MarshalBinary()
	return PrivKey(data)
}

func GenTestPrivKey(seed int64) PrivKey {
	rander := rand.New(rand.NewSource(seed))
	return genPrivKey(rander)
}

// Kyber2PrivKey kyber类型转换为内部私钥类型
func Kyber2PrivKey(x kyber.Scalar) PrivKey {
	bytesPriv, _ := x.MarshalBinary()
	return PrivKey(bytesPriv)
}

//-------------------------------------

var _ crypto.PubKey = PubKey{}

// PubKeyEd25519 implements crypto.PubKey for the Ed25519 signature scheme.
type PubKey []byte

// Address is the SHA256-20 of the raw pubkey bytes.
func (pubKey PubKey) Address() crypto.Address {
	return crypto.Address(tmhash.SumTruncated(pubKey))
}

// Bytes returns the PubKey byte format.
func (pubKey PubKey) Bytes() []byte {
	return []byte(pubKey)
}

func (pubKey PubKey) VerifySignature(msg []byte, sig []byte) bool {
	if sig == nil || len(sig) == 0 {
		return false
	}

	// 将byte数据还原成公钥
	pub := bn256_suite.G2().Point()
	if err := pub.UnmarshalBinary(pubKey); err != nil {
		return false
	}

	// byte还原为bls的Sign结构
	if err := bls.Verify(bn256_suite, pub, msg, sig); err == nil {
		return true
	} else {
		return false
	}
}

func (pubKey PubKey) String() string {
	return fmt.Sprintf("PubKeyBLS{%X}", []byte(pubKey))
}

func (pubKey PubKey) Type() string {
	return KeyType
}

func (pubKey PubKey) Equals(other crypto.PubKey) bool {
	if otherEd, ok := other.(PubKey); ok {
		return bytes.Equal(pubKey[:], otherEd[:])
	}

	return false
}

// GenPrimaryKey 返回一个集群的公共公钥
func GenPrimaryKeyWithSeed(seed int64) crypto.PubKey {
	priv := GenPrivKeyWithSeed(seed)

	return priv.PubKey()
}

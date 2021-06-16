package bls_test

import (
	"chainbft_demo/crypto/bls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"
	"testing"
)

func TestSignAndValidateBLS(t *testing.T) {

	privKey := bls.GenPrivKey()
	pubKey := privKey.PubKey()

	msg := crypto.CRandBytes(128)
	sig, err := privKey.Sign(msg)
	require.Nil(t, err)

	// Test the signature
	assert.True(t, pubKey.VerifySignature(msg, sig))

	// Mutate the signature, just one bit.
	sig[7] ^= byte(0x01)

	assert.False(t, pubKey.VerifySignature(msg, sig))
}

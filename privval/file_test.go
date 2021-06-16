package privval

import (
	"chainbft_demo/crypto/bls"
	"testing"
)

const (
	keyFilePath = "/Users/thegloves/workspace/chainbft_demo/test/test.key"
)

func TestExampleGenFilePV(t *testing.T) {
	priv := bls.GenPrivKey()
	filePV := NewFilePV(priv, keyFilePath)

	filePV.Save()
}

func TestLoadFilePV(t *testing.T) {
	filePV := LoadFilePV(keyFilePath)

	t.Log(filePV)
}

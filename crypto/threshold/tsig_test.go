package threshold

import (
	"bytes"
	"chainbft_demo/crypto/bls"
	"github.com/stretchr/testify/assert"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	kbls "go.dedis.ch/kyber/v3/sign/bls"
	"testing"
	"time"
)

const (
	testseeds = int64(12341235)
)

func TestSignatureRecovery(t *testing.T) {
	msg := []byte("test signature for threshold")
	d := 100

	private := bls.GenPrivKey()
	t.Log(len(private))
	pub := (private.PubKey()).(bls.PubKey)
	public := bn256_suite.G2().Point()
	err := public.UnmarshalBinary(pub)
	assert.Nil(t, err)

	origin_sig, _ := private.Sign(msg)

	priv := bn256_suite.G2().Scalar().One()
	err = priv.UnmarshalBinary(private)
	assert.Nil(t, err)

	polynome := Master(priv, d, testseeds)

	t.Log("share private: ", private)
	t.Log("polynome: ", polynome)

	//ids := []int{151, 2426, 45, 673, 123, 7564}
	ids := func(lens int) []int64 {
		data := make([]int64, lens, lens)
		for i := 1; i <= lens; i++ {
			data[i-1] = int64(i) //rand.Int() % 90000
		}
		return data
	}(d + 1)

	suite := bn256.NewSuite()
	t.Log(polynome)
	sigs := [][]byte{}
	for _, id := range ids {
		x_i, err := polynome.GetValue(id)
		ss, _ := x_i.MarshalBinary()
		t.Log(id, len(ss), x_i)
		assert.Nil(t, err, "计算f(x)函数值出错。err：", err)
		if sig, err := kbls.Sign(suite, x_i, msg); err != nil {
			t.Error(err)
		} else {
			assert.Nil(t, kbls.Verify(suite, suite.G2().Point().Mul(x_i, nil), msg, sig))
			sigs = append(sigs, sig)
		}
	}
	tt := time.Now()
	sig, err := SignatureRecovery(d, sigs, ids)
	t.Log(time.Now().Sub(tt).Seconds())
	t.Log("ids: ", ids)
	t.Log("sigs: ", sigs)
	t.Log("sig: ", sig)
	t.Log("origin sig: ", origin_sig)

	assert.Nil(t, err, "签名还原错误")
	assert.NotNil(t, sig, "还原的签名为空")
	assert.True(t, bytes.Equal(sig, origin_sig), "还原出来的签名和原始签名不相等")
	assert.Nil(t, kbls.Verify(suite, public, msg, sig), "还原出来的签名验证错误")
	assert.Nil(t, kbls.Verify(suite, public, msg, origin_sig), "还原出来的签名验证错误")
}

func TestMaster(t *testing.T) {
	d := 3

	for i := 0; i < 3; i++ {
		private := bls.GenPrivKey()
		priv := bn256_suite.G2().Scalar().One()
		err := priv.UnmarshalBinary(private)
		assert.Nil(t, err)
		polynome := Master(priv, d, testseeds)

		t.Log(polynome)
	}
}

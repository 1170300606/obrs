package main

import "testing"

func TestGenerateTx(t *testing.T) {
	for i := 0; i < 100; i++ {
		tx := generateTx(1000)
		t.Log(tx)
	}
}

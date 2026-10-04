package main

import (
	"crypto/sha256"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/tyler-smith/go-bip39"
	"testing"
)

const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

func masterKey(t *testing.T) *hdkeychain.ExtendedKey {
	t.Helper()
	seed := bip39.NewSeed(testMnemonic, "")
	key, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("NewMaster: %v", err)
	}
	return key
}

func fingerprintOf(key *hdkeychain.ExtendedKey) (uint32, error) {
	pub, err := key.ECPubKey()
	if err != nil {
		return 0, err
	}
	hash160 := btcutil.Hash160(pub.SerializeCompressed())
	return uint32(hash160[0])<<24 | uint32(hash160[1])<<16 | uint32(hash160[2])<<8 | uint32(hash160[3]), nil
}

func sha256Of(b []byte) []byte {
	sum := sha256.Sum256(b)
	return sum[:]
}

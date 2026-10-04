package main

import (
	"testing"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// accountKeyFor derives m/48'/0'/0' — the key SignTx signs with for account 0.
func accountKeyFor(t *testing.T, master *hdkeychain.ExtendedKey) *hdkeychain.ExtendedKey {
	t.Helper()
	purpose, err := master.Derive(hdkeychain.HardenedKeyStart + 48)
	if err != nil {
		t.Fatalf("derive purpose: %v", err)
	}
	coin, err := purpose.Derive(hdkeychain.HardenedKeyStart)
	if err != nil {
		t.Fatalf("derive coin: %v", err)
	}
	account, err := coin.Derive(hdkeychain.HardenedKeyStart)
	if err != nil {
		t.Fatalf("derive account: %v", err)
	}
	return account
}

// psbtWithOneSegwitInput builds a minimal signed-style PSBT with one segwit v0
// P2WSH input carrying the account fingerprint and a change/index path, so
// SignTx accepts and signs it deterministically.
func psbtWithOneSegwitInput(t *testing.T) *psbt.Packet {
	t.Helper()

	master := masterKey(t)
	acct := accountKeyFor(t, master)
	fingerprint, err := fingerprintOf(acct)
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}

	change, err := acct.Derive(0)
	if err != nil {
		t.Fatalf("derive change: %v", err)
	}
	leaf, err := change.Derive(0)
	if err != nil {
		t.Fatalf("derive leaf: %v", err)
	}
	pubKey, err := leaf.ECPubKey()
	if err != nil {
		t.Fatalf("leaf pubkey: %v", err)
	}
	serialized := pubKey.SerializeCompressed()

	witnessScript, err := txscript.NewScriptBuilder().
		AddOp(txscript.OP_1).
		AddData(serialized).
		AddOp(txscript.OP_CHECKSIG).
		Script()
	if err != nil {
		t.Fatalf("witness script: %v", err)
	}
	wsh := sha256Of(witnessScript)
	p2wshScript, err := txscript.NewScriptBuilder().
		AddOp(txscript.OP_0).
		AddData(wsh).
		Script()
	if err != nil {
		t.Fatalf("p2wsh script: %v", err)
	}

	tx := wire.NewMsgTx(2)
	prevHash, err := chainhash.NewHashFromStr("000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	tx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{Hash: *prevHash, Index: 0},
	})
	changeAddr, err := btcutil.NewAddressWitnessScriptHash(wsh, &chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("addr: %v", err)
	}
	changeScript, err := txscript.PayToAddrScript(changeAddr)
	if err != nil {
		t.Fatalf("script: %v", err)
	}
	tx.AddTxOut(wire.NewTxOut(1000, changeScript))

	pkt, err := psbt.NewFromUnsignedTx(tx)
	if err != nil {
		t.Fatalf("NewFromUnsignedTx: %v", err)
	}

	pkt.Inputs[0].WitnessUtxo = wire.NewTxOut(1000, p2wshScript)
	pkt.Inputs[0].WitnessScript = witnessScript
	pkt.Inputs[0].SighashType = txscript.SigHashAll
	pkt.Inputs[0].Bip32Derivation = []*psbt.Bip32Derivation{
		{
			PubKey:               serialized,
			MasterKeyFingerprint: swapUint32(fingerprint),
			Bip32Path:            []uint32{0, 0},
		},
	}

	return pkt
}

func swapUint32(x uint32) uint32 {
	return x<<24 | (x&0xff00)<<8 | (x&0xff0000)>>8 | x>>24
}

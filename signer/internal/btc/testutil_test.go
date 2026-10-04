package btc

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/tyler-smith/go-bip39"
)

// Fixed test seed (BIP39 standard test vector mnemonic, empty password).
// Never contains value.
const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

func testMasterKey(t *testing.T) *hdkeychain.ExtendedKey {
	t.Helper()
	seed := bip39.NewSeed(testMnemonic, "")
	key, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("NewMaster: %v", err)
	}
	return key
}

// accountKey derives m/48'/0'/0' (the key SignTx signs with for account 0).
func accountKey(t *testing.T, master *hdkeychain.ExtendedKey) *hdkeychain.ExtendedKey {
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

// psbtWithOneSegwitInput builds a minimal, well-formed PSBT with one segwit v0
// P2WSH input carrying the account-level fingerprint and the derivation path
// SignInput expects, so signing finds a matching derivation deterministically.
func psbtWithOneSegwitInput(t *testing.T) *psbt.Packet {
	t.Helper()

	master := testMasterKey(t)
	acct := accountKey(t, master)
	fingerprint, err := ComputeFingerprint(acct)
	if err != nil {
		t.Fatalf("ComputeFingerprint: %v", err)
	}

	// The leaf key the derivation path resolves to (change 0 / index 0 below
	// the account key).
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
		t.Fatalf("ECPubKey: %v", err)
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
	wsh := sha256.Sum256(witnessScript)
	p2wshScript, err := txscript.NewScriptBuilder().
		AddOp(txscript.OP_0).
		AddData(wsh[:]).
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
	changeAddr, err := btcutil.NewAddressWitnessScriptHash(wsh[:], &chaincfg.MainNetParams)
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
			PubKey: serialized,
			// SignInput compares bits.ReverseBytes32(stored) against the
			// computed fingerprint, so store it byte-swapped to match the
			// serialization convention used by our coordinators.
			MasterKeyFingerprint: swapUint32(fingerprint),
			// SignInput walks the path *below* the account key it is given
			// (m/48'/0'/acct'), so the recorded steps are change/index.
			Bip32Path: []uint32{0, 0},
		},
	}

	return pkt
}

func psbtBytes(t *testing.T, p *psbt.Packet) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := p.Serialize(&buf); err != nil {
		t.Fatalf("serialize: %v", err)
	}
	return buf.Bytes()
}

func truncatedPrevTx() *wire.MsgTx {
	tx := wire.NewMsgTx(2)
	tx.AddTxIn(&wire.TxIn{})
	// Zero outputs: any vout index is out of range.
	return tx
}

var _ = hex.EncodeToString

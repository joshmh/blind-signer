package btc

import (
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/txscript"
)

// TestSignInputRejectsSigHashDefault pins the delta of commit 6f12b3e:
// accepting a SIGHASH_DEFAULT input (0x00) while emitting a SIGHASH_ALL
// signature produces a trailing hashtype byte that contradicts the declared
// sighash — nonstandard for segwit v0 and a guaranteed downstream failure.
// The fix refuses such inputs (fail-closed); real SIGHASH_DEFAULT support
// requires Taproot signing.
func TestSignInputRejectsSigHashDefault(t *testing.T) {
	p := psbtWithOneSegwitInput(t)
	p.Inputs[0].SighashType = txscript.SigHashDefault

	fingerprint, err := ComputeFingerprint(accountKey(t, testMasterKey(t)))
	if err != nil {
		t.Fatalf("ComputeFingerprint: %v", err)
	}

	_, err = SignInput(p, 0, accountKey(t, testMasterKey(t)), fingerprint)
	if err == nil {
		t.Fatal("expected error for SIGHASH_DEFAULT input on segwit v0, got nil")
	}
	if !strings.Contains(err.Error(), "SIGHASH_ALL") {
		t.Fatalf("expected SIGHASH_ALL diagnosis, got: %v", err)
	}
}

// TestSignInputEmitsSigHashAllByte asserts the emitted partial signature's
// trailing hashtype byte equals SIGHASH_ALL (0x01).
func TestSignInputEmitsSigHashAllByte(t *testing.T) {
	p := psbtWithOneSegwitInput(t)

	fingerprint, err := ComputeFingerprint(accountKey(t, testMasterKey(t)))
	if err != nil {
		t.Fatalf("ComputeFingerprint: %v", err)
	}

	signed, err := SignInput(p, 0, accountKey(t, testMasterKey(t)), fingerprint)
	if err != nil {
		t.Fatalf("SignInput: %v", err)
	}
	if len(signed.Inputs[0].PartialSigs) != 1 {
		t.Fatalf("expected 1 partial sig, got %d", len(signed.Inputs[0].PartialSigs))
	}
	sig := signed.Inputs[0].PartialSigs[0].Signature
	if len(sig) == 0 || sig[len(sig)-1] != byte(txscript.SigHashAll) {
		t.Fatalf("signature must end in SIGHASH_ALL byte 0x01, got %#x", sig)
	}
}

// TestSignInputRejectsLegacyInput pins issue 14: with no witness UTXO and no
// witness script, the signer must fail fast instead of emitting a signature
// that only fails validation at broadcast time.
func TestSignInputRejectsLegacyInput(t *testing.T) {
	p := psbtWithOneSegwitInput(t)
	p.Inputs[0].WitnessScript = nil
	p.Inputs[0].WitnessUtxo = nil
	p.Inputs[0].NonWitnessUtxo = nil

	fingerprint, err := ComputeFingerprint(accountKey(t, testMasterKey(t)))
	if err != nil {
		t.Fatalf("ComputeFingerprint: %v", err)
	}

	_, err = SignInput(p, 0, accountKey(t, testMasterKey(t)), fingerprint)
	if err == nil {
		t.Fatal("expected error for legacy input, got nil")
	}
	if !strings.Contains(err.Error(), "legacy") {
		t.Fatalf("expected legacy-input diagnosis, got: %v", err)
	}
}

// TestSignInputDeriveErrorDiagnosis pins issue 5: a malformed/hostile
// derivation path must report "invalid derivation path", never the misleading
// "Public Key does not match".
func TestSignInputDeriveErrorDiagnosis(t *testing.T) {
	p := psbtWithOneSegwitInput(t)
	// Deriving a hardened step from a *public* key fails (BIP32 forbids it),
	// forcing Derive to fail mid-walk. The old code swallowed this and later
	// reported a misleading pubkey mismatch.
	p.Inputs[0].Bip32Derivation[0].Bip32Path = []uint32{hdkeychain.HardenedKeyStart + 48, hdkeychain.HardenedKeyStart + 99}

	fingerprint, err := ComputeFingerprint(accountKey(t, testMasterKey(t)))
	if err != nil {
		t.Fatalf("ComputeFingerprint: %v", err)
	}

	// Hand SignInput the neutered (public) account key so the hardened step
	// in the recorded path fails instead of silently deriving a wrong key.
	pubKey, err := accountKey(t, testMasterKey(t)).Neuter()
	if err != nil {
		t.Fatalf("Neuter: %v", err)
	}

	_, err = SignInput(p, 0, pubKey, fingerprint)
	if err == nil {
		t.Fatal("expected derivation error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid derivation path") {
		t.Fatalf("expected invalid-derivation-path diagnosis, got: %v", err)
	}
	if strings.Contains(err.Error(), "does not match") {
		t.Fatalf("derive failure misdiagnosed as pubkey mismatch: %v", err)
	}
}

// TestGetUtxoBoundsChecksPrevoutIndex pins issue 11: a crafted PSBT whose
// prevout index exceeds the non-witness utxo's outputs must return an error,
// not panic with index-out-of-range.
func TestGetUtxoBoundsChecksPrevoutIndex(t *testing.T) {
	p := psbtWithOneSegwitInput(t)
	p.Inputs[0].WitnessUtxo = nil
	p.Inputs[0].NonWitnessUtxo = truncatedPrevTx()
	p.UnsignedTx.TxIn[0].PreviousOutPoint.Index = 5

	if _, err := GetUtxo(p, 0); err == nil {
		t.Fatal("expected out-of-range prevout index error, got nil")
	}
}

func TestSignTxRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	out := dir + "/out.psbt"
	psbt := psbtBytes(t, psbtWithOneSegwitInput(t))

	if err := SignTx(0, 0, psbt, testMasterKey(t), out); err != nil {
		t.Fatalf("first SignTx: %v", err)
	}
	if err := SignTx(0, 0, psbt, testMasterKey(t), out); err == nil {
		t.Fatal("expected refuse-if-exists error on second SignTx, got nil")
	}
}

// TestSignTxRejectsHugeAccount pins issue 10 at the library boundary.
func TestSignTxRejectsHugeAccount(t *testing.T) {
	dir := t.TempDir()
	out := dir + "/out.psbt"
	psbt := psbtBytes(t, psbtWithOneSegwitInput(t))

	err := SignTx(0, hdkeychain.HardenedKeyStart, psbt, testMasterKey(t), out)
	if err == nil {
		t.Fatal("expected account >= 2^31 rejection")
	}
	if !strings.Contains(err.Error(), "2^31") {
		t.Fatalf("expected hardened-bit diagnosis, got: %v", err)
	}
}

// TestSignTxOutputPermissions pins issue 6: signed outputs are 0600.
func TestSignTxOutputPermissions(t *testing.T) {
	dir := t.TempDir()
	out := dir + "/out.psbt"
	psbt := psbtBytes(t, psbtWithOneSegwitInput(t))

	if err := SignTx(0, 0, psbt, testMasterKey(t), out); err != nil {
		t.Fatalf("SignTx: %v", err)
	}
	// umask can only remove bits; assert owner-read/write survive and no
	// group/other bits leak through a permissive umask of 000.
	info, err := statFile(out)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("expected 0600, got %o", perm)
	}
}

// TestSignTxGolden pins byte-stable signing output for a fixed seed: the
// serialized signed PSBT must be identical across runs (deterministic RFC6979
// nonces in btcec).
func TestSignTxGolden(t *testing.T) {
	dir := t.TempDir()
	out := dir + "/golden.psbt"
	psbt := psbtBytes(t, psbtWithOneSegwitInput(t))

	if err := SignTx(0, 0, psbt, testMasterKey(t), out); err != nil {
		t.Fatalf("SignTx: %v", err)
	}
	first := readFile(t, out)

	out2 := dir + "/golden2.psbt"
	if err := SignTx(0, 0, psbt, testMasterKey(t), out2); err != nil {
		t.Fatalf("SignTx 2: %v", err)
	}
	second := readFile(t, out2)

	if !bytesEqual(first, second) {
		t.Fatal("signed PSBT output is not byte-stable across runs")
	}
}

// TestValidateNetworkMalformed pins the malformed-program branch.
func TestValidateNetworkMalformed(t *testing.T) {
	p := psbtWithOneSegwitInput(t)
	if err := ValidateNetwork(p, 0); err != nil {
		t.Fatalf("valid v0 program should pass: %v", err)
	}

	p.Inputs[0].WitnessUtxo.PkScript[1] = 0xFF // impossible push length
	if err := ValidateNetwork(p, 0); err == nil {
		t.Fatal("expected malformed witness program rejection")
	}
}

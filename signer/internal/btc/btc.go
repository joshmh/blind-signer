package btc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/bech32"
	"github.com/btcsuite/btcd/chaincfg"
	"math/bits"
	"os"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/pkg/errors"
)

func GetUtxo(p *psbt.Packet, index int) (*wire.TxOut, error) {
	input := p.Inputs[index]
	voutIndex := p.UnsignedTx.TxIn[index].PreviousOutPoint.Index

	if input.NonWitnessUtxo != nil {
		prevOut := p.UnsignedTx.TxIn[index].PreviousOutPoint
		// If a non-witness UTXO is provided, its hash must match the hash specified in the prevout
		if input.NonWitnessUtxo.TxHash() != prevOut.Hash {
			return nil, fmt.Errorf("utxo hash doens't match previous outpoint")
		}

		if int(voutIndex) >= len(input.NonWitnessUtxo.TxOut) {
			return nil, fmt.Errorf("prevout index %d out of range for non-witness utxo", voutIndex)
		}

		return input.NonWitnessUtxo.TxOut[voutIndex], nil
	}

	if input.WitnessUtxo != nil {
		return input.WitnessUtxo, nil
	}

	return nil, fmt.Errorf("error fetching utxo for index")
}

func ComputeFingerprint(masterKey *hdkeychain.ExtendedKey) (uint32, error) {
	seedPub, err := masterKey.ECPubKey()
	if err != nil {
		return 0, fmt.Errorf("failed to derive key: %v", err)
	}

	// Compute the HASH160 of the public key
	hash160 := btcutil.Hash160(seedPub.SerializeCompressed())

	// The fingerprint is the first 4 bytes of the HASH160 hash
	fingerprint := binary.BigEndian.Uint32(hash160[:4])

	return fingerprint, nil
}

func SignInput(p *psbt.Packet, index int, masterKey *hdkeychain.ExtendedKey, fingerprint uint32) (*psbt.Packet, error) {
	input := p.Inputs[index]

	count := 0
	for _, derivation := range input.Bip32Derivation {
		masterKeyFingerprint := bits.ReverseBytes32(derivation.MasterKeyFingerprint)

		if masterKeyFingerprint == fingerprint {
			count += 1

			path := derivation.Bip32Path
			derivationKey := masterKey
			for _, d := range path {
				var err error
				derivationKey, err = derivationKey.Derive(d)
				if err != nil {
					return nil, errors.Wrap(err, "invalid derivation path")
				}
			}

			pubKey, err := derivationKey.ECPubKey()
			if err != nil {
				return nil, err
			}

			serializedPubKey := pubKey.SerializeCompressed()
			if !bytes.Equal(derivation.PubKey, serializedPubKey) {
				return nil, errors.New("Public Key does not match")
			}

			privKey, err := derivationKey.ECPrivKey()
			if err != nil {
				return nil, err
			}

			// Witness (segwit v0) inputs only; a legacy input would receive a
			// signature that fails validation only at broadcast time. Check
			// before touching the UTXO so the diagnosis is not masked by a
			// missing-utxo error.
			if input.WitnessScript == nil && input.WitnessUtxo == nil {
				return nil, errors.New("legacy (non-segwit) inputs unsupported")
			}

			// Get the utxo
			utxo, err := GetUtxo(p, index)
			if err != nil {
				return nil, errors.Wrap(err, "Error fetching utxo")
			}
			// Create the signature.
			prevOutputFetcher := txscript.NewCannedPrevOutputFetcher(
				utxo.PkScript, utxo.Value,
			)
			sigHashes := txscript.NewTxSigHashes(p.UnsignedTx, prevOutputFetcher)

			// Sighash 0x00 (SIGHASH_DEFAULT) is undefined for segwit v0, so
			// only SIGHASH_ALL can be honored here; supporting SIGHASH_DEFAULT
			// requires Taproot signing (BIP341/342).
			if input.SighashType != txscript.SigHashAll {
				return nil, errors.New("Only SIGHASH_ALL is supported")
			}

			sig, err := txscript.RawTxInWitnessSignature(p.UnsignedTx, sigHashes, index,
				utxo.Value, input.WitnessScript,
				txscript.SigHashAll, privKey)
			if err != nil {
				return nil, errors.Wrap(err, "Error creating signature")
			}

			// Use the Updater to add the signature to the input.
			u, err := psbt.NewUpdater(p)
			if err != nil {
				return nil, errors.Wrap(err, "Error creating updater")
			}
			success, err := u.Sign(index, sig, serializedPubKey, nil, nil)
			if err != nil {
				return nil, errors.Wrap(err, "Error updating PSBT")
			}
			if success != psbt.SignSuccesful {
				return nil, errors.Wrap(err, "Error signing PSBT")
			}

			return p, nil
		}
	}

	if count == 0 {
		return nil, errors.New("No matching derivation found (are you using the correct account?)")
	}
	return p, nil
}

// ValidateNetwork checks that the PSBT's inputs belong to the coin type the
// operator selected. A mismatch means the transaction was built for another
// network; signing it would only produce confusing pubkey-mismatch errors
// downstream, so fail fast with an explicit diagnosis.
func ValidateNetwork(p *psbt.Packet, coin_type uint32) error {
	expectedHrp := chaincfg.MainNetParams.Bech32HRPSegwit
	if coin_type == 1 {
		expectedHrp = chaincfg.TestNet3Params.Bech32HRPSegwit
	}

	for index := range p.Inputs {
		witnessUtxo := p.Inputs[index].WitnessUtxo
		if witnessUtxo == nil || len(witnessUtxo.PkScript) == 0 {
			continue
		}

		// Witness programs (BIP141): script = [version_byte, push_len,
		// program...]. Segwit v0 (0x00) uses bech32 and is what we sign;
		// other versions (e.g. 0x51 taproot) are skipped here.
		if witnessUtxo.PkScript[0] != 0x00 || len(witnessUtxo.PkScript) < 4 {
			continue
		}

		program := witnessUtxo.PkScript[2:]

		// Re-encode as a segwit address under the *expected* HRP: the data
		// part is the witness version (0) followed by the 5-bit-converted
		// program, mirroring encodeSegWitAddress. Rebuilding the script from
		// the decoded address must reproduce the original script exactly;
		// any mismatch means the PSBT belongs to another network (or is
		// malformed), and signing it would only fail later with misleading
		// pubkey noise.
		converted, err := bech32.ConvertBits(program, 8, 5, true)
		if err != nil {
			return fmt.Errorf("input %d: invalid witness program: %v", index, err)
		}
		data := append([]byte{0x00}, converted...)
		encoded, err := bech32.Encode(expectedHrp, data)
		if err != nil {
			return fmt.Errorf("input %d: cannot encode witness address: %v", index, err)
		}
		addr, err := btcutil.DecodeAddress(encoded, nil)
		if err != nil {
			return fmt.Errorf("input %d: cannot decode witness address: %v", index, err)
		}
		rebuilt, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return fmt.Errorf("input %d: cannot rebuild script: %v", index, err)
		}
		if !bytes.Equal(rebuilt, witnessUtxo.PkScript) {
			return fmt.Errorf("input %d script does not match a %q segwit address; wrong --testnet selection?", index, expectedHrp)
		}
	}

	return nil
}

func SignTx(coin_type uint32, account uint32, psbtBytes []byte, extPrivateKey *hdkeychain.ExtendedKey, out_file string) error {
	// Create reader for the PSBT
	r := bytes.NewReader(psbtBytes)

	// Create instance of a PSBT
	p, err := psbt.NewFromRawBytes(r, false)
	if err != nil {
		return errors.Wrap(err, "Error parsing PSBT")
	}

	if err := ValidateNetwork(p, coin_type); err != nil {
		return err
	}

	// Derivation path is m / purpose' / coin_type' / account' / change / address_index

	// Purpose
	masterKey := extPrivateKey
	childKey, err := masterKey.Derive(hdkeychain.HardenedKeyStart + 48)
	if err != nil {
		return errors.Wrap(err, "Failed to derive key")
	}

	// Coin type; 0 for mainnet, 1 for testnet
	childKey, err = childKey.Derive(hdkeychain.HardenedKeyStart + coin_type)
	if err != nil {
		return errors.Wrap(err, "Failed to derive key")
	}

	// Account. Callers reject account >= 2^31; this guard keeps the hardened
	// bit set if SignTx gains other callers.
	if account >= hdkeychain.HardenedKeyStart {
		return errors.New("account must be less than 2^31")
	}
	childKey, err = childKey.Derive(hdkeychain.HardenedKeyStart + account)
	if err != nil {
		return errors.Wrap(err, "Failed to derive key")
	}

	fingerprint, err := ComputeFingerprint(childKey)
	if err != nil {
		return errors.Wrap(err, "Failed to compute fingerprint")
	}

	// Sign inputs
	for index := range p.Inputs {
		_, err := SignInput(p, index, childKey, fingerprint)
		if err != nil {
			return errors.Wrap(err, "Error signing input")
		}
	}

	// Refuse to overwrite: a truncated or stale output file must never
	// masquerade as a fresh signature. 0600 since the payload commits funds.
	file, err := os.OpenFile(out_file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.Wrap(err, "Error opening file for writing")
	}
	defer file.Close()

	// Serialize the PSBT directly into the file
	err = p.Serialize(file)
	if err != nil {
		return errors.Wrap(err, "Error serializing PSBT")
	}

	return nil
}

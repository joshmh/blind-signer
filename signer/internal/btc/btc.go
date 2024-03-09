package btc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math/bits"

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

		return input.NonWitnessUtxo.TxOut[voutIndex], nil
	}

	if input.WitnessUtxo != nil {
		return input.WitnessUtxo, nil
	}

	return nil, fmt.Errorf("error fetching utxo for index")
}

func ComputeFingerprint(masterKey *hdkeychain.ExtendedKey) uint32 {
	seedPub, err := masterKey.ECPubKey()
	if err != nil {
		log.Fatalf("Failed to derive key: %v", err)
	}

	// Compute the HASH160 of the public key
	hash160 := btcutil.Hash160(seedPub.SerializeCompressed())

	// The fingerprint is the first 4 bytes of the HASH160 hash
	fingerprint := binary.BigEndian.Uint32(hash160[:4])

	return fingerprint
}

func SignInput(p *psbt.Packet, index int, masterKey *hdkeychain.ExtendedKey, fingerprint uint32) (*psbt.Packet, error) {
	input := p.Inputs[index]

	count := 0
	for _, derivation := range input.Bip32Derivation {
		masterKeyFingerprint := bits.ReverseBytes32(derivation.MasterKeyFingerprint)

		if masterKeyFingerprint == fingerprint {
			count += 1
			// fmt.Printf("Master Key Fingerprint matches.\n")
			path := derivation.Bip32Path
			derivationKey := masterKey
			for _, d := range path {
				derivationKey, _ = derivationKey.Derive(d)
			}

			pubKey, err := derivationKey.ECPubKey()
			if err != nil {
				return nil, err
			}

			serializedPubKey := pubKey.SerializeCompressed()
			if bytes.Equal(derivation.PubKey, serializedPubKey) {
				// fmt.Printf("Public Key matches.\n")
			} else {
				fmt.Printf("Public Key does not match\n")
				return nil, errors.New("Public Key does not match")
			}

			privKey, err := derivationKey.ECPrivKey()
			if err != nil {
				return nil, err
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

			// // Finalize PSBT.
			// err = psbt.Finalize(p, index)
			// if err != nil {
			// 	return nil, errors.Wrap(err, "Error finalizing PSBT")
			// }
			return p, nil
		}
	}

	if count == 0 {
		return nil, errors.New("No matching derivation found (are you using the correct account?)")
	}
	return p, nil
}

func SignTx(coin_type uint32, account uint32, psbtBytes []byte, extPrivateKey *hdkeychain.ExtendedKey) (string, error) {
	// Create reader for the PSBT
	r := bytes.NewReader(psbtBytes)

	// Create instance of a PSBT
	p, err := psbt.NewFromRawBytes(r, false)
	if err != nil {
		return "", errors.Wrap(err, "Error creating PSBT")
	}

	// Derivation path is m / purpose' / coin_type' / account' / change / address_index

	// Purpose
	masterKey := extPrivateKey
	childKey, err := masterKey.Derive(hdkeychain.HardenedKeyStart + 48)
	if err != nil {
		log.Fatalf("Failed to derive key: %v", err)
	}

	// Coin type; 0 for mainnet, 1 for testnet
	childKey, err = childKey.Derive(hdkeychain.HardenedKeyStart + coin_type)
	if err != nil {
		log.Fatalf("Failed to derive key: %v", err)
	}

	// Account
	childKey, err = childKey.Derive(hdkeychain.HardenedKeyStart + account)
	if err != nil {
		log.Fatalf("Failed to derive key: %v", err)
	}

	fingerprint := ComputeFingerprint(childKey)

	// Sign inputs
	for index := range p.Inputs {
		_, err := SignInput(p, index, childKey, fingerprint)
		if err != nil {
			return "", errors.Wrap(err, "Error signing input")
		}
	}

	s, err := p.B64Encode()
	return s, err
}

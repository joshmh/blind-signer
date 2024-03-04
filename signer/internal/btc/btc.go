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

func PrintMasterKey(masterKey *hdkeychain.ExtendedKey) {
	childKey := masterKey
	seedPub, err := childKey.ECPubKey()
	if err != nil {
		log.Fatalf("Failed to derive key: %v", err)
	}

	// Compute the HASH160 of the public key
	hash160 := btcutil.Hash160(seedPub.SerializeCompressed())

	// The fingerprint is the first 4 bytes of the HASH160 hash
	fp1 := hash160[:4]

	childKey, err = childKey.Derive(0)
	if err != nil {
		log.Fatalf("Failed to derive key: %v", err)
	}

	fp2 := childKey.ParentFingerprint()

	childKey, err = childKey.Derive(0)
	if err != nil {
		log.Fatalf("Failed to derive key: %v", err)
	}

	pubkey, err := childKey.ECPubKey()
	if err != nil {
		log.Fatalf("Failed to derive key: %v", err)
	}

	fmt.Printf("Public Key: %x\n", pubkey.SerializeCompressed())
	fmt.Printf("Master Key Fingerprint: %x, %x\n", fp1, fp2)
}

func SignInput(p *psbt.Packet, index int, masterKey *hdkeychain.ExtendedKey, fingerprint uint32) (*psbt.Packet, error) {
	input := p.Inputs[index]

	for _, derivation := range input.Bip32Derivation {
		masterKeyFingerprint := bits.ReverseBytes32(derivation.MasterKeyFingerprint)

		if masterKeyFingerprint == fingerprint {
			fmt.Printf("Master Key Fingerprint matches: %x\n", masterKeyFingerprint)
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
				fmt.Printf("Public Key matches: %x\n", derivation.PubKey)
			} else {
				fmt.Printf("Public Key does not match\n")
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
				return nil, err
			}

			// Use the Updater to add the signature to the input.
			u, err := psbt.NewUpdater(p)
			if err != nil {
				return nil, err
			}
			sucess, err := u.Sign(index, sig, serializedPubKey, nil, nil)
			if err != nil {
				return nil, err
			}
			if sucess != psbt.SignSuccesful {
				return nil, fmt.Errorf("error signing transaction")
			}

			// Finalize PSBT.
			err = psbt.Finalize(p, index)

			return p, err
		}
	}

	return p, nil
}

func SignTx(psbtBytes []byte, extPrivateKey *hdkeychain.ExtendedKey) (string, error) {
	// Create reader for the PSBT
	r := bytes.NewReader(psbtBytes)

	// Create instance of a PSBT
	p, err := psbt.NewFromRawBytes(r, false)
	if err != nil {
		return "", errors.Wrap(err, "Error creating PSBT")
	}

	masterKey := extPrivateKey
	childKey, err := masterKey.Derive(hdkeychain.HardenedKeyStart + 48)
	if err != nil {
		log.Fatalf("Failed to derive key: %v", err)
	}

	childKey, err = childKey.Derive(hdkeychain.HardenedKeyStart + 1)
	if err != nil {
		log.Fatalf("Failed to derive key: %v", err)
	}

	childKey, err = childKey.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		log.Fatalf("Failed to derive key: %v", err)
	}

	PrintMasterKey(childKey)

	fingerprint := ComputeFingerprint(childKey)

	// Sign inputs
	for index := range p.Inputs {
		SignInput(p, index, childKey, fingerprint)
	}

	// tx, err := psbt.Extract(p)
	// if err != nil {
	// 	return "", errors.Wrap(err, "Error extracting transaction")
	// }

	// var signedTx bytes.Buffer
	// tx.Serialize(&signedTx)
	// return hex.EncodeToString(signedTx.Bytes()), nil

	s, err := p.B64Encode()
	return s, err
}

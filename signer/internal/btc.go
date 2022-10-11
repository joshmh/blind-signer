package internal

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

func GetUtxo(p *psbt.Packet, index int) (*wire.TxOut, error) {
	input := p.Inputs[index]
	voutIndex := p.UnsignedTx.TxIn[index].PreviousOutPoint.Index

	if input.NonWitnessUtxo != nil {
		prevOut := p.UnsignedTx.TxIn[index].PreviousOutPoint
		// If a non-witness UTXO is provided, its hash must match the hash specified in the prevout
		if input.NonWitnessUtxo.TxHash() != prevOut.Hash {
			return nil, fmt.Errorf("Utxo hash doens't match previous outpoint")
		}

		return input.NonWitnessUtxo.TxOut[voutIndex], nil
	}

	if input.WitnessUtxo != nil {
		return input.WitnessUtxo, nil
	}

	return nil, fmt.Errorf("Error fetching utxo for index")
}

func SignInput(p *psbt.Packet, index int, masterKey *hdkeychain.ExtendedKey) (*psbt.Packet, error) {
	input := p.Inputs[index]

	// Read the derivation path from the PSBT
	path := input.Bip32Derivation[0]

	// Derive the input key from the master key
	derivatedKey := masterKey
	for _, d := range path.Bip32Path {
		derivatedKey, _ = derivatedKey.Derive(d)

	}

	// Get the derivated public key
	pubKey, err := derivatedKey.ECPubKey()
	if err != nil {
		return nil, err
	}

	// Get the derivated private key
	privKey, err := derivatedKey.ECPrivKey()
	if err != nil {
		return nil, err
	}

	// Get the utxo
	utxo, err := GetUtxo(p, index)

	// Create the signature.
	prevOutputFetcher := txscript.NewCannedPrevOutputFetcher(
		utxo.PkScript, utxo.Value,
	)
	sigHashes := txscript.NewTxSigHashes(p.UnsignedTx, prevOutputFetcher)

	sig, err := txscript.RawTxInWitnessSignature(p.UnsignedTx, sigHashes, index,
		utxo.Value, utxo.PkScript,
		txscript.SigHashAll, privKey)
	if err != nil {
		return nil, err
	}

	// Use the Updater to add the signature to the input.
	u, err := psbt.NewUpdater(p)
	if err != nil {
		return nil, err
	}
	sucess, err := u.Sign(index, sig, pubKey.SerializeCompressed(), nil, nil)
	if err != nil {
		return nil, err
	}
	if sucess != psbt.SignSuccesful {
		return nil, fmt.Errorf("Error signing transaction")
	}

	// Finalize PSBT.
	err = psbt.Finalize(p, index)

	return p, err
}

func SignTx(psbtBase64 string, extPrivateKey string) (string, error) {
	// Create reader for the PSBT
	psbtBytes := []byte(psbtBase64)
	r := bytes.NewReader(psbtBytes)

	// Create instance of a PSBT
	p, err := psbt.NewFromRawBytes(r, true)
	if err != nil {
		return "", err
	}

	// Load the private key
	masterKey, err := hdkeychain.NewKeyFromString(extPrivateKey)
	if err != nil {
		return "", err
	}

	// Sign inputs
	for index := range p.Inputs {
		SignInput(p, index, masterKey)
	}

	tx, err := psbt.Extract(p)
	if err != nil {
		return "", err
	}

	var signedTx bytes.Buffer
	tx.Serialize(&signedTx)
	return hex.EncodeToString(signedTx.Bytes()), nil
}

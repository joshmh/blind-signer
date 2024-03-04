* Signing seems to be working
* Some signing issue
* match the derived pubkey with the expected pubkey in the tx
* Try doing some verification in btcd, might help pinpoint the problem
* Could have to do with missing derivation path

* Can use signtransaction_with_privkey in Electrum. For this we'd need to derive the privkey for the transaction but that's it.
* Sparrow isn't helpful

Try this:

package main

import (
    "github.com/btcsuite/btcd/btcec"
    "github.com/btcsuite/btcutil/hdkeychain"
    "github.com/btcsuite/btcutil/psbt"
    // Other imports as needed
)

func main() {
    // Example PSBT data (in base64 format)
    base64PSBT := "your_base64_encoded_psbt_here"

    // Your extended public key (assuming you have an hdkeychain.ExtendedKey)
    extPubKey := getYourExtendedPublicKey() // Replace with your actual extended public key

    // Extract the EC public key
    ecPubKey, err := extPubKey.ECPubKey()
    if err != nil {
        // Handle error
    }

    // Serialize the EC public key
    serializedPubKey := ecPubKey.SerializeCompressed()

    // Decode the PSBT
    packet, err := psbt.NewFromRawBytes(base64PSBT, false)
    if err != nil {
        // Handle error
    }

    // Iterate over each input and compare public keys
    for i, input := range packet.Inputs {
        for _, partialSig := range input.PartialSigs {
            if len(partialSig.PubKey) == len(serializedPubKey) && string(partialSig.PubKey) == string(serializedPubKey) {
                // Found a matching public key in input i
                // Do something with this information
            }
        }
    }

    // Additional logic...
}

// Replace with your actual function to get the extended public key
func getYourExtendedPublicKey() *hdkeychain.ExtendedKey {
    // Your logic to get the extended public key
}
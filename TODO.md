# SD cards

## Simple solution
We need:
* mnemonic
* password
* txs

1. Read size bytes
2. Read blocks
3. Parse

Format:


# Other
* Signing seems to be working
* Some signing issue
* match the derived pubkey with the expected pubkey in the tx
* Try doing some verification in btcd, might help pinpoint the problem
* Could have to do with missing derivation path

* Can use signtransaction_with_privkey in Electrum. For this we'd need to derive the privkey for the transaction but that's it.
* Sparrow isn't helpful


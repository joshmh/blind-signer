package main

import (
	"gitlab.lamassu.is/pazuz/blind-signer/signer/internal"

	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

func init() {
	internal.SwitchLed("white", false)
	internal.SwitchLed("blue", false)
}

func readData(dataType string) string {
	value, err := internal.ReadData(dataType)
	if err != nil {
		internal.WaitForSDCardInsert()
		return readData(dataType)
	} else {
		return value
	}
}

func main() {
	internal.SwitchLed("white", true)

	internal.WaitForSDCardInsert()

	mnemonic := readData("mnemonic")

	internal.SwitchLed("blue", true)

	internal.WaitForSDCardRemove()

	internal.SwitchLed("blue", false)

	internal.WaitForSDCardInsert()

	password := readData("password")

	internal.SwitchLed("blue", true)

	internal.WaitForSDCardRemove()

	internal.SwitchLed("blue", false)

	internal.WaitForSDCardInsert()

	psbt := readData("psbt")

	seed := bip39.NewSeed(mnemonic, password)
	masterKey, _ := bip32.NewMasterKey(seed)
	masterKeyString := masterKey.String()

	tx, err := internal.SignTx(psbt, masterKeyString)
	if err != nil {
		internal.BlinkLed("blue")
	}

	internal.WriteData("tx", tx)

	internal.SwitchLed("blue", true)
	internal.BlinkLed("white")
}

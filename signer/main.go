package main

import (
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
	"gitlab.lamassu.is/pazuz/blind-signer/signer/internal/armory"
	"gitlab.lamassu.is/pazuz/blind-signer/signer/internal/btc"
)

func init() {
	armory.SwitchLed("white", false)
	armory.SwitchLed("blue", false)
}

func readData(dataType string) string {
	value, err := armory.ReadData(dataType)
	if err != nil {
		armory.WaitForSDCardInsert()
		return readData(dataType)
	} else {
		return value
	}
}

func main() {
	armory.SwitchLed("white", true)

	armory.WaitForSDCardInsert()

	mnemonic := readData("mnemonic")

	armory.SwitchLed("blue", true)

	armory.WaitForSDCardRemove()

	armory.SwitchLed("blue", false)

	armory.WaitForSDCardInsert()

	password := readData("password")

	armory.SwitchLed("blue", true)

	armory.WaitForSDCardRemove()

	armory.SwitchLed("blue", false)

	armory.WaitForSDCardInsert()

	psbt := readData("psbt")

	seed := bip39.NewSeed(mnemonic, password)
	masterKey, _ := bip32.NewMasterKey(seed)
	masterKeyString := masterKey.String()

	tx, err := btc.SignTx(psbt, masterKeyString)
	if err != nil {
		armory.BlinkLed("blue")
	}

	armory.WriteData("tx", tx)

	armory.SwitchLed("blue", true)
	armory.BlinkLed("white")
}

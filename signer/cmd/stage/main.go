package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
	"gitlab.lamassu.is/pazuz/blind-signer/signer/internal/btc"
)

func main() {
	// Parse command-line arguments
	if len(os.Args) != 4 {
		log.Fatal("Usage: stage <handle> <data-dir-path> <psbt-file-path>")
	}
	handle := os.Args[1]
	dataDirPath := os.Args[2]
	psbtFilePath := os.Args[3]

	// Construct the mnemonic file path
	mnemonicFilePath := filepath.Join(dataDirPath, "vault", "mnemonics", handle, "mn.txt")
	passwordFilePath := filepath.Join(dataDirPath, "vault", "pw", "pw.txt")

	// Read mnemonic and PSBT from their paths
	mnemonic := readData(mnemonicFilePath)
	password := readData(passwordFilePath)
	psbt := readBytes(psbtFilePath)

	// Derive master key from mnemonic
	seed := bip39.NewSeed(mnemonic, password)
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		log.Fatalf("Failed to create master key: %v", err)
	}
	masterKeyString := masterKey.String()

	// Sign the transaction
	tx, err := btc.SignTx(psbt, masterKeyString)
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		log.Fatalf("Failed to sign transaction: %v", err)
	}

	// Output or process the signed transaction
	log.Println("Signed Transaction:", tx)
}

func readBytes(filepath string) []byte {
	data, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatalf("Failed to read file at %s: %v", filepath, err)
	}
	return data
}

// readData reads data from a file and returns it as a string
func readData(filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read file at %s: %v", filePath, err)
	}
	return string(data)
}

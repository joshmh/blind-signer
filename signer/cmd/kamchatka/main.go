package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/tyler-smith/go-bip39"
	"gitlab.lamassu.is/pazuz/blind-signer/signer/internal/btc"
)

func sign(handle string) {
	// Read mnemonic and password from files
	mnemonic := readData("play/toxic/mn.txt")
	password := readData("play/toxic/pw.txt")

	// Derive master key from mnemonic and password
	seed := bip39.NewSeed(mnemonic, password)
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		log.Fatalf("Failed to create master key: %v", err)
	}

	// Process transactions from the txs directory
	txsDir := "play/txs"
	signedDir := "play/signed"
	err = processTransactions(txsDir, signedDir, handle, masterKey)
	if err != nil {
		log.Fatalf("Failed to process transactions: %v", err)
	}

	fmt.Println("All transactions processed successfully.")
}

func main() {
	// Parse command-line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  kamchatka init")
		fmt.Println("  kamchatka sign <handle>")
		fmt.Println("\nVersion: 1.0.0")
		os.Exit(1)
	}
	cmd := os.Args[1]

	if cmd == "init" {
		fmt.Println("Initializing kamchatka directories...")
		createDirectory("play")
		createDirectory("play/txs")
		createDirectory("play/signed")
		createDirectory("play/toxic")
		fmt.Println("Done.")
	} else if cmd == "sign" {
		if len(os.Args) != 3 {
			fmt.Println("Usage: kamchatka sign <handle>")
			os.Exit(1)
		}
		handle := os.Args[2]
		sign(handle)
	} else {
		fmt.Println("Invalid command")
		os.Exit(1)
	}
}

func createDirectory(dir string) {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Fatalf("failed to create directory [%s]: %v", dir, err)
	}
}

func processTransactions(txsDir, signedDir, handle string, masterKey *hdkeychain.ExtendedKey) error {
	// Read transactions from the txs directory
	files, err := os.ReadDir(txsDir)
	if err != nil {
		return fmt.Errorf("failed to read txs directory [%s]: %v", txsDir, err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		txnFilename := file.Name()
		parts := strings.Split(txnFilename, "_")
		if len(parts) != 2 {
			log.Printf("Skipping invalid transaction file: %s", txnFilename)
			continue
		}

		name := parts[0]
		account := strings.TrimSuffix(parts[1], ".psbt")

		txnFilePath := filepath.Join(txsDir, txnFilename)
		psbt := readBytes(txnFilePath)

		// Sign the transaction
		tx, err := btc.SignTx(psbt, masterKey)
		if err != nil {
			log.Printf("Failed to sign transaction %s: %v", txnFilename, err)
			continue
		}

		// Convert tx from string to []byte
		txBytes := []byte(tx)

		// Write the signed transaction to the signed directory
		signedTxnFilename := fmt.Sprintf("%s_%s_%s_signed.psbt", name, account, handle)
		signedTxnFilePath := filepath.Join(signedDir, signedTxnFilename)
		err = os.WriteFile(signedTxnFilePath, txBytes, os.ModePerm)
		if err != nil {
			log.Printf("Failed to write signed transaction %s: %v", signedTxnFilename, err)
			continue
		}

		fmt.Printf("Transaction %s signed and saved as %s\n", txnFilename, signedTxnFilename)
	}

	return nil
}

func readBytes(filepath string) []byte {
	data, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatalf("Failed to read file at %s: %v", filepath, err)
	}
	return data
}

func readData(filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read file at %s: %v", filePath, err)
	}
	return strings.TrimSpace(string(data))
}

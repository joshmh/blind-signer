package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip39"
	"gitlab.lamassu.is/pazuz/blind-signer/signer/internal/btc"
)

const (
	version = "1.1.0"
)

func sign(coin_type uint32, account uint32, handle string) {
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
	txsDir := "play/unsigned"
	signedDir := "play/signed"
	total_success, err := processTransactions(coin_type, account, txsDir, signedDir, handle, masterKey)
	if err != nil {
		log.Fatalf("Failed to process transactions: %v", err)
	} else {
		if total_success {
			fmt.Println("All transactions signed successfully.")
			err := os.RemoveAll("play/toxic")
			if err != nil {
				log.Fatalf("Failed to delete toxic directory: %v", err)
			}
			fmt.Println("Toxic directory deleted.")
		} else {
			fmt.Println("Some transactions failed to sign, leaving toxic directory intact.")
		}
	}
}

func main() {
	// Parse command-line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  stevie init")
		fmt.Println("  stevie sign <handle> <account> [--testnet]")
		fmt.Printf("\nVersion: %s\n", version)
		os.Exit(1)
	}
	cmd := os.Args[1]

	if cmd == "init" {
		fmt.Println("Initializing stevie directories...")
		createDirectory("play")
		createDirectory("play/unsigned")
		createDirectory("play/signed")
		createDirectory("play/toxic")
		fmt.Println("Done.")
	} else if cmd == "sign" {
		if len(os.Args) != 4 && len(os.Args) != 5 {
			fmt.Println("Usage: stevie sign <handle> <account> [--testnet]")
			os.Exit(1)
		}

		handle := os.Args[2]

		accountStr := os.Args[3]
		account, err := strconv.ParseUint(accountStr, 10, 32)
		if err != nil {
			fmt.Printf("Invalid account value: %s\n", accountStr)
			os.Exit(1)
		}

		coin_type := uint32(0)
		if len(os.Args) == 5 && os.Args[4] == "--testnet" {
			coin_type = uint32(1)
		}

		sign(coin_type, uint32(account), handle)
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

func processTransactions(coin_type uint32, account uint32, txsDir string, signedDir string, handle string, masterKey *hdkeychain.ExtendedKey) (total_success, error) {
	// Read transactions from the txs directory
	files, err := os.ReadDir(txsDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read txs directory [%s]: %v", txsDir, err)
	}

	count := 0
	signedCount := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		txnFilename := file.Name()
		txnFilenameWithoutSuffix := strings.TrimSuffix(txnFilename, filepath.Ext(txnFilename))
		txnFilePath := filepath.Join(txsDir, txnFilename)
		psbt, err := readBytes(txnFilePath)
		if err != nil {
			log.Printf("Failed to read transaction %s: %v", txnFilename, err)
			continue
		}

		count += 1

		// Sign the transaction
		tx, err := btc.SignTx(coin_type, account, psbt, masterKey)
		if err != nil {
			log.Printf("Failed to sign transaction %s: %v", txnFilename, err)
			continue
		}

		signedCount += 1

		// Convert tx from string to []byte
		txBytes := []byte(tx)

		// Write the signed transaction to the signed directory
		signedTxnFilename := fmt.Sprintf("%s_signed_%s.psbt", txnFilenameWithoutSuffix, handle)
		signedTxnFilePath := filepath.Join(signedDir, signedTxnFilename)
		err = os.WriteFile(signedTxnFilePath, txBytes, os.ModePerm)
		if err != nil {
			log.Printf("Failed to write signed transaction %s: %v", signedTxnFilename, err)
			continue
		}

		fmt.Printf("Transaction %s signed and saved as %s\n", txnFilename, signedTxnFilename)
	}

	fmt.Printf("\nSigned %d out of %d transactions\n", signedCount, count)
	return signedCount == count, nil
}

func readBytes(filepath string) ([]byte, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading file")
	}
	return data, nil
}

func readData(filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read file at %s: %v", filePath, err)
	}
	return strings.TrimSpace(string(data))
}

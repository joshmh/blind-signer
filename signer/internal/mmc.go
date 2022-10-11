package internal

import (
	"errors"
	"log"
	"math/big"
	"time"

	usbarmory "github.com/usbarmory/tamago/board/usbarmory/mk2"
	"github.com/usbarmory/tamago/soc/nxp/usdhc"
	"gitlab.lamassu.is/pazuz/blind-signer/signer/internal/proto"
)

const (
	// We could use the entire iRAM before USB activation,
	// accounting for required dTD alignment which takes
	// additional space (readSize = 0x20000 - 4096).
	readSize         = 0x7fff
	totalReadSize    = 10 * 1024 * 1024
	MaxTransferBytes = 32 * 1024
)

func WriteData(dataType string, dataValue string) (string, error) {
	var lba uint

	if dataType == "mnemonic" {
		lba = 0
	} else if dataType == "password" {
		lba = 8
	} else if dataType == "psbt" {
		lba = 16
	} else if dataType == "tx" {
		lba = 24
	} else {
		return "", errors.New("Invalid datatype")
	}

	bytes, err := proto.Marshal(dataType, dataValue)
	if err != nil {
		return "", err
	}

	err = WriteToSD(bytes, lba)
	if err != nil {
		return "", err
	}

	log.Printf("%x", bytes)

	return dataValue, nil
}

func ReadData(dataType string) (string, error) {
	var lba uint

	if dataType == "mnemonic" {
		lba = 0
	} else if dataType == "password" {
		lba = 8
	} else if dataType == "psbt" {
		lba = 16
	} else if dataType == "tx" {
		lba = 24
	} else {
		return "", errors.New("Invalid datatype")
	}

	buffer := make([]byte, 512)
	err := ReadFromSD(buffer, lba)
	if err != nil {
		return "", err
	}

	lengthData := int(big.NewInt(0).SetBytes(buffer[:4]).Int64())
	totalLength := lengthData + 4

	if r := totalLength % 512; r != 0 && totalLength > 512 {
		buffer = make([]byte, totalLength+512-r)
	}

	err = ReadFromSD(buffer, lba)
	if err != nil {
		return "", err
	}

	dataProto := buffer[4 : lengthData+4]

	if func(data []byte) bool {
		for _, v := range data {
			if v != 0 {
				return false
			}
		}
		return true
	}(dataProto) {
		return "", errors.New("No data to read")
	}

	dataType, dataValue, err := proto.Unmarshal(buffer[4 : lengthData+4])
	if err != nil {
		return "", err
	}

	return dataValue, nil
}

// BlockSize returns the size in bytes of the each block in the underlying storage.
func BlockSize(card *usdhc.USDHC) uint {
	return uint(card.Info().BlockSize)
}

// WriteBlocks writes the data in b to the device blocks starting at the given block address.
// If the final block to be written is partial, it will be padded with zeroes to ensure that
// full blocks are written.
func Write(card *usdhc.USDHC, lba uint, bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}

	blockSize := int(BlockSize(card))

	var lengthBytes = big.NewInt(int64(len(bytes)))

	buffer := make([]byte, 4)
	lengthBytes.FillBytes(buffer)

	bytes = append(buffer, bytes...)

	if r := len(bytes) % blockSize; r != 0 {
		bytes = append(bytes, make([]byte, blockSize-r)...)
	}

	for len(bytes) > 0 {
		max := len(bytes)
		if max > MaxTransferBytes {
			max = MaxTransferBytes
		}
		if err := card.WriteBlocks(int(lba), bytes[:max]); err != nil {
			return err
		}
		bytes = bytes[max:]
		lba += uint(max / blockSize)
	}
	return nil
}

// ReadBlocks reads data from the storage device at the given address into b.
// b must be a multiple of the underlying device's block size.
func Read(card *usdhc.USDHC, lba uint, bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}
	bs := int(BlockSize(card))
	for len(bytes) > 0 {
		max := len(bytes)
		if max > MaxTransferBytes {
			max = MaxTransferBytes
		}
		if err := card.ReadBlocks(int(lba), bytes[:max]); err != nil {
			return err
		}
		bytes = bytes[max:]
		lba += uint(max / bs)
	}
	return nil
}

func WriteToSD(data []byte, lba uint) error {
	card := usbarmory.SD

	err := card.Detect()
	if err != nil {
		return err

	}

	err = Write(card, lba, data)
	if err != nil {
		return errors.New("Error writing to SD card")
	}

	return nil
}

func ReadFromSD(buffer []byte, lba uint) error {
	card := usbarmory.SD

	err := card.Detect()
	if err != nil {
		return err
	}

	err = Read(card, lba, buffer)
	if err != nil {
		return errors.New("Error reading from SD card")
	}

	return nil
}

func WaitForSDCardInsert() {
	card := usbarmory.SD

	for card.Detect() != nil {
		time.Sleep(time.Second / 10)
	}
}

func WaitForSDCardRemove() {
	card := usbarmory.SD

	for card.Detect() == nil {
		time.Sleep(time.Second / 10)
	}
}

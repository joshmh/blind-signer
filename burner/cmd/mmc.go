package cmd

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"

	"golang.org/x/term"

	usbarmory "github.com/usbarmory/tamago/board/usbarmory/mk2"
	"github.com/usbarmory/tamago/soc/nxp/usdhc"
	"gitlab.lamassu.is/pazuz/blind-signer/burner/internal/proto"
)

const (
	// We could use the entire iRAM before USB activation,
	// accounting for required dTD alignment which takes
	// additional space (readSize = 0x20000 - 4096).
	readSize         = 0x7fff
	totalReadSize    = 10 * 1024 * 1024
	MaxTransferBytes = 32 * 1024
)

func String(d string) string {
	switch d {
	case "mnemonic":
		return "Mnemonic"
	case "password":
		return "Password"
	case "psbt":
		return "Partially Signed Transaction"
	case "tx":
		return "Transaction"
	}
	return "unknown"
}

func Block(d string) (uint, error) {
	switch d {
	case "mnemonic":
		return 0, nil
	case "password":
		return 8, nil
	case "psbt":
		return 16, nil
	case "tx":
		return 24, nil
	}
	return 0, errors.New("Error calculating memory location")
}

func init() {
	Add(Cmd{
		Name:    "write",
		Args:    2,
		Pattern: regexp.MustCompile(`^write (mnemonic|password|psbt) (.+)`),
		Syntax:  "(mnemonic|password|psbt) (.+)",
		Help:    "Write to SD Card",
		Fn:      writeCmd,
	})

	Add(Cmd{
		Name:    "read",
		Args:    1,
		Pattern: regexp.MustCompile(`^read (mnemonic|password|psbt|tx)`),
		Syntax:  "(mnemonic|password|psbt|tx)",
		Help:    "Read from SD Card",
		Fn:      readCmd,
	})
}

func writeCmd(_ *term.Terminal, arg []string) (res string, err error) {
	dataType := arg[0]

	lba, err := Block(dataType)

	if err != nil {
		return "Command malformed", nil
	}

	bytes, err := proto.Marshal(arg[0], arg[1])
	if err != nil {
		return "", err
	}

	err = WriteToSD(bytes, lba)
	if err != nil {
		return "", err
	}

	return "Success writing to SD Card", nil
}

func readCmd(_ *term.Terminal, arg []string) (string, error) {
	dataType := arg[0]

	lba, err := Block(dataType)

	if err != nil {
		return "Command malformed", nil
	}

	buffer := make([]byte, 512)
	err = ReadFromSD(buffer, lba)
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
		return "No data to read", nil
	}

	dataType, dataValue, err := proto.Unmarshal(buffer[4 : lengthData+4])
	if err != nil {
		return "", err
	}

	formattedOutput := fmt.Sprintf("\n%s:\n-------------------\n%s\n-------------------\n", String(dataType), dataValue)

	return formattedOutput, nil
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

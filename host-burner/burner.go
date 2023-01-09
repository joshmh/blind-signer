package main

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"

	"gitlab.lamassu.is/pazuz/blind-signer/host-burner/proto"
)

func main() {
	burn, binary, device, file, datatype := parseArguments()

	if burn {
		burnToSD(burn, binary, device, file, datatype)
	} else {
		extractFromSD(burn, binary, device, file, datatype)
	}

	fmt.Println("\033[1;32mDone!\033[0m You can now safely remove SD card.")
}

func burnToSD(burn, binary bool, device, file, datatype string) {
	var stringToSave string
	var data []byte

	if file != "" {
		confirmChoice(fmt.Sprintf("You will load %s from %s and save it into %s.", datatype, file, device))
		data = loadFromFile(file)
	} else {
		fmt.Printf("Input file not specified. Please enter your %s. Press Enter to finish input.\n", datatype)
		data = getDataFromPrompt()
		if len(data) > 0 {
			data = data[:len(data)-1]
		}
		confirmChoice(fmt.Sprintf("You will save %s into %s.", datatype, device))
	}

	if binary {
		stringToSave = base64.StdEncoding.EncodeToString(data)
	} else {
		stringToSave = string(data)
	}

	packedData, err := proto.Marshal(datatype, stringToSave)
	check(err)

	saveToDevice(device, datatype, packedData)
}

func extractFromSD(burn, binary bool, device, file, datatype string) {
	if file != "" {
		confirmChoice(fmt.Sprintf("You will load %s from %s and save it into %s.", datatype, device, file))
	} else {
		confirmChoice(fmt.Sprintf("You will load %s from %s and display it in terminal.", datatype, device))
	}

	var dataToSave []byte
	data := loadFromDevice(device, datatype)

	extractedType, extractedData, err := proto.Unmarshal(data)
	check(err)

	if extractedType != datatype {
		check(errors.New("Types not matching"))
	}

	if binary {
		dataToSave, err = base64.StdEncoding.DecodeString(extractedData)
		check(err)
	} else {
		dataToSave = []byte(extractedData)
	}

	if file != "" {
		saveToFile(file, dataToSave)
	} else {
		fmt.Println(extractedData)
	}
}

func loadFromFile(filePath string) []byte {
	data, err := os.ReadFile(filePath)
	check(err)
	return data
}

func saveToFile(filePath string, data []byte) {
	err := os.WriteFile(filePath, data, 0644)
	check(err)
}

func loadFromDevice(devicePath, datatype string) []byte {
	f, err := os.Open(devicePath)
	check(err)

	blockSize := uint32(512)

	position := blockPosition(datatype)
	_, err = f.Seek(int64(blockSize*position), 0)
	check(err)

	sizeBuffer := make([]byte, 4)
	_, err = f.Read(sizeBuffer)
	check(err)

	size := binary.BigEndian.Uint32(sizeBuffer)

	if size == 0 {
		fmt.Println("Seems like nothing is stored on expected position.")
		os.Exit(1)
	}

	_, err = f.Seek(int64(blockSize*position+4), 0)
	check(err)

	dataBuffer := make([]byte, size)
	_, err = f.Read(dataBuffer)
	check(err)

	return dataBuffer
}

func saveToDevice(devicePath, dataType string, data []byte) {
	f, err := os.OpenFile(devicePath, os.O_RDWR, 0666)
	check(err)

	blockSize := uint32(512)

	position := blockPosition(dataType)
	_, err = f.Seek(int64(blockSize*position), 0)
	check(err)

	lengthBytes := big.NewInt(int64(len(data)))

	buffer := make([]byte, 4)
	lengthBytes.FillBytes(buffer)

	data = append(buffer, data...)

	if r := uint32(len(data)) % blockSize; r != 0 {
		data = append(data, make([]byte, blockSize-r)...)
	}

	_, err = f.Write(data)
	check(err)
}

func check(e error) {
	if e != nil {
		fmt.Printf("\033[1;31m%s\033[0m\n", e)
		os.Exit(1)
	}
}

func blockPosition(d string) uint32 {
	switch d {
	case "mnemonic":
		return 0
	case "password":
		return 8
	case "psbt":
		return 16
	default:
		return 24
	}
}

func parseArguments() (bool, bool, string, string, string) {
	burnFlag := flag.Bool("burn", false, "Write from the file the device")
	binaryFlag := flag.Bool("binary", false, "Read and write tx data in a binary format.")
	devFlag := flag.String("device", "", "Device to write into or read from")
	fileFlag := flag.String("file", "", "File to write into or read from")
	typeFlag := flag.String("type", "", "Type of data [mnemonic|password|psbt|tx]")

	flag.Parse()

	if *typeFlag == "" {
		fmt.Println("Type flag is mandatory")
		os.Exit(1)
	}

	if *typeFlag != "mnemonic" && *typeFlag != "password" && *typeFlag != "psbt" && *typeFlag != "tx" {
		fmt.Println("Type flag can be only 'mnemonic', 'password', 'psbt' or 'tx'.")
		os.Exit(1)
	}

	if *devFlag == "" {
		fmt.Println("Device flag is mandatory")
		os.Exit(1)
	}

	if *burnFlag && *fileFlag != "" {
		_, err := os.Stat(*fileFlag)
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("File %s doesn't exists!\n", *fileFlag)
			os.Exit(2)
		}
	}

	return *burnFlag, *binaryFlag, *devFlag, *fileFlag, *typeFlag
}

func getDataFromPrompt() []byte {
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	check(err)

	return []byte(response)
}

func confirmChoice(message string) {
	fmt.Printf("%s Are you sure? [y/n]: ", message)

	reader := bufio.NewReader(os.Stdin)
	yesNo, _ := reader.ReadString('\n')

	if len(yesNo) > 0 {
		yesNo = yesNo[:len(yesNo)-1]
	}

	if yesNo != "yes" && yesNo != "y" && yesNo != "Y" {
		fmt.Println("Quitting!")
		os.Exit(0)
	}
}

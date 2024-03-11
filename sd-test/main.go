package main

import (
	"os"
	"time"

	usbarmory "github.com/usbarmory/tamago/board/usbarmory/mk2"
)

func main() {
	card := usbarmory.SD
	cardInfo := card.Info()
	blocksize := cardInfo.BlockSize

	if cardInfo.MMC {
		for true {
			usbarmory.LED("blue", true)
			usbarmory.LED("white", true)
			time.Sleep(time.Second / 2)
			usbarmory.LED("blue", false)
			usbarmory.LED("white", false)
			time.Sleep(time.Second / 2)
		}
	}

	for true {
		usbarmory.LED("blue", true)
		usbarmory.LED("white", false)
		for true {
			err := card.Detect()
			if err == nil {
				usbarmory.LED("blue", true)
				usbarmory.LED("white", true)
				break
			}
			time.Sleep(time.Second / 10)
		}

		if !cardInfo.SD {
			for true {
				usbarmory.LED("blue", false)
				usbarmory.LED("white", true)
				time.Sleep(time.Second / 2)
				usbarmory.LED("blue", false)
				usbarmory.LED("white", false)
				time.Sleep(time.Second / 2)
			}
		}

		usbarmory.LED("blue", false)
		usbarmory.LED("white", true)

		buf := make([]byte, blocksize)
		// str := fmt.Sprintf("blocksize: %d", blocksize)
		// strLen := len(str)
		// binary.BigEndian.PutUint16(buf, uint16(strLen))
		copy(buf, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

		time.Sleep(time.Second * 2)

		usbarmory.LED("blue", true)
		usbarmory.LED("white", false)

		time.Sleep(time.Second * 2)

		err := card.WriteBlocks(0, buf)
		if err != nil {
			usbarmory.LED("blue", false)
			usbarmory.LED("white", false)
			time.Sleep(time.Second * 2)
			os.Exit(1)
		}

		time.Sleep(time.Second * 2)

		for true {
			usbarmory.LED("blue", true)
			usbarmory.LED("white", true)
			err := card.Detect()
			if err != nil {
				break
			}
			time.Sleep(time.Second / 10)
			usbarmory.LED("blue", false)
			usbarmory.LED("white", true)
			time.Sleep(time.Second / 10)
		}
	}
}

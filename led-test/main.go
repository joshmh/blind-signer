package main

import (
	"time"

	usbarmory "github.com/usbarmory/tamago/board/usbarmory/mk2"
)

func main() {
	for true {
		usbarmory.LED("blue", true)
		usbarmory.LED("white", false)
		time.Sleep(time.Second / 2)
		usbarmory.LED("blue", false)
		usbarmory.LED("whilte", true)
		time.Sleep(time.Second / 2)
	}
}

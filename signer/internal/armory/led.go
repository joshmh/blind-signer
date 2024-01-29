package armory

import (
	"time"

	usbarmory "github.com/usbarmory/tamago/board/usbarmory/mk2"
)

func SwitchLed(color string, on bool) {
	usbarmory.LED(color, on)
}

func BlinkLed(color string) {
	for true {
		usbarmory.LED(color, true)
		time.Sleep(time.Second / 10)
		usbarmory.LED(color, false)
		time.Sleep(time.Second / 10)
	}
}

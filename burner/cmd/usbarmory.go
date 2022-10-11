package cmd

import (
	"time"

	"golang.org/x/term"

	usbarmory "github.com/usbarmory/tamago/board/usbarmory/mk2"
	"github.com/usbarmory/tamago/soc/nxp/imx6ul"
)

var boardName = "USB armory Mk II"

func init() {
	boardName = usbarmory.Model()
	console = usbarmory.UART2

	if !imx6ul.Native {
		return
	}

	imx6ul.SetARMFreq(900)

	// On the USB armory Mk II the standard serial console (UART2) is
	// exposed through the debug accessory, which needs to be enabled.
	debugConsole, _ := usbarmory.DetectDebugAccessory(250 * time.Millisecond)
	<-debugConsole
}

func rebootCmd(_ *term.Terminal, _ []string) (_ string, _ error) {
	usbarmory.Reset()
	return
}

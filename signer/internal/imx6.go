package internal

import (
	"bytes"
	"fmt"
	"runtime"

	"golang.org/x/term"

	"github.com/usbarmory/tamago/soc/nxp/imx6ul"
)

func Remote() bool {
	return imx6ul.Native && (imx6ul.Family == imx6ul.IMX6UL || imx6ul.Family == imx6ul.IMX6ULL)
}

func Target() (t string) {
	t = fmt.Sprintf("%s %v MHz", imx6ul.Model(), float32(imx6ul.ARMFreq())/1000000)

	if !imx6ul.Native {
		t += " (emulated)"
	}

	return
}

func date(epoch int64) {
	imx6ul.ARM.SetTimer(epoch)
}

func infoCmd(_ *term.Terminal, _ []string) (string, error) {
	var res bytes.Buffer

	res.WriteString(fmt.Sprintf("Runtime ......: %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH))
	res.WriteString(fmt.Sprintf("Board ........: %s\n", boardName))
	res.WriteString(fmt.Sprintf("SoC ..........: %s\n", Target()))
	res.WriteString(fmt.Sprintf("SDP ..........: %v\n", imx6ul.SDP))
	res.WriteString(fmt.Sprintf("Secure boot ..: %v\n", imx6ul.HAB()))

	if imx6ul.Native {
		res.WriteString(fmt.Sprintf("Unique ID ....: %X\n", imx6ul.UniqueID()))
	}

	return res.String(), nil
}

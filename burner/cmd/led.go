package cmd

import (
	"regexp"

	"golang.org/x/term"

	usbarmory "github.com/usbarmory/tamago/board/usbarmory/mk2"
)

func init() {
	Add(Cmd{
		Name:    "led",
		Args:    2,
		Pattern: regexp.MustCompile(`^led (white|blue) (on|off)`),
		Syntax:  "(white|blue) (on|off)",
		Help:    "LED control",
		Fn:      ledCmd,
	})
}

func ledCmd(_ *term.Terminal, arg []string) (res string, err error) {
	var on bool

	if arg[1] == "on" {
		on = true
	}

	usbarmory.LED(arg[0], on)

	return
}

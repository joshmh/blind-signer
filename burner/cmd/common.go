package cmd

import (
	"io"
	"log"
	"regexp"
	"runtime"

	"golang.org/x/term"
)

func init() {
	Add(Cmd{
		Name:    "help",
		Args:    0,
		Pattern: regexp.MustCompile(`^help`),
		Help:    "this help",
		Fn:      helpCmd,
	})

	Add(Cmd{
		Name:    "exit, quit",
		Args:    1,
		Pattern: regexp.MustCompile(`^(exit|quit)`),
		Help:    "close session",
		Fn:      exitCmd,
	})

	// The following commands are board specific, therefore their Fn
	// pointers are defined elsewhere in the respective target files.

	Add(Cmd{
		Name:    "info",
		Args:    0,
		Pattern: regexp.MustCompile(`^info`),
		Help:    "device information",
		Fn:      infoCmd,
	})

	Add(Cmd{
		Name: "reboot",
		Help: "reset device",
		Fn:   rebootCmd,
	})

}

func helpCmd(term *term.Terminal, _ []string) (string, error) {
	return Help(term), nil
}

func exitCmd(_ *term.Terminal, _ []string) (string, error) {
	log.Printf("Goodbye from %s/%s\n", runtime.GOOS, runtime.GOARCH)
	return "logout", io.EOF
}

package main

import (
	"log"

	"gitlab.lamassu.is/pazuz/blind-signer/burner/cmd"
	"gitlab.lamassu.is/pazuz/blind-signer/burner/network"
)

func main() {
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	network.Start(cmd.Console)
}

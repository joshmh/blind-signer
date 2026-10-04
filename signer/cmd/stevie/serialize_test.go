package main

import (
	"bytes"
	"testing"

	"github.com/btcsuite/btcd/btcutil/psbt"
)

// serializePsbt round-trips a packet to bytes for fixture files.
func serializePsbt(t *testing.T, p *psbt.Packet) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := p.Serialize(&buf); err != nil {
		t.Fatalf("serialize: %v", err)
	}
	return buf.Bytes()
}

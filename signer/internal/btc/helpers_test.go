package btc

import (
	"bytes"
	"os"
)

func readFile(t interface{ Fatalf(string, ...interface{}) }, path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func swapUint32(x uint32) uint32 {
	return x<<24 | (x&0xff00)<<8 | (x&0xff0000)>>8 | x>>24
}

func bytesEqual(a, b []byte) bool {
	return bytes.Equal(a, b)
}

func statFile(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

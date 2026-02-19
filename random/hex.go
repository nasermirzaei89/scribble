package random

import (
	"crypto/rand"
	"encoding/hex"
)

// Bytes generates n random bytes.
func Bytes(n int) []byte {
	bytes := make([]byte, n)

	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}

	return bytes
}

func String(n int) string {
	bytes := make([]byte, n)

	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(bytes)
}

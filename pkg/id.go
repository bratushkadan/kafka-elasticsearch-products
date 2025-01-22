package pkg

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
)

func GenId() string {
	idBytes := make([]byte, 32)
	_, _ = rand.Read(idBytes)
	dst := make([]byte, base32.StdEncoding.EncodedLen(len(idBytes)))
	base32.StdEncoding.Encode(dst, idBytes)

	return string(bytes.ToLower(bytes.TrimRight(dst, "=")))
}

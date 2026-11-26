package redact

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
)

//go:embed redact.go
var implementation []byte

func ProtocolDigest() string { sum := sha256.Sum256(implementation); return hex.EncodeToString(sum[:]) }

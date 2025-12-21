package manifest

import (
	"crypto/sha256"
	"encoding/hex"
)

// ComputeChecksum calculates a SHA-256 checksum of the provided content.
// It returns a string in the format "sha256:" followed by the lowercase
// hex-encoded hash value.
//
// The function handles nil and empty input gracefully, returning a valid
// checksum for empty content.
func ComputeChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(hash[:])
}

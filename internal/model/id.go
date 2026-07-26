package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// HashID derives a short, deterministic external ID for an Annotation from
// stable input parts (e.g. a JUnit test's classname and name, or a SARIF
// finding's rule ID, file and line). The same parts always produce the same
// ID, so republishing on the same commit updates existing annotations
// instead of creating duplicates.
func HashID(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(h[:])[:16]
}

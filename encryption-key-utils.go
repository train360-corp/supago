package supago

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

func createEncryptionKeyFile(path string) error {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Errorf("key file \"%s\" does not exist and an error occurred while trying to create it: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(hex.EncodeToString(b)), 0o600); err != nil {
		return fmt.Errorf("key file \"%s\" does not exist and an error occurred while trying to create it: %v", path, err)
	}
	return nil
}

// readEncryptionKeyFile returns the validated, canonical lowercase hex string.
// It enforces perms <= 0600 and 64 hex chars (32 bytes).
func readEncryptionKeyFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("key file %q does not exist", path)
		}
		return "", fmt.Errorf("error statting key file %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("key file %q exists but is not a regular file", path)
	}

	// Enforce strict permissions: owner rw only (0600). Reject if group/others have bits.
	if info.Mode().Perm()&0o177 != 0 {
		return "", fmt.Errorf("insecure permissions on %q: got %o, want 0600", path, info.Mode().Perm())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error reading key file %q: %w", path, err)
	}

	// Trim whitespace/newlines
	s := strings.TrimSpace(string(data))

	// Must be 64 hex chars (32 bytes)
	if len(s) != 64 {
		return "", fmt.Errorf("invalid key length in %q: got %d chars, want 64 hex chars", path, len(s))
	}

	// Validate hex and canonicalize
	raw, err := hex.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("invalid hex in key file %q: %w", path, err)
	}

	// Return canonical lowercase hex (re-encode) so callers get normalized value
	return hex.EncodeToString(raw), nil
}

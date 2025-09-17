package supago

import (
	"fmt"
	"os"
	"path/filepath"
)

type EncryptionKeyGetter func() (string, error)

type GetEncryptionKeyFrom[T any] func(T) EncryptionKeyGetter

// StaticEncryptionKey pass a static (or self-retrieved, from external means/methods) encryption key
var StaticEncryptionKey GetEncryptionKeyFrom[string] = func(key string) EncryptionKeyGetter {
	return func() (string, error) {
		return key, nil
	}
}

// EncryptionKeyFromConfig construct an encryption key from a Config
var EncryptionKeyFromConfig GetEncryptionKeyFrom[Config] = func(config Config) EncryptionKeyGetter {
	// save the file one level up from the database's own data directory
	return EncryptionKeyFromFile(filepath.Join(filepath.Dir(config.Database.DataDirectory), "pgsodium_root.key"))
}

// EncryptionKeyFromFile retrieve an encryption key from a file (at `path`)
// If `path` does not exist, EncryptionKeyFromFile will attempt to create it
var EncryptionKeyFromFile GetEncryptionKeyFrom[string] = func(path string) EncryptionKeyGetter {
	return func() (string, error) {
		if info, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				// create the file
				if err := createEncryptionKeyFile(path); err != nil {
					return "", err
				}

				// read from the created file
				return readEncryptionKeyFile(path)
			} else {
				return "", fmt.Errorf("error checking postgres pgsodium key file \"%s\" exists: %v", path, err)
			}
		} else if info.IsDir() {
			return "", fmt.Errorf("postgres pgsodium key file \"%s\" exists but is not a file", path)
		} else {
			// read from the file
			return readEncryptionKeyFile(path)
		}
	}
}

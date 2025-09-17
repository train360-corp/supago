package supago

import (
	"errors"
	"fmt"
)

type configBuilder struct {
	platform            *string
	encryptionKeyGetter EncryptionKeyGetter
}

func ConfigBuilder() *configBuilder {
	return &configBuilder{}
}

func (b *configBuilder) Platform(platform string) *configBuilder {
	b.platform = &platform
	return b
}

func (b *configBuilder) EncryptionKey(key string) *configBuilder {
	b.encryptionKeyGetter = StaticEncryptionKey(key)
	return b
}

func (b *configBuilder) GetEncryptionKeyUsing(getter EncryptionKeyGetter) *configBuilder {
	b.encryptionKeyGetter = getter
	return b
}

func (b *configBuilder) Build() *Config {
	cfg, err := b.BuildE()
	if err != nil {
		panic(err)
	}
	return cfg
}

func (b *configBuilder) BuildE() (*Config, error) {

	if b.platform == nil {
		return nil, errors.New("no platform specified (use the .Platform(...) method to specify one)")
	} else if !IsValidPlatformName(*b.platform) {
		return nil, errors.New("invalid platform specified (use the .Platform(...) method to specify a proper one)")
	}
	cfg, err := newBaseConfig(*b.platform)
	if err != nil {
		return nil, fmt.Errorf("could not create config: %w", err)
	}

	// use default generator from the config itself
	if b.encryptionKeyGetter == nil {
		b.encryptionKeyGetter = EncryptionKeyFromConfig(*cfg)
	}

	// get the key
	if key, err := b.encryptionKeyGetter(); err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	} else {

		if _, err := IsValidEncryptionKey(key); err != nil {
			return nil, fmt.Errorf("failed to validate encryption key: %v", err)
		}

		cfg.Keys.PgSodiumEncryption = key
		return cfg, nil
	}
}

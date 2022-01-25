package cryptotools

import (
	"os"

	"github.com/google/uuid"
	libp2pcrypto "github.com/libp2p/go-libp2p-core/crypto"

	"happystoic/p2pnetwork/pkg/config"
)

func GetPrivateKey(conf *config.IdentityConfig) (key libp2pcrypto.PrivKey, err error) {
	if conf.GenerateNewKey {
		key, err = generateKey()
		if err != nil {
			return nil, err
		}
	} else {
		key, err = loadKeyFromFile(conf.LoadKeyFromFile)
		if err != nil {
			return nil, err
		}
	}

	if conf.SaveKeyToFile != "" {
		err := saveKeyToFile(conf.SaveKeyToFile, key)
		if err != nil {
			return nil, err
		}
	}
	return key, nil
}

func generateKey() (libp2pcrypto.PrivKey, error) {
	priv, _, err := libp2pcrypto.GenerateKeyPair(
		libp2pcrypto.Ed25519, // Select your key type. Ed25519 are nice short
		-1,                   // Select key length when possible (i.e. RSA).
	)
	return priv, err
}

func loadKeyFromFile(file string) (libp2pcrypto.PrivKey, error) {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	key, err := libp2pcrypto.UnmarshalPrivateKey(bytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func saveKeyToFile(file string, key libp2pcrypto.PrivKey) error {
	bytes, err := libp2pcrypto.MarshalPrivateKey(key)
	if err != nil {
		return err
	}
	err = os.WriteFile(file, bytes, 0644)
	if err != nil {
		return err
	}
	return nil
}

func GenerateUUID() string {
	return uuid.New().String()
}

package rsautil

import (
	"crypto/rsa"
	"os"
)

func ReadPublicKey(publicKeyFilePath string) (*rsa.PublicKey, error) {
	bytes, err := os.ReadFile(publicKeyFilePath)
	if err != nil {
		return nil, err
	}

	pubKey, err := BytesToPublicKey(bytes)
	if err != nil {
		return nil, err
	}

	return pubKey, nil
}

func ReadPrivateKey(privateKeyFilePath string) (*rsa.PrivateKey, error) {
	bytes, err := os.ReadFile(privateKeyFilePath)
	if err != nil {
		return nil, err
	}

	privKey, err := BytesToPrivateKey(bytes)
	if err != nil {
		return nil, err
	}

	return privKey, nil
}

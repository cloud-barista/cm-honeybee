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

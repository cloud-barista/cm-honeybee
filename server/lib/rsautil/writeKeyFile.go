package rsautil

import (
	"os"
)

func GeneratePrivateKeyAndPublicKey(bits int, privateKeyFilePath string, publicKeyFilePath string) error {
	privKey, pubKey, err := GenerateKeyPair(bits)
	if err != nil {
		return err
	}

	privKeyData := PrivateKeyToBytes(privKey)

	pubKeyData, err := PublicKeyToBytes(pubKey)
	if err != nil {
		return err
	}

	err = os.WriteFile(privateKeyFilePath, privKeyData, 0600)
	if err != nil {
		return err
	}

	err = os.WriteFile(publicKeyFilePath, pubKeyData, 0644)
	if err != nil {
		return err
	}

	return nil
}

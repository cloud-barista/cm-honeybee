package common

import "crypto/rsa"

var PrivateKeyFileName = "honeybee.key"
var PublicKeyFileName = "honeybee.pub"

var PubKey *rsa.PublicKey

// PrivKey is loaded at startup and used to decrypt secrets that honeybee stored
// encrypted at rest (e.g. CSP credentials registered transiently with cb-spider).
var PrivKey *rsa.PrivateKey

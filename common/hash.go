package common

import (
	"crypto/sha256"

	"golang.org/x/crypto/sha3"
)

func HashFuncFactory(hash string) func(data ...[]byte) []byte {
	switch hash {
	case "sha2":
		return Sha2
	case "sha3":
		return Sha3
	default:
		return Sha2
	}
}

func Sha2(data ...[]byte) []byte {
	hashes := sha256.New()
	for i := 0; i < len(data); i++ {
		hashes.Write(data[i])
	}
	return hashes.Sum(nil)
}

func Sha3(data ...[]byte) []byte {
	hashes := sha3.New256()
	for i := 0; i < len(data); i++ {
		hashes.Write(data[i])
	}
	return hashes.Sum(nil)
}

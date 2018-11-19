package types

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/anaskhan96/base58check"
)

type Address = []byte

const AddressVersion = 0x76

//const PrivKeyVersion = 0xAA

type AddressEncoder interface {
	EncodeAddress(addr Address) (string, error)
}

type AddressDecoder interface {
	DecodeAddress(encodedAddr string) (Address, error)
}

type Base58Address struct {
}

func (address Base58Address) EncodeAddress(addr Address) (string, error) {
	return base58check.Encode(fmt.Sprintf("%x", AddressVersion), hex.EncodeToString(addr))
}

func (address Base58Address) DecodeAddress(encodedAddr string) (Address, error) {
	decodedString, err := base58check.Decode(encodedAddr)
	if err != nil {
		return nil, err
	}
	decodedBytes, err := hex.DecodeString(decodedString)
	if err != nil {
		return nil, err
	}
	version := decodedBytes[0]
	if version != AddressVersion {
		return nil, errors.New("invalid address version")
	}
	decoded := decodedBytes[1:]
	return decoded, nil
}

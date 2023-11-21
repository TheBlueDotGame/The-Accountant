package serializer

import "github.com/mr-tron/base58"

// Base58Encode encodes byte array to base58 string.
func Base58Encode(input []byte) []byte {
	encode := base58.Encode(input)

	return []byte(encode)
}

// Base58Decode decodes base58 string to byte array.
func Base58Decode(input []byte) ([]byte, error) {
	decode, err := base58.Decode(string(input[:]))
	if err != nil {
		return nil, err
	}

	return decode, nil
}

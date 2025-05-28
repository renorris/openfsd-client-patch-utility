package patch

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"errors"
	"fmt"
)

// Decrypts ciphertext using Triple DES in ECB mode
func tripleDESDecrypt(ciphertext []byte, key []byte) (plaintext []byte, err error) {
	// Create a new DES cipher
	var block cipher.Block
	if block, err = des.NewTripleDESCipher(key); err != nil {
		return nil, err
	}

	// Initialize byte array for the plaintext
	plaintext = make([]byte, len(ciphertext))

	// decrypt in ECB mode
	for i := 0; i < len(ciphertext); i += block.BlockSize() {
		block.Decrypt(plaintext[i:i+block.BlockSize()], ciphertext[i:i+block.BlockSize()])
	}

	return pkcs7strip(plaintext, block.BlockSize())
}

// Encrypts plaintext using Triple DES in ECB mode
func tripleDESEncrypt(plaintext []byte, key []byte) (ciphertext []byte, err error) {
	// Create a new DES cipher
	var block cipher.Block
	if block, err = des.NewTripleDESCipher(key); err != nil {
		return nil, err
	}

	// Pad the plaintext to be a multiple of the block size
	paddedPlaintext, err := pkcs7pad(plaintext, block.BlockSize())
	if err != nil {
		return nil, err
	}

	// Initialize byte array for the ciphertext
	ciphertext = make([]byte, len(paddedPlaintext))

	// encrypt in ECB mode
	for i := 0; i < len(paddedPlaintext); i += block.BlockSize() {
		block.Encrypt(ciphertext[i:i+block.BlockSize()], paddedPlaintext[i:i+block.BlockSize()])
	}

	return ciphertext, nil
}

// pkcs7strip remove pkcs7 padding
func pkcs7strip(data []byte, blockSize int) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("pkcs7: data is empty")
	}
	if length%blockSize != 0 {
		return nil, errors.New("pkcs7: data is not block-aligned")
	}

	padLen := int(data[length-1])
	ref := bytes.Repeat([]byte{byte(padLen)}, padLen)
	if padLen > blockSize || padLen == 0 || !bytes.HasSuffix(data, ref) {
		return nil, errors.New("pkcs7: invalid padding")
	}

	return data[:length-padLen], nil
}

// pkcs7pad add pkcs7 padding
func pkcs7pad(data []byte, blockSize int) ([]byte, error) {
	if blockSize <= 1 || blockSize >= 256 {
		return nil, fmt.Errorf("pkcs7: invalid block size %d", blockSize)
	}

	padLen := blockSize - len(data)%blockSize
	padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
	return append(data, padding...), nil
}

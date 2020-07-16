package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/sha3"
)

// Save saves the seed to a dat file
func saveSeed(seed []byte, password string, file string) error {
	// Overwriting a seed file may cause loss of funds
	if _, err := os.Stat(file); err == nil {
		return ErrSeedFileExists
	}

	digest := sha3.Sum256([]byte(password))

	c, err := aes.NewCipher(digest[:])
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	return ioutil.WriteFile(file, gcm.Seal(nonce, nonce, seed, nil), 0600)
}

//Modified from https://tutorialedge.net/golang/go-encrypt-decrypt-aes-tutorial/
func fetchSeed(password string, file string) ([]byte, error) {

	digest := sha3.Sum256([]byte(password))

	ciphertext, err := ioutil.ReadFile(file) //nolint
	if err != nil {
		return nil, err
	}

	c, err := aes.NewCipher(digest[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, err
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

package rpc

import (
	"crypto/ed25519"
	"encoding/json"
)

// AuthToken is what we put in the authorization header
type AuthToken struct {
	AccessToken string `json:"access-token"`
	Time        int64  `json:"time"`
	Sig         []byte `json:"signature,omitempty"`
}

// AsSignable returns the signable payload of the struct
func (a *AuthToken) AsSignable() ([]byte, error) {
	clone := &AuthToken{
		AccessToken: a.AccessToken,
		Time:        a.Time,
	}

	return json.Marshal(clone)
}

// Verify the token using ed25519
func (a *AuthToken) Verify(edPk ed25519.PublicKey) bool {
	if a.Sig == nil {
		return false
	}
	payload, err := a.AsSignable()
	if err != nil {
		// TODO: log
		return false
	}
	return ed25519.Verify(edPk, payload, a.Sig)
}

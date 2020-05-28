package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var (
	errAlreadyLoggedIn   = errors.New("login already there")
	errAccessDenied      = errors.New("access denied")
	errInvalidToken      = errors.New("token is invalid")
	errMissingMetadata   = errors.New("missing metadata")
	errSigMethodMismatch = errors.New("wrong signature scheme used")
)

// JWTManager is a simple struct for managing the JWT token lifecycle
type JWTManager struct {
	pk        ed25519.PublicKey
	sk        ed25519.PrivateKey
	tDuration time.Duration
}

// ClientClaims is a simple extension of jwt.StandardClaims that includes the
// ED25519 public key of a client
type ClientClaims struct {
	jwt.StandardClaims
	ClientEdPk string `json:"client-edpk"`
}

// NewJWTManager creates a JWTMnager
func NewJWTManager(duration time.Duration) (*JWTManager, error) {
	pk, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &JWTManager{
		pk:        pk,
		sk:        sk,
		tDuration: duration,
	}, nil
}

// Generate a session token used by the client to authenticate
func (m *JWTManager) Generate(edPkBase64 string) (string, error) {
	claims := ClientClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(m.tDuration).Unix(),
		},
		ClientEdPk: edPkBase64,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	return token.SignedString(m.sk)
}

// Verify the session token
func (m *JWTManager) Verify(accessToken string) (*ClientClaims, error) {
	token, err := jwt.ParseWithClaims(
		accessToken,
		&ClientClaims{},
		func(t *jwt.Token) (interface{}, error) {
			_, ok := t.Method.(*jwt.SigningMethodECDSA)
			if !ok {
				return nil, errSigMethodMismatch
			}
			return m.sk, nil
		},
	)

	if err != nil {
		return nil, errInvalidToken
	}

	claims, ok := token.Claims.(*ClientClaims)
	if !ok {
		return nil, errInvalidToken
	}
	return claims, nil
}

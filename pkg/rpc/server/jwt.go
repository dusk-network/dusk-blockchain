package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var (
	errAccessDenied      = errors.New("access denied")
	errInvalidToken      = errors.New("token is invalid")
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

func init() {
	method := &SigningMethodEdDSA{}
	jwt.RegisterSigningMethod(method.Alg(), func() jwt.SigningMethod {
		return method
	})
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

	token := jwt.NewWithClaims(&SigningMethodEdDSA{}, claims)
	return token.SignedString(m.sk)
}

// Verify the session token
func (m *JWTManager) Verify(accessToken string) (*ClientClaims, error) {
	token, err := jwt.ParseWithClaims(
		accessToken,
		&ClientClaims{},
		func(t *jwt.Token) (interface{}, error) {
			_, ok := t.Method.(*SigningMethodEdDSA)
			if !ok {
				return nil, errSigMethodMismatch
			}
			return m.pk, nil
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

// SigningMethodEdDSA is the encryption method based on ed25519. It is demanded
// by the JWT library and implements jwt.SigningMethod interface
type SigningMethodEdDSA struct{}

// ErrEdDSAVerification is the error triggered when verification of ed25519
// signatures within the JWT is not successful
var ErrEdDSAVerification = errors.New("crypto/ed25519: verification error")

// Alg complies with jwt.SigningMethod interface
func (m *SigningMethodEdDSA) Alg() string {
	return "EdDSA"
}

// Verify complies with jwt.SigningMethod interface for signature verification
func (m *SigningMethodEdDSA) Verify(signingString string, signature string, key interface{}) error {
	sig, err := jwt.DecodeSegment(signature)
	if err != nil {
		return err
	}

	ed25519Key, ok := key.(ed25519.PublicKey)
	if !ok {
		return jwt.ErrInvalidKeyType
	}

	if len(ed25519Key) != ed25519.PublicKeySize {
		return jwt.ErrInvalidKey
	}

	if ok := ed25519.Verify(ed25519Key, []byte(signingString), sig); !ok {
		return ErrEdDSAVerification
	}

	return nil
}

// Sign complies with jwt.SigningMethod interface for signing
func (m *SigningMethodEdDSA) Sign(signingString string, key interface{}) (str string, err error) {
	ed25519Key, ok := key.(ed25519.PrivateKey)
	if !ok {
		return "", jwt.ErrInvalidKeyType
	}

	if len(ed25519Key) != ed25519.PrivateKeySize {
		return "", jwt.ErrInvalidKey
	}

	// Sign the string and return the encoded result
	sig := ed25519.Sign(ed25519Key, []byte(signingString))
	return jwt.EncodeSegment(sig), nil
}

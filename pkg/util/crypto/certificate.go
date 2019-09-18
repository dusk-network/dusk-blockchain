package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"
)

// GenerateX509Certificate will create a new x509 self signed certificate with ed25519 private key. It will expire in 1 year
func GenerateX509Certificate() (*x509.Certificate, *ed25519.PrivateKey, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 24 * 365)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Dusk Network"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		priv.Public().(ed25519.PublicKey),
		priv)
	if err != nil {
		return nil, nil, err
	}

	generatedCert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, nil, err
	}

	return generatedCert, &priv, nil
}

// GenerateTLSCertificate will produce a tls certificate from GenerateX509Certificate
func GenerateTLSCertificate() (*tls.Certificate, error) {
	x509Cert, priv, err := GenerateX509Certificate()
	if err != nil {
		return nil, err
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{x509Cert.Raw},
		PrivateKey:  priv,
		Leaf:        x509Cert,
	}

	return &tlsCert, nil
}

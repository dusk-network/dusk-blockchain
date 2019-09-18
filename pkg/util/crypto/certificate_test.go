package crypto

import (
	"log"
	"testing"
)

func TestGenerateX509Certificate(t *testing.T) {
	_, _, err := GenerateX509Certificate()
	if err != nil {
		log.Fatalf("Failed to generate x509 certificate: %s", err)
	}
}

func TestGenerateTLSCertificate(t *testing.T) {
	_, err := GenerateTLSCertificate()
	if err != nil {
		log.Fatalf("Failed to generate TLS certificate: %s", err)
	}
}

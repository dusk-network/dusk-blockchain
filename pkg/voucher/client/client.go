package client

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/voucher/client/request"
)

// Client will connect to a voucher, perform Requests, and extract Responses
type Client struct {
	addr string
	conn *tls.Conn
}

// NewClient receives a socket (ip:port) as parameter, and return an instance of Client
func NewClient(addr string) (*Client, error) {
	c := Client{addr: addr, conn: nil}
	_, err := c.Dial()
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Dial will establish and return a connection
// to a voucher in the provided address, using TLS.
// The certificate will be self-signed
func (c *Client) Dial() (*tls.Conn, error) {
	generatedCert, err := generateCertificate()
	if err != nil {
		return nil, err
	}

	roots := x509.NewCertPool()
	roots.AddCert(generatedCert)

	conn, err := tls.Dial("tcp", c.addr, &tls.Config{
		RootCAs:            roots,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, err
	}

	c.conn = conn

	return conn, nil
}

// Close will close the connection, if active. It is idempotent.
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// Read will call the inner tls.Conn implementation of io.Reader
func (c *Client) Read(p []byte) (int, error) {
	return c.conn.Read(p)
}

// Write will call the inner tls.Conn implementation of io.Writer
func (c *Client) Write(p []byte) (int, error) {
	return c.conn.Write(p)
}

// Execute will perform a request on the connected voucher, and return the response
func (c *Client) Execute(r request.Request) error {
	buf := []byte{byte(*r.Operation())}
	buf = append(buf, *r.Payload()...)

	_, err := c.conn.Write(buf)
	if err != nil {
		return err
	}

	// Get the response status
	buf = make([]byte, 1)
	_, err = io.ReadFull(c.conn, buf[:])
	if err != nil {
		return err
	}
	s, err := request.ByteToStatus(buf[0])
	if err != nil {
		return err
	}

	if s != request.Success {
		description := ""
		buf = make([]byte, 2)
		_, err = io.ReadFull(c.conn, buf[:])
		if err == nil {
			size := binary.LittleEndian.Uint16(buf)
			buf = make([]byte, size)
			_, err = io.ReadFull(c.conn, buf[:])
			if err == nil {
				description = string(buf)
			}
		}
		return fmt.Errorf("The request was not successful. %s", description)
	}

	return nil
}

// generateCertificate creates a new self-signed
// certificate, using ed25519.
func generateCertificate() (*x509.Certificate, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 24 * 365)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	generatedCert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, err
	}

	return generatedCert, nil
}

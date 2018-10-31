package main

import (
	"strings"
	"testing"

	"github.com/toghrulmaharramov/dusk-go/crypto/base58"
	"github.com/toghrulmaharramov/dusk-go/crypto/hash"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/rpc"
)

func TestMethod(t *testing.T) {
	cfg, err := StartServer()
	if err != nil {
		t.Fail()
	}

	// Make command
	ping, err := MarshalCmd("ping", []string{})
	if err != nil {
		t.Fail()
	}

	// Send it off..
	resp, err := SendPostRequest(ping, cfg)
	if err != nil {
		t.Fail()
	}

	// Make sure we got the right response
	assert.Equal(t, resp.Result, "pong")
}

func TestMethodWithParams(t *testing.T) {
	cfg, err := StartServer()
	if err != nil {
		t.Fail()
	}

	// Make command
	hashCmd, err := MarshalCmd("hash", []string{"foo"})
	if err != nil {
		t.Fail()
	}

	// Send off
	resp, err := SendPostRequest(hashCmd, cfg)
	if err != nil {
		t.Fail()
	}

	// Make another command
	hashCmd2, err := MarshalCmd("hash", []string{"bar", "baz"})
	if err != nil {
		t.Fail()
	}

	// Send off
	resp2, err := SendPostRequest(hashCmd2, cfg)
	if err != nil {
		t.Fail()
	}

	// Trim whitespaces and split results
	foo := strings.TrimSpace(resp.Result)
	s := strings.Split(resp2.Result, " ")

	// Hash words for comparison
	fooHash, _ := hash.Sha3256([]byte("foo"))
	fooText := base58.Base58Encoding(fooHash)
	barHash, _ := hash.Sha3256([]byte("bar"))
	barText := base58.Base58Encoding(barHash)
	bazHash, _ := hash.Sha3256([]byte("baz"))
	bazText := base58.Base58Encoding(bazHash)

	// Compare
	assert.Equal(t, foo, fooText)
	assert.Equal(t, s[0], barText)
	assert.Equal(t, s[1], bazText)
}

func TestAdminRestriction(t *testing.T) {
	// Discard config, we don't need it
	if _, err := StartServer(); err != nil {
		t.Fail()
	}

	// Make new config with different credentials
	cfg := rpc.Config{
		RPCUser: "dusk456",
		RPCPass: "password",
		RPCPort: "9999",
	}

	// Make admin command and send it
	stopNode, err := MarshalCmd("stopnode", []string{})
	if err != nil {
		t.Fail()
	}

	// This should give us an error response
	resp, err := SendPostRequest(stopNode, &cfg)
	if err != nil {
		t.Fail()
	}

	t.Log(resp.Error)

	// Result will be "error" if something went wrong
	assert.Equal(t, resp.Result, "error")
}

// Convenience function for setting up RPC server
func StartServer() (*rpc.Config, error) {
	cfg := rpc.Config{}
	if err := cfg.Load(); err != nil {
		return nil, err
	}

	srv, err := rpc.NewRPCServer(&cfg)
	if err != nil {
		return nil, err
	}

	if err := srv.Start(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

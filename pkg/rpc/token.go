// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package rpc

import (
	"crypto/ed25519"
	"encoding/json"

	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/hashset"
)

const servicePrefix = "/node.Auth/"

// CreateSessionRoute is the RPC to create a session
const CreateSessionRoute = servicePrefix + "CreateSession"

// DropSessionRoute is the RPC to create a session
const DropSessionRoute = servicePrefix + "DropSession"

// StatusRoute is the RPC to inquiry the status of the wallet
const StatusRoute = servicePrefix + "Status"

// OpenRoutes is the set of RPC that do not require session authentication
var OpenRoutes = hashset.New()

func init() {
	OpenRoutes.Add([]byte(CreateSessionRoute))
	OpenRoutes.Add([]byte(StatusRoute))
}

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

// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package blindbid

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// Bid represents a blind bid, made by a block generator.
type Bid struct {
	EncryptedData  []byte               `json:"encrypted_data"`
	HashedSecret   []byte               `json:"hashed_secret"`
	Nonce          []byte               `json:"nonce"`
	StealthAddress *keys.StealthAddress `json:"stealth_address"`
	Commitment     []byte               `json:"commitment"`
	Eligibility    []byte               `json:"eligibility"`
	Expiration     []byte               `json:"expiration"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (b *Bid) Copy() *Bid {
	encData := make([]byte, len(b.EncryptedData))
	hashedSecret := make([]byte, len(b.HashedSecret))
	nonce := make([]byte, len(b.Nonce))
	commitment := make([]byte, len(b.Commitment))
	eligibility := make([]byte, len(b.Eligibility))
	expiration := make([]byte, len(b.Expiration))

	copy(encData, b.EncryptedData)
	copy(hashedSecret, b.HashedSecret)
	copy(nonce, b.Nonce)
	copy(commitment, b.Commitment)
	copy(eligibility, b.Eligibility)
	copy(expiration, b.Expiration)

	return &Bid{
		EncryptedData:  encData,
		HashedSecret:   hashedSecret,
		Nonce:          nonce,
		StealthAddress: b.StealthAddress.Copy(),
		Commitment:     commitment,
		Eligibility:    eligibility,
		Expiration:     expiration,
	}
}

// MBid copies the Bid structure into the Rusk equivalent.
func MBid(r *rusk.Bid, f *Bid) {
	copy(r.EncryptedData, f.EncryptedData)
	copy(r.HashedSecret, f.HashedSecret)
	copy(r.Nonce, f.Nonce)
	keys.MStealthAddress(r.StealthAddress, f.StealthAddress)
	copy(r.Commitment, f.Commitment)
	// XXX: fix typo in rusk-schema
	copy(r.Elegibility, f.Eligibility)
	copy(r.Expiration, f.Expiration)
}

// UBid copies the Rusk Bid structure into the native equivalent.
func UBid(r *rusk.Bid, f *Bid) {
	copy(f.EncryptedData, r.EncryptedData)
	copy(f.HashedSecret, r.HashedSecret)
	copy(f.Nonce, r.Nonce)
	keys.UStealthAddress(r.StealthAddress, f.StealthAddress)
	copy(f.Commitment, r.Commitment)
	// XXX: fix typo in rusk-schema
	copy(f.Eligibility, r.Elegibility)
	copy(f.Expiration, r.Expiration)
}

// MarshalBid writes the Bid struct into a bytes.Buffer.
func MarshalBid(r *bytes.Buffer, f *Bid) error {
	if err := encoding.WriteVarBytes(r, f.EncryptedData); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.HashedSecret); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Nonce); err != nil {
		return err
	}

	if err := keys.MarshalStealthAddress(r, f.StealthAddress); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Commitment); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Eligibility); err != nil {
		return err
	}

	return encoding.Write256(r, f.Expiration)
}

// UnmarshalBid reads a Bid struct from a bytes.Buffer.
func UnmarshalBid(r *bytes.Buffer, f *Bid) error {
	if err := encoding.ReadVarBytes(r, &f.EncryptedData); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.HashedSecret); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.Nonce); err != nil {
		return err
	}

	if err := keys.UnmarshalStealthAddress(r, f.StealthAddress); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.Commitment); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.Eligibility); err != nil {
		return err
	}

	return encoding.Read256(r, f.Expiration)
}

// BidTransactionRequest is used to construct a Bid transaction.
type BidTransactionRequest struct {
	K                    []byte               `json:"k"`
	Value                uint64               `json:"value"`
	Secret               []byte               `json:"secret"`
	StealthAddress       *keys.StealthAddress `json:"stealth_address"`
	Seed                 []byte               `json:"seed"`
	LatestConsensusRound uint64               `json:"latest_consensus_round"`
	LatestConsensusStep  uint64               `json:"latest_consensus_step"`
	GasLimit             uint64               `json:"gas_limit"`
	GasPrice             uint64               `json:"gas_price"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (b *BidTransactionRequest) Copy() *BidTransactionRequest {
	k := make([]byte, len(b.K))
	secret := make([]byte, len(b.Secret))
	seed := make([]byte, len(b.Seed))

	copy(k, b.K)
	copy(secret, b.Secret)
	copy(seed, b.Seed)

	return &BidTransactionRequest{
		K:                    k,
		Value:                b.Value,
		Secret:               secret,
		StealthAddress:       b.StealthAddress.Copy(),
		Seed:                 seed,
		LatestConsensusRound: b.LatestConsensusRound,
		LatestConsensusStep:  b.LatestConsensusStep,
		GasLimit:             b.GasLimit,
		GasPrice:             b.GasPrice,
	}
}

// MBidTransactionRequest copies the BidTransactionRequest structure into the Rusk equivalent.
func MBidTransactionRequest(r *rusk.BidTransactionRequest, f *BidTransactionRequest) {
	copy(r.K, f.K)
	r.Value = f.Value
	copy(r.Secret, f.Secret)
	keys.MStealthAddress(r.StealthAddress, f.StealthAddress)
	copy(r.Seed, f.Seed)
	r.LatestConsensusRound = f.LatestConsensusRound
	r.LatestConsensusStep = f.LatestConsensusStep
	r.GasLimit = f.GasLimit
	r.GasPrice = f.GasPrice
}

// UBidTransactionRequest copies the Rusk BidTransactionRequest structure into the native equivalent.
func UBidTransactionRequest(r *rusk.BidTransactionRequest, f *BidTransactionRequest) {
	copy(f.K, r.K)
	f.Value = r.Value
	copy(f.Secret, r.Secret)
	keys.UStealthAddress(r.StealthAddress, f.StealthAddress)
	copy(f.Seed, r.Seed)
	f.LatestConsensusRound = r.LatestConsensusRound
	f.LatestConsensusStep = r.LatestConsensusStep
	f.GasLimit = r.GasLimit
	f.GasPrice = r.GasPrice
}

// MarshalBidTransactionRequest writes the BidTransactionRequest struct into a bytes.Buffer.
func MarshalBidTransactionRequest(r *bytes.Buffer, f *BidTransactionRequest) error {
	if err := encoding.Write256(r, f.K); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, f.Value); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Secret); err != nil {
		return err
	}

	if err := keys.MarshalStealthAddress(r, f.StealthAddress); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Seed); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, f.LatestConsensusRound); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, f.LatestConsensusStep); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, f.GasLimit); err != nil {
		return err
	}

	return encoding.WriteUint64LE(r, f.GasPrice)
}

// UnmarshalBidTransactionRequest reads a BidTransactionRequest struct from a bytes.Buffer.
func UnmarshalBidTransactionRequest(r *bytes.Buffer, f *BidTransactionRequest) error {
	if err := encoding.Read256(r, f.K); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &f.Value); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.Secret); err != nil {
		return err
	}

	if err := keys.UnmarshalStealthAddress(r, f.StealthAddress); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.Seed); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &f.LatestConsensusRound); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &f.LatestConsensusStep); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &f.GasLimit); err != nil {
		return err
	}

	return encoding.ReadUint64LE(r, &f.GasPrice)
}

// FindBidRequest is used to find a specific Bid belonging to a public key.
type FindBidRequest struct {
	StealthAddress *keys.StealthAddress `json:"stealth_address"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (f *FindBidRequest) Copy() *FindBidRequest {
	return &FindBidRequest{
		StealthAddress: f.StealthAddress.Copy(),
	}
}

// MFindBidRequest copies the FindBidRequest structure into the Rusk equivalent.
func MFindBidRequest(r *rusk.FindBidRequest, f *FindBidRequest) {
	keys.MStealthAddress(r.StealthAddress, f.StealthAddress)
}

// UFindBidRequest copies the Rusk FindBidRequest structure into the native equivalent.
func UFindBidRequest(r *rusk.FindBidRequest, f *FindBidRequest) {
	keys.UStealthAddress(r.StealthAddress, f.StealthAddress)
}

// MarshalFindBidRequest writes the FindBidRequest struct into a bytes.Buffer.
func MarshalFindBidRequest(r *bytes.Buffer, f *FindBidRequest) error {
	return keys.MarshalStealthAddress(r, f.StealthAddress)
}

// UnmarshalFindBidRequest reads a FindBidRequest struct from a bytes.Buffer.
func UnmarshalFindBidRequest(r *bytes.Buffer, f *FindBidRequest) error {
	return keys.UnmarshalStealthAddress(r, f.StealthAddress)
}

// BidList is the collection of all blind bids.
type BidList struct {
	BidList     []*Bid   `json:"bid_list"`
	BidHashList [][]byte `json:"bid_hash_list"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (b *BidList) Copy() *BidList {
	bidList := make([]*Bid, len(b.BidList))
	for i := range bidList {
		bidList[i] = b.BidList[i].Copy()
	}

	bidHashList := make([][]byte, len(b.BidHashList))
	for i := range bidHashList {
		bidHashList[i] = make([]byte, 32)
		copy(bidHashList[i], b.BidHashList[i])
	}

	return &BidList{
		BidList:     bidList,
		BidHashList: bidHashList,
	}
}

// MBidList copies the BidList structure into the Rusk equivalent.
func MBidList(r *rusk.BidList, f *BidList) {
	r.BidList = make([]*rusk.Bid, len(f.BidList))
	for i, bid := range r.BidList {
		MBid(bid, f.BidList[i])
	}

	r.BidHashList = make([][]byte, len(f.BidHashList))
	for i := range r.BidHashList {
		r.BidHashList[i] = make([]byte, 32)
		copy(r.BidHashList[i], f.BidHashList[i])
	}
}

// UBidList copies the Rusk BidList structure into the native equivalent.
func UBidList(r *rusk.BidList, f *BidList) {
	f.BidList = make([]*Bid, len(r.BidList))
	for i, bid := range f.BidList {
		UBid(r.BidList[i], bid)
	}

	f.BidHashList = make([][]byte, len(r.BidHashList))
	for i := range f.BidHashList {
		f.BidHashList[i] = make([]byte, 32)
		copy(f.BidHashList[i], r.BidHashList[i])
	}
}

// MarshalBidList writes the BidList struct into a bytes.Buffer.
func MarshalBidList(r *bytes.Buffer, f *BidList) error {
	if err := encoding.WriteVarInt(r, uint64(len(f.BidList))); err != nil {
		return err
	}

	for _, bid := range f.BidList {
		if err := MarshalBid(r, bid); err != nil {
			return err
		}
	}

	if err := encoding.WriteVarInt(r, uint64(len(f.BidHashList))); err != nil {
		return err
	}

	for _, bidHash := range f.BidHashList {
		if err := encoding.Write256(r, bidHash); err != nil {
			return err
		}
	}

	return nil
}

// UnmarshalBidList reads a BidList struct from a bytes.Buffer.
func UnmarshalBidList(r *bytes.Buffer, f *BidList) error {
	lenBidList, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	f.BidList = make([]*Bid, lenBidList)
	for _, bid := range f.BidList {
		if err = UnmarshalBid(r, bid); err != nil {
			return err
		}
	}

	lenBidHashList, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	f.BidHashList = make([][]byte, lenBidHashList)
	for _, bidHash := range f.BidHashList {
		if err = encoding.Read256(r, bidHash); err != nil {
			return err
		}
	}

	return nil
}

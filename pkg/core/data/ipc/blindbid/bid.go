package blindbid

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// Bid represents a blind bid, made by a block generator.
type Bid struct {
	EncryptedData *common.PoseidonCipher   `json:"encrypted_data"`
	HashedSecret  *common.BlsScalar        `json:"hashed_secret"`
	Nonce         *common.BlsScalar        `json:"nonce"`
	PkR           *keys.StealthAddress     `json:"stealth_address"`
	Commitment    *common.JubJubCompressed `json:"commitment"`
	EligibilityTS *common.BlsScalar        `json:"eligibility_ts"`
	ExpirationTS  *common.BlsScalar        `json:"expiration_ts"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (b *Bid) Copy() *Bid {
	return &Bid{
		EncryptedData: b.EncryptedData.Copy(),
		HashedSecret:  b.HashedSecret.Copy(),
		Nonce:         b.Nonce.Copy(),
		PkR:           b.PkR.Copy(),
		Commitment:    b.Commitment.Copy(),
		EligibilityTS: b.EligibilityTS.Copy(),
		ExpirationTS:  b.ExpirationTS.Copy(),
	}
}

// MBid copies the Bid structure into the Rusk equivalent.
func MBid(r *rusk.Bid, f *Bid) {
	common.MPoseidonCipher(r.EncryptedData, f.EncryptedData)
	common.MBlsScalar(r.HashedSecret, f.HashedSecret)
	common.MBlsScalar(r.Nonce, f.Nonce)
	keys.MStealthAddress(r.PkR, f.PkR)
	common.MJubJubCompressed(r.Commitment, f.Commitment)
	// XXX: fix typo in rusk-schema
	common.MBlsScalar(r.ElegibilityTs, f.EligibilityTS)
	common.MBlsScalar(r.ExpirationTs, f.ExpirationTS)
}

// UBid copies the Rusk Bid structure into the native equivalent.
func UBid(r *rusk.Bid, f *Bid) {
	common.UPoseidonCipher(r.EncryptedData, f.EncryptedData)
	common.UBlsScalar(r.HashedSecret, f.HashedSecret)
	common.UBlsScalar(r.Nonce, f.Nonce)
	keys.UStealthAddress(r.PkR, f.PkR)
	common.UJubJubCompressed(r.Commitment, f.Commitment)
	// XXX: fix typo in rusk-schema
	common.UBlsScalar(r.ElegibilityTs, f.EligibilityTS)
	common.UBlsScalar(r.ExpirationTs, f.ExpirationTS)
}

// MarshalBid writes the Bid struct into a bytes.Buffer.
func MarshalBid(r *bytes.Buffer, f *Bid) error {
	if err := common.MarshalPoseidonCipher(r, f.EncryptedData); err != nil {
		return err
	}

	if err := common.MarshalBlsScalar(r, f.HashedSecret); err != nil {
		return err
	}

	if err := common.MarshalBlsScalar(r, f.Nonce); err != nil {
		return err
	}

	if err := keys.MarshalStealthAddress(r, f.PkR); err != nil {
		return err
	}

	if err := common.MarshalJubJubCompressed(r, f.Commitment); err != nil {
		return err
	}

	if err := common.MarshalBlsScalar(r, f.EligibilityTS); err != nil {
		return err
	}

	return common.MarshalBlsScalar(r, f.ExpirationTS)
}

// UnmarshalBid reads a Bid struct from a bytes.Buffer.
func UnmarshalBid(r *bytes.Buffer, f *Bid) error {
	if err := common.UnmarshalPoseidonCipher(r, f.EncryptedData); err != nil {
		return err
	}

	if err := common.UnmarshalBlsScalar(r, f.HashedSecret); err != nil {
		return err
	}

	if err := common.UnmarshalBlsScalar(r, f.Nonce); err != nil {
		return err
	}

	if err := keys.UnmarshalStealthAddress(r, f.PkR); err != nil {
		return err
	}

	if err := common.UnmarshalJubJubCompressed(r, f.Commitment); err != nil {
		return err
	}

	if err := common.UnmarshalBlsScalar(r, f.EligibilityTS); err != nil {
		return err
	}

	return common.UnmarshalBlsScalar(r, f.ExpirationTS)
}

// BidTransactionRequest is used to construct a Bid transaction.
type BidTransactionRequest struct {
	K                    *common.BlsScalar        `json:"k"`
	Value                uint64                   `json:"value"`
	Secret               *common.JubJubCompressed `json:"secret"`
	PkR                  *keys.StealthAddress     `json:"pk_r"`
	Seed                 *common.BlsScalar        `json:"seed"`
	LatestConsensusRound uint64                   `json:"latest_consensus_round"`
	LatestConsensusStep  uint64                   `json:"latest_consensus_step"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (b *BidTransactionRequest) Copy() *BidTransactionRequest {
	return &BidTransactionRequest{
		K:                    b.K.Copy(),
		Value:                b.Value,
		Secret:               b.Secret.Copy(),
		PkR:                  b.PkR.Copy(),
		Seed:                 b.Seed.Copy(),
		LatestConsensusRound: b.LatestConsensusRound,
		LatestConsensusStep:  b.LatestConsensusStep,
	}
}

// MBidTransactionRequest copies the BidTransactionRequest structure into the Rusk equivalent.
func MBidTransactionRequest(r *rusk.BidTransactionRequest, f *BidTransactionRequest) {
	common.MBlsScalar(r.K, f.K)
	r.Value = f.Value
	common.MJubJubCompressed(r.Secret, f.Secret)
	keys.MStealthAddress(r.PkR, f.PkR)
	common.MBlsScalar(r.Seed, f.Seed)
	r.LatestConsensusRound = f.LatestConsensusRound
	r.LatestConsensusStep = f.LatestConsensusStep
}

// UBidTransactionRequest copies the Rusk BidTransactionRequest structure into the native equivalent.
func UBidTransactionRequest(r *rusk.BidTransactionRequest, f *BidTransactionRequest) {
	common.UBlsScalar(r.K, f.K)
	f.Value = r.Value
	common.UJubJubCompressed(r.Secret, f.Secret)
	keys.UStealthAddress(r.PkR, f.PkR)
	common.UBlsScalar(r.Seed, f.Seed)
	f.LatestConsensusRound = r.LatestConsensusRound
	f.LatestConsensusStep = r.LatestConsensusStep
}

// MarshalBidTransactionRequest writes the BidTransactionRequest struct into a bytes.Buffer.
func MarshalBidTransactionRequest(r *bytes.Buffer, f *BidTransactionRequest) error {
	if err := common.MarshalBlsScalar(r, f.K); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, f.Value); err != nil {
		return err
	}

	if err := common.MarshalJubJubCompressed(r, f.Secret); err != nil {
		return err
	}

	if err := keys.MarshalStealthAddress(r, f.PkR); err != nil {
		return err
	}

	if err := common.MarshalBlsScalar(r, f.Seed); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, f.LatestConsensusRound); err != nil {
		return err
	}

	return encoding.WriteUint64LE(r, f.LatestConsensusStep)
}

// UnmarshalBidTransactionRequest reads a BidTransactionRequest struct from a bytes.Buffer.
func UnmarshalBidTransactionRequest(r *bytes.Buffer, f *BidTransactionRequest) error {
	if err := common.UnmarshalBlsScalar(r, f.K); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &f.Value); err != nil {
		return err
	}

	if err := common.UnmarshalJubJubCompressed(r, f.Secret); err != nil {
		return err
	}

	if err := keys.UnmarshalStealthAddress(r, f.PkR); err != nil {
		return err
	}

	if err := common.UnmarshalBlsScalar(r, f.Seed); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &f.LatestConsensusRound); err != nil {
		return err
	}

	return encoding.ReadUint64LE(r, &f.LatestConsensusStep)
}

// FindBidRequest is used to find a specific Bid belonging to a public key.
type FindBidRequest struct {
	Addr *keys.StealthAddress `json:"addr"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (f *FindBidRequest) Copy() *FindBidRequest {
	return &FindBidRequest{
		Addr: f.Addr.Copy(),
	}
}

// MFindBidRequest copies the FindBidRequest structure into the Rusk equivalent.
func MFindBidRequest(r *rusk.FindBidRequest, f *FindBidRequest) {
	keys.MStealthAddress(r.Addr, f.Addr)
}

// UFindBidRequest copies the Rusk FindBidRequest structure into the native equivalent.
func UFindBidRequest(r *rusk.FindBidRequest, f *FindBidRequest) {
	keys.UStealthAddress(r.Addr, f.Addr)
}

// MarshalFindBidRequest writes the FindBidRequest struct into a bytes.Buffer.
func MarshalFindBidRequest(r *bytes.Buffer, f *FindBidRequest) error {
	return keys.MarshalStealthAddress(r, f.Addr)
}

// UnmarshalFindBidRequest reads a FindBidRequest struct from a bytes.Buffer.
func UnmarshalFindBidRequest(r *bytes.Buffer, f *FindBidRequest) error {
	return keys.UnmarshalStealthAddress(r, f.Addr)
}

// BidList is the collection of all blind bids.
type BidList struct {
	BidList     []*Bid              `json:"bid_list"`
	BidHashList []*common.BlsScalar `json:"bid_hash_list"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (b *BidList) Copy() *BidList {
	bidList := make([]*Bid, len(b.BidList))
	for i := range bidList {
		bidList[i] = b.BidList[i].Copy()
	}

	bidHashList := make([]*common.BlsScalar, len(b.BidHashList))
	for i := range bidHashList {
		bidHashList[i] = b.BidHashList[i].Copy()
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

	r.BidHashList = make([]*rusk.BlsScalar, len(f.BidHashList))
	for i, bidHash := range r.BidHashList {
		common.MBlsScalar(bidHash, f.BidHashList[i])
	}
}

// UBidList copies the Rusk BidList structure into the native equivalent.
func UBidList(r *rusk.BidList, f *BidList) {
	f.BidList = make([]*Bid, len(r.BidList))
	for i, bid := range f.BidList {
		UBid(r.BidList[i], bid)
	}

	f.BidHashList = make([]*common.BlsScalar, len(r.BidHashList))
	for i, bidHash := range f.BidHashList {
		common.UBlsScalar(r.BidHashList[i], bidHash)
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
		if err := common.MarshalBlsScalar(r, bidHash); err != nil {
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

	f.BidHashList = make([]*common.BlsScalar, lenBidHashList)
	for _, bidHash := range f.BidHashList {
		if err = common.UnmarshalBlsScalar(r, bidHash); err != nil {
			return err
		}
	}

	return nil
}

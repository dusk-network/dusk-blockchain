package syncmgr

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermgr"
	"sync"
)

var instance *Syncmgr
var once sync.Once

// GetInstance creates a Syncmgr instance as a Singleton
func GetInstance() (*Syncmgr, error) {
	var err error

	if instance == nil {
		once.Do(func() {
			instance = &Syncmgr{pmgr: peermgr.New(), Mode: 1, headers: [][]byte{}, inflightBlockReqs: make(map[string]*peermgr.Peer, 2000)}
			instance.chain, err = core.GetBcInstance()
			instance.pcfg = instance.setupPeerResponseHandler()
		})
	}
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (s *Syncmgr) setupPeerResponseHandler() peermgr.ResponseHandler {

	peerRspHndlr := peermgr.ResponseHandler{
		OnHeaders: s.OnHeaders,
		//OnNotFound:	      s.OnNotFound,
		OnGetData: s.OnGetData,
		//OnTx:	          s.OnTx,
		OnGetHeaders: s.OnGetHeaders,
		//OnAddr:           s.OnAddr,
		//OnGetAddr:        s.OnGetAddr,
		//OnGetBlocks:      s.OnGetBlocks,
		OnBlock: s.OnBlock,
		//OnBinary:         s.OnBinary,
		//OnCandidate:      s.OnCandidate,
		//OnCertificate:    s.OnCertificate,
		//OnCertificateReq: s.OnCertificateReq,
		OnMemPool: s.OnMemPool,
		//OnPing:           s.OnPing,
		//OnPong:           s.OnPong,
		//OnReduction:      s.OnReduction,
		//OnReject:         s.OnReject,
		//OnScore:          s.OnScore
		//OnInv:            s.OnInv
	}

	return peerRspHndlr
}

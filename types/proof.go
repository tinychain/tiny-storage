package types

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/tinychain/tinychain/common"
	"sync/atomic"
	"time"
)

// Proof holds the provable meta info of storing request.
// In contract, it's stored as kv pair, whose key is hash of proof(proof id),value is proof serialize format.
type Proof struct {
	id        atomic.Value
	Cid       string        `json:"cid"`       // content hash of data
	Size      int           `json:"size"`      // data size
	Duration  time.Duration `json:"duration"`  // storing duration
	Peers     int           `json:"peers"`     // amount of storing peers
	Signature []byte        `json:"signature"` // signature of order caller
}

func (p *Proof) ID() string {
	if id := p.id.Load(); id != nil {
		return id.(string)
	}

	data := make([]byte, 16)
	binary.BigEndian.PutUint64(data[:8], uint64(p.Size))
	binary.BigEndian.PutUint64(data[8:16], uint64(p.Duration))
	hash := common.Sha256(bytes.Join([][]byte{[]byte(p.Cid), data}, nil))
	p.id.Store(hash)
	return hash.String()
}

func (p *Proof) VerifySign(pubKey crypto.PubKey) error {
	match, err := pubKey.Verify([]byte(p.ID()), p.Signature)
	if err != nil {
		return err
	}
	if !match {
		return errSignInvalid
	}
	return nil
}

func (p *Proof) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Proof) Deserialize(d []byte) error {
	return json.Unmarshal(d, p)
}

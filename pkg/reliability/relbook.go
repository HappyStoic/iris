package reliability

import (
	"github.com/libp2p/go-libp2p-core/peer"
)

type Reliability float64

const DefaultReliability = 0

type Book struct {
	peersRel map[peer.ID]Reliability
}

func NewBook() *Book {
	return &Book{peersRel: make(map[peer.ID]Reliability)}
}

func (rb *Book) UpdatePeerRel(p peer.ID, r Reliability) {
	rb.peersRel[p] = r
}

func (rb *Book) PeerRel(p peer.ID) Reliability {
	if val, ok := rb.peersRel[p]; ok {
		return val
	}
	return DefaultReliability
}

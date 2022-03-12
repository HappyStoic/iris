package reliability

import (
	"github.com/libp2p/go-libp2p-core/peer"
)

type Reliability float64

type Callback func(p peer.ID, r Reliability)

const DefaultReliability = 0

type Book struct {
	peersRel map[peer.ID]Reliability

	callbacks []Callback
}

func NewBook() *Book {
	return &Book{
		peersRel:  make(map[peer.ID]Reliability),
		callbacks: make([]Callback, 0),
	}
}

func (rb *Book) UpdatePeerRel(p peer.ID, r Reliability) {
	rb.peersRel[p] = r

	// call registered callbacks with new values
	for _, f := range rb.callbacks {
		f(p, r)
	}
}

func (rb *Book) SubscribeForChange(f Callback) {
	rb.callbacks = append(rb.callbacks, f)
}

func (rb *Book) PeerRel(p peer.ID) Reliability {
	if val, ok := rb.peersRel[p]; ok {
		return val
	}
	return DefaultReliability
}

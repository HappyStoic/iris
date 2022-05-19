package reliability

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"math"
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

// ExpTransformedPeerRel transforms a peer's reliability with function
// y=((a^x) - 1)/(a - 1) * 1000; a=10
// to a weighting factor
// visualisation -> https://www.desmos.com/calculator/glz5g7fvrz
func (rb *Book) ExpTransformedPeerRel(p peer.ID) uint {
	const a = 10.0
	rel := float64(rb.PeerRel(p))
	return uint((math.Pow(a, rel) - 1) / (a - 1) * 1000)
}

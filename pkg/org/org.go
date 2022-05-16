package org

import (
	"encoding/base64"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
)

var log = logging.Logger("iris")

// TODO: use this to write docs?
// # I trust some organisations, thus I need:
// * PubKey -> used as ID of the org and also as key to verify the signatures
// # I am trusted by some orgs, thus I need:
// * PubKey    -> used as ID of the org
// * Signature -> used as a proof that I am trusted by this org

// Org represents public ID of an organisation. It can be used to verify
// organisation signatures
type Org peer.ID

// String returns its string representation
func (o *Org) String() string {
	return peer.ID(*o).String()
}

// Cid returns ipfs cid which can be used in a DHTs as key
func (o *Org) Cid() (cid.Cid, error) {
	// TODO: maybe there is already somehow hidden cid in peer.ID
	return cid.V0Builder.Sum(cid.V0Builder{}, []byte(*o))
}

// VerifyPeer verifies if signature was signed by underlying organization. It
// expects signature encoded in base64
func (o *Org) VerifyPeer(p peer.ID, b64Sig string) (bool, error) {
	// first get raw bytes which should have been signed
	signedKey, err := p.ExtractPublicKey()
	if err != nil {
		return false, err
	}
	signedData, err := signedKey.Raw()
	if err != nil {
		return false, err
	}
	// decode and verify the signature
	sig, err := base64.StdEncoding.DecodeString(b64Sig)
	if err != nil {
		return false, err
	}
	orgPubKey, err := peer.ID(*o).ExtractPublicKey()
	if err != nil {
		return false, err
	}
	return orgPubKey.Verify(signedData, sig)
}

// Decode decodes Org from its string ID representation
func Decode(s string) (*Org, error) {
	o, err := peer.Decode(s)
	if err != nil {
		return nil, errors.Errorf("error creating redis client: %s", err)
	}
	x := Org(o)
	return &x, nil
}

// SignPeer signs given peer. It signs raw bytes of peer public key which means
// it basically signs peer's ID. It returns signature encoded in base64
func SignPeer(key crypto.PrivKey, p peer.ID) (string, error) {
	// first get raw bytes to sign out of peer p
	keyToSign, err := p.ExtractPublicKey()
	if err != nil {
		return "", err
	}
	dataToSign, err := keyToSign.Raw()
	if err != nil {
		return "", err
	}
	// sign the data
	sig, err := key.Sign(dataToSign)
	if err != nil {
		return "", nil
	}
	// encode signature it in base64
	b64Sig := base64.StdEncoding.EncodeToString(sig)
	return b64Sig, nil
}

package protocols

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"

	"happystoic/p2pnetwork/pkg/messaging/pb"
	"happystoic/p2pnetwork/pkg/messaging/utils"
	"happystoic/p2pnetwork/pkg/org"
)

// p2p protocol definition
const p2pOrgSignatureProtocol = "/org-signature/0.0.1"

// OrgSigProtocol type
type OrgSigProtocol struct {
	*utils.ProtoUtils
}

func NewOrgSigProtocol(pu *utils.ProtoUtils) *OrgSigProtocol {
	os := &OrgSigProtocol{pu}

	os.Host.SetStreamHandler(p2pOrgSignatureProtocol, os.onP2POrgSigRequest)

	return os
}

func (os *OrgSigProtocol) AskForOrgSignatures(p peer.ID) {
	if len(os.OrgBook.Trustworthy) == 0 {
		// I don't trust any organisation, no need to ask about signatures
		return
	}

	// TODO: ask just once in a period of time?

	log.Debugf("requesting org signatures from peer '%s'", p)

	s, err := os.OpenStream(p, p2pOrgSignatureProtocol)
	if err != nil {
		log.Errorf("error opening stream: %s", err)
		return
	}

	// deserialize the msg
	orgSigs := &pb.OrgSig{}
	err = os.DeserializeMessageFromStream(s, orgSigs, true)
	if err != nil {
		log.Errorf("error deserilising org sig msg from stream: %s", err)
		return
	}

	// authenticate the msg
	err = os.AuthenticateMessage(orgSigs, orgSigs.Metadata)
	if err != nil {
		log.Errorf("error authenticating org sig message: %s", err)
		return
	}

	// process each signature
	for _, o := range orgSigs.Organisations {
		os.processOrgSig(o, p)
	}

	log.Debugf("ended requesting org signatures from peer '%s'", p)
}

func (os *OrgSigProtocol) onP2POrgSigRequest(s network.Stream) {
	p := s.Conn().RemotePeer()
	log.Debugf("received org signature request from '%s'", p)

	//create msg metadata
	msgMetaData, err := os.NewProtoMetaData()
	if err != nil {
		log.Errorf("error generating new proto metadata: %s", err)
		return
	}

	// create message
	msg := &pb.OrgSig{
		Metadata:      msgMetaData,
		Organisations: os.OrgBook.MySignaturesProto,
	}

	// sign the message
	signature, err := os.SignProtoMessage(msg)
	if err != nil {
		log.Errorf("error generating signature: %s", err)
	}
	msg.Metadata.Signature = signature

	// send the message
	err = os.WriteProtoMsg(msg, s)
	if err != nil {
		log.Errorf("error sending org signatures msg to peer %s: %s", p, err)
	}
	log.Debugf("sucessfully sent my org signatures msg to peer %s", p)
	_ = s.Close()
}

func (os *OrgSigProtocol) processOrgSig(pbO *pb.Organisation, p peer.ID) {
	o, err := org.Decode(pbO.OrgId)
	if err != nil {
		log.Errorf("error decoding org from '%s': %s", pbO.OrgId, err)
		err = os.ReportPeer(p, "provided invalid org ID")
		if err != nil {
			log.Errorf("error reporting peer: %s", err)
		}
		return
	}
	// we don't care about signatures from organisations we don't trust
	if !os.OrgBook.IsTrustworthy(o) {
		log.Debugf("org '%s' is not trusted, skipping processing", o)
		return
	}
	// check the signature
	ok, err := o.VerifyPeer(p, pbO.Signature)
	if err != nil {
		log.Errorf("error verifying signature of org '%s'", o)
	}
	if !ok {
		log.Errorf("signature of org '%s' is invalid!", o)
		err = os.ReportPeer(p, "provided invalid org signature")
		if err != nil {
			log.Errorf("error reporting peer: %s", err)
		}
		return
	}

	// everything is correct, save the information
	os.OrgBook.AddVerifiedSig(p, o)
	log.Infof("successfully verified signature of org '%s'", o)
}

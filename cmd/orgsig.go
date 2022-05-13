package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/libp2p/go-libp2p-core/crypto"
	libp2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"

	"happystoic/p2pnetwork/pkg/config"
	"happystoic/p2pnetwork/pkg/cryptotools"
	"happystoic/p2pnetwork/pkg/org"
)

/*
 *
 * TODO: write docs
 *
 */

const version = "v0.0.1"

func parseFlags() (*string, *string, *bool, *string, error) {
	// flags regarding the organisation key
	loadKeyPath := flag.String("load-key-path", "",
		"Path to a file with organisation private key. If not set, new "+
			"private-key is generated.")
	saveKeyPath := flag.String("save-key-path", "", "If "+
		"set, value will be used as a path to save organisation private-key.")

	// flags regarding the signature of a peer
	signPeer := flag.Bool("sign-peer", false, "Flag to "+
		"sign peer ID. Flag peer-id can be used to set peer"+
		"ID, otherwise, cli will ask. The signature will be printed to stdout.")
	peerId := flag.String("peer-id", "",
		"Public ID of a peer to sign. Flag --sign-peer must be "+
			"set for this option to be valid.")

	flag.Parse() // parse the args

	return loadKeyPath, saveKeyPath, signPeer, peerId, nil
}

func processArgs() (*string, *string, *bool, *string, error) {
	loadKeyPath, saveKeyPath, signPeer, peerId, err := parseFlags()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	if *saveKeyPath == "" && !(*signPeer) {
		err = errors.Errorf("Nothing to do. At least one of" +
			" 'save-key-path' or 'sign-peer' flags must be set. Run '--help'" +
			" for more information")
	}
	return loadKeyPath, saveKeyPath, signPeer, peerId, err
}

func readPeerIDFromCli() (string, error) {
	fmt.Println("Peer ID to sign:")
	fmt.Print("> ")

	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	// get rid of \n at the end of a line
	return input[:len(input)-1], err
}

func produceOutput(key crypto.PrivKey, saveKeyPath string, signature string) error {
	var out strings.Builder
	out.WriteString("\n")

	if signature != "" {
		out.WriteString(fmt.Sprintf("Peer's signature:\n%s\n\n", signature))
	}

	if saveKeyPath != "" {
		bytes, err := libp2pcrypto.MarshalPrivateKey(key)
		if err != nil {
			return err
		}
		err = os.WriteFile(saveKeyPath, bytes, 0644)
		if err != nil {
			return err
		}
		out.WriteString(fmt.Sprintf("Saved organisation private key to path:"+
			"\n\t%s\n", saveKeyPath))
	}
	pubId, err := peer.IDFromPrivateKey(key)
	if err != nil {
		return err
	}
	out.WriteString(fmt.Sprintf("Organisation's ID (public-key):"+
		"\n\t%s\n\n", pubId.String()))

	fmt.Print(out.String())
	return nil
}

func fatal(code int, err error) {
	fmt.Println(err)
	os.Exit(code)
}

func getPeerToSign(peerIdArg string) (peer.ID, error) {
	var err error
	if peerIdArg == "" {
		peerIdArg, err = readPeerIDFromCli()
		if err != nil {
			return "", err
		}
	}
	return peer.Decode(peerIdArg)
}

func main() {
	fmt.Printf("Running %s orgsig\n\n", version)

	// process command line arguments
	loadKeyPath, saveKeyPath, signPeer, peerId, err := processArgs()
	if err != nil {
		fatal(1, err)
	}

	// load or generate new private key of the organization
	var i config.IdentityConfig
	if *loadKeyPath != "" {
		i = config.IdentityConfig{LoadKeyFromFile: *loadKeyPath}
	} else {
		if *saveKeyPath == "" {
			fatal(2, fmt.Errorf("--save-key-path must be set"+
				" when generating new organisation private key"))
		}
		i = config.IdentityConfig{GenerateNewKey: true}
	}
	key, err := cryptotools.GetPrivateKey(&i)
	if err != nil {
		fatal(3, err)
	}

	// sign the peer if possible
	signature := ""
	if *signPeer {
		p, err := getPeerToSign(*peerId)
		if err != nil {
			fatal(4, err)
		}
		signature, err = org.SignPeer(key, p)
		if err != nil {
			fatal(5, err)
		}
	}

	// produce desired output
	err = produceOutput(key, *saveKeyPath, signature)
	if err != nil {
		fatal(6, err)
	}
	fmt.Println("Finished...")
}

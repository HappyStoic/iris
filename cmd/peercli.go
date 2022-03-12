package main

import (
	"context"
	"flag"
	"math/rand"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"happystoic/p2pnetwork/pkg/config"
	"happystoic/p2pnetwork/pkg/node"
)

var log = logging.Logger("p2pnetwork")

// TODO: signal handling
// TODO: search for buddies from the same organisation in peer discovery
// TODO: add rate limitting/queue for each peer
// TODO: make a goroutine to purge message cache keys after some time
// TODO: finish peer discovery
// TODO: create connection manager (using reliability and some other smart algorithms)
// TODO: deal with NAT
// TODO: add intelligence msg (be careful, id of original msg has to be present when forwarding and checking if peer has already seen that)
// TODO: add redis api
// TODO: add network notiffee and send update to TL through redis when peers connecgt/disconnect
// TODO: use msgs timestamp to throw away old messages?
// TODO: can you spoof QUIC sender? If not, tell TL if stream is corrupted and cannot be deserialized so the IP can be punished
// TODO: what peers to ask when asking for intelligence? Or ask all of them? How to rotate them? Maybe use some probability distributiion along with reliability?
// TODO: responseStorage should not wait for responses from peers that disconnect. Otherwise when that happens it's gonna wait always till the timeout occurs?
// TODO: wait in storageResponse only for responses from peers where requests were sucessfully sent (err was nil)
// TODO: maybe I should send all messages in new goroutine so it does not black? (especially p2p newStream functions?)
// TODO: create tool to generate orgs priv/pub key and tool sign peers
// TODO: use more reporting of peers to TL
// TODO: verify other peers' organisations signatures
// TODO: maybe delete file meta after expired elapsed? right now ElapsedAt is not used
// TODO: is basic manager really trimming peers based on their reliability?

func loadConfig() (*config.Config, error) {
	var c config.Config

	configFile := flag.String("conf", "", "path to configuration file")
	flag.Parse()

	if configFile == nil || *configFile == "" {
		return nil, errors.New("missing path of configuration file")
	} else {
	}
	viper.SetConfigFile(*configFile)

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	if err := viper.Unmarshal(&c); err != nil {
		return nil, err
	}

	return &c, c.Check()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create p2p node
	localNode, err := node.NewNode(cfg, ctx)
	if err != nil {
		log.Fatal(err)
	}

	// print connection strings
	log.Infof("created node with ID: %s", localNode.ID())
	for _, addr := range localNode.Addrs() {
		log.Infof("connection string: '%s %s'", addr, localNode.ID())
	}

	localNode.Start(ctx)
	log.Infof("finished, program terminating...")
}

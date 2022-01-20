package main

import (
	"context"
	"flag"
	logging "github.com/ipfs/go-log/v2"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"happystoic/p2pnetwork/pkg/config"
	"happystoic/p2pnetwork/pkg/node"
	"os"
)

var log = logging.Logger("p2pnetwork")

// TODO: signal handling
// TODO: search for buddies from the same organisation in peer discovery
// TODO: add rate limitting/queue for each peer
// TODO: make a goroutine to purge message cache keys after some time
// TODO: finish peer discovery
// TODO: create connection manager (using reliability and some other smart algorithms)
// TODO: deal with NAT

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
	ctx, _ := context.WithCancel(context.Background())
	conf, err := loadConfig()
	if err != nil {
		panic(err)
	}

	localNode, err := node.NewNode(conf, ctx)
	if err != nil {
		panic(err)
	}
	log.Infof("created node with ID: %s", localNode.ID())
	for _, addr := range localNode.Addrs() {
		log.Infof("connection string: '%s %s'", addr, localNode.ID())
	}

	doSomething := len(os.Getenv("DO_SOMETHING")) > 0

	localNode.ConnectToInitPeers()
	localNode.Start(doSomething)

	log.Infof("finished, program terminating...")
}

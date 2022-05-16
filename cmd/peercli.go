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

var log = logging.Logger("iris")

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
	rand.Seed(time.Now().UnixNano()) // Todo use something more cryptographically secure

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

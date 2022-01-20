package config

import (
	"fmt"
	"github.com/pkg/errors"
)

type Config struct {
	IdentityConfig IdentityConfig
	PeerDiscovery  PeerDiscovery
	Server         Server
}

type Server struct {
	Port uint
}

type PeerDiscovery struct {
	UseDns               bool
	UseRedisCache        bool
	ListOfMultiAddresses []string
}

type IdentityConfig struct {
	GenerateNewKey  bool
	LoadKeyFromFile string
	SaveKeyToFile   string
}

func (c *Config) Check() error {
	fmt.Println(c.IdentityConfig)
	if c.IdentityConfig.GenerateNewKey && c.IdentityConfig.LoadKeyFromFile != "" {
		return errors.New("cannot generate new key and load one from file at the same time")
	}
	if !c.IdentityConfig.GenerateNewKey && c.IdentityConfig.LoadKeyFromFile == "" {
		return errors.New("specify either to generate a new key or load one from a file")
	}

	return nil
}

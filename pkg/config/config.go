package config

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

type Config struct {
	Identity         IdentityConfig
	PeerDiscovery    PeerDiscovery
	Server           Server
	Redis            Redis
	ProtocolSettings ProtocolSettings
}

type Server struct {
	Port uint32
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

type Redis struct {
	Host         string
	Port         uint
	Db           int
	Username     string
	Password     string
	Tl2NlChannel string
}

type ProtocolSettings struct {
	Recommendation RecommendationSettings
}

type RecommendationSettings struct {
	Timeout time.Duration
}

// Addr constructs address from host and port
func (r *Redis) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

func (c *Config) Check() error {
	// validity check
	if c.Identity.GenerateNewKey && c.Identity.LoadKeyFromFile != "" {
		return errors.New("cannot generate new key and load one from file at the same time")
	}
	if !c.Identity.GenerateNewKey && c.Identity.LoadKeyFromFile == "" {
		return errors.New("specify either to generate a new key or load one from a file")
	}
	if c.Redis.Host == "" {
		return errors.New("redis host must be specified")
	}
	if c.Redis.Tl2NlChannel == "" {
		return errors.New("tl2nl redis channel must be specified")
	}
	if c.ProtocolSettings.Recommendation.Timeout == 0 {
		return errors.New("ProtocolSettings.Recommendation.Timeout must be set")
	}
	// default values
	if c.Redis.Port == 0 {
		c.Redis.Port = 6379 // Use default redis port if port is not specified
	}
	return nil
}

package config

import (
	"fmt"
	"strings"
	"time"

	logging "github.com/ipfs/go-log/v2"

	"github.com/pkg/errors"
)

var log = logging.Logger("p2pnetwork")

type Config struct {
	Identity         IdentityConfig
	PeerDiscovery    PeerDiscovery
	Server           Server
	Redis            Redis
	ProtocolSettings ProtocolSettings
	Organisations    OrgConfig
}

type Server struct {
	Port uint32
}

type OrgConfig struct {
	Trustworthy  []string
	MySignatures []OrgSig

	// DhtUpdatePeriod says often to ask DHT about members of trusted orgs
	DhtUpdatePeriod time.Duration
}

type OrgSig struct {
	ID        string
	Signature string
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
	Intelligence   IntelligenceSettings
	FileShare      FileShareSettings
}

type FileShareSettings struct {
	MetaSpreadSettings map[string]SpreadStrategy
	DownloadDir        string // default value /tmp TODO test this value working
}

type SpreadStrategy struct {
	NumberOfPeers int
	Every         time.Duration
	Until         time.Duration
}

type RecommendationSettings struct {
	Timeout time.Duration
}

type IntelligenceSettings struct {
	MaxTtl           uint32        // max ttl the peer will forward to prevent going too deep
	Ttl              uint32        // ttl to set when initiating intelligence request
	MaxParentTimeout time.Duration // max allowed timeout parent is waiting (so he does not make me stuck waiting forever)
	RootTimeout      time.Duration // timeout for responses after initiating intelligence request
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

	for sev, settings := range c.ProtocolSettings.FileShare.MetaSpreadSettings {
		lsev := strings.ToUpper(sev)
		if lsev != "MINOR" && lsev != "MAJOR" && lsev != "CRITICAL" {
			return errors.Errorf("unknown severity ProtocolSettings.FileShare.MetaSpreadSettings=%s", lsev)
		}
		if settings.NumberOfPeers <= 0 {
			log.Warnf("Config: ProtocolSettings.FileShare.MetaSpread"+
				"Settings.%s.NumberOfPeers=%d - spreading disabled",
				lsev, settings.NumberOfPeers)
			continue
		}
		if settings.Until <= 0 {
			log.Warnf("Config: ProtocolSettings.FileShare.MetaSpread"+
				"Settings.%s.Until=%s - undefined behavior", lsev, settings.Until)
		}
		if settings.Every <= 0 {
			log.Warnf("Config: ProtocolSettings.FileShare.MetaSpread"+
				"Settings.%s.Every=%s - periodical spreading disabled",
				lsev, settings.Every)
		}
	}

	// default values
	if c.Redis.Port == 0 {
		c.Redis.Port = 6379 // Use default redis port if port is not specified
	}
	if c.ProtocolSettings.FileShare.DownloadDir == "" {
		c.ProtocolSettings.FileShare.DownloadDir = "/tmp"
	}
	return nil
}

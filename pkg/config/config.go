package config

import (
	"fmt"
	"strings"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/pkg/errors"

	"happystoic/p2pnetwork/pkg/utils"
)

var log = logging.Logger("iris")

type Config struct {
	Identity         IdentityConfig
	PeerDiscovery    PeerDiscovery
	Server           Server
	Redis            Redis
	ProtocolSettings ProtocolSettings
	Organisations    OrgConfig
	Connections      Connections
}

type Server struct {
	// UDP Port to listen on. Missing or zero value indicates random port
	// between <9000,11000>
	Port          uint32
	Host          string
	DhtServerMode bool // default (false) means auto mode
}

func (s *Server) validate() error {
	if s.Port != 0 {
		return utils.CheckUDPPortAvailability(s.Port)
	}
	return nil
}

func (s *Server) setDefaults() error {
	if s.Host == "" {
		s.Host = "127.0.0.1"
	}
	if s.Port != 0 {
		return nil
	}
	var p uint32
	for p = 9000; p < 11000; p++ {
		err := utils.CheckUDPPortAvailability(p)
		if err == nil {
			// found free port
			s.Port = p
			return nil
		}
	}
	return errors.Errorf("no available port found")
}

type Connections struct {
	// lower bound for number of connections
	Low int
	// number of connections this peer will proactively seek. Other slots will
	// be saved for incoming connections
	Medium int
	// upper bound for number of connections
	High int

	// ReconnectInterval sets how often we try to connect to new peers if
	// number of peers is below Medium. Negative value disables updater.
	// Defaults to 10 minutes
	ReconnectInterval time.Duration
}

func (c *Connections) setDefaults() {
	if c.Low == 0 {
		c.Low = 15
	}
	if c.Medium == 0 {
		c.Medium = 30
	}
	if c.High == 0 {
		c.High = 50
	}
	if c.ReconnectInterval == 0 {
		c.ReconnectInterval = 10 * time.Minute
	}
}

type OrgConfig struct {
	Trustworthy  []string
	MySignatures []OrgSig

	// DhtUpdatePeriod says often to ask DHT about members of trusted orgs
	DhtUpdatePeriod time.Duration
}

func (o *OrgConfig) setDefaults() {
	if o.DhtUpdatePeriod == 0 {
		o.DhtUpdatePeriod = time.Minute * 5
	}
}

type OrgSig struct {
	ID        string
	Signature string
}

type PeerDiscovery struct {
	UseDns                    bool
	UseRedisCache             bool
	DisableBootstrappingNodes bool
	ListOfMultiAddresses      []string
}

type IdentityConfig struct {
	GenerateNewKey  bool
	LoadKeyFromFile string
	SaveKeyToFile   string
}

func (ic *IdentityConfig) validate() error {
	if ic.GenerateNewKey && ic.LoadKeyFromFile != "" {
		return errors.New("cannot generate new key and load one from file at the same time")
	}
	if !ic.GenerateNewKey && ic.LoadKeyFromFile == "" {
		return errors.New("specify either to generate a new key or load one from a file")
	}
	return nil
}

type Redis struct {
	Host         string
	Port         uint
	Db           int
	Username     string
	Password     string
	Tl2NlChannel string
}

func (r *Redis) validate() error {
	if r.Host == "" {
		return errors.New("redis host must be specified")
	}
	if r.Tl2NlChannel == "" {
		return errors.New("tl2nl redis channel must be specified")
	}
	return nil
}

func (r *Redis) setDefaults() {
	if r.Port == 0 {
		r.Port = 6379
	}
}

type ProtocolSettings struct {
	Recommendation RecommendationSettings
	Intelligence   IntelligenceSettings
	FileShare      FileShareSettings
}

func (ps *ProtocolSettings) validate() error {
	for sev, settings := range ps.FileShare.MetaSpreadSettings {
		uSev := strings.ToUpper(sev)
		if uSev != "MINOR" && uSev != "MAJOR" && uSev != "CRITICAL" {
			return errors.Errorf("unknown severity ProtocolSettings.FileShare.MetaSpreadSettings=%s", uSev)
		}
		if settings.NumberOfPeers <= 0 {
			log.Warnf("Config: ProtocolSettings.FileShare.MetaSpread"+
				"Settings.%s.NumberOfPeers=%d - spreading disabled",
				uSev, settings.NumberOfPeers)
			continue
		}
		if settings.Until <= 0 {
			log.Warnf("Config: ProtocolSettings.FileShare.MetaSpread"+
				"Settings.%s.Until=%s - undefined behavior", uSev, settings.Until)
		}
		if settings.Every <= 0 {
			log.Warnf("Config: ProtocolSettings.FileShare.MetaSpread"+
				"Settings.%s.Every=%s - periodical spreading disabled",
				uSev, settings.Every)
		}
	}
	return nil
}

func (ps *ProtocolSettings) setDefaults() {
	if ps.FileShare.DownloadDir == "" {
		ps.FileShare.DownloadDir = "/tmp"
	}
	if ps.Recommendation.Timeout == 0 {
		ps.Recommendation.Timeout = 10 * time.Second
	}
	if ps.Intelligence.MaxTtl == 0 {
		ps.Intelligence.MaxTtl = 5
	}
	if ps.Intelligence.Ttl == 0 {
		ps.Intelligence.Ttl = 5
	}
	if ps.Intelligence.RootTimeout == 0 {
		ps.Intelligence.RootTimeout = 10 * time.Second
	}
	if ps.Intelligence.MaxParentTimeout == 0 {
		ps.Intelligence.MaxParentTimeout = 10 * time.Second
	}
}

type FileShareSettings struct {
	MetaSpreadSettings map[string]SpreadStrategy
	DownloadDir        string
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
	if err := c.Identity.validate(); err != nil {
		return err
	}
	if err := c.Redis.validate(); err != nil {
		return err
	}
	if err := c.ProtocolSettings.validate(); err != nil {
		return err
	}
	if err := c.Server.validate(); err != nil {
		return err
	}

	// default values
	c.Redis.setDefaults()
	c.ProtocolSettings.setDefaults()
	c.Connections.setDefaults()
	c.Organisations.setDefaults()
	if err := c.Server.setDefaults(); err != nil {
		return err
	}
	return nil
}

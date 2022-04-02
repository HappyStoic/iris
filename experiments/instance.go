package main

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"

	"happystoic/p2pnetwork/pkg/config"
	"happystoic/p2pnetwork/pkg/cryptotools"
)

var portCount = 9000

func nextPort() int {
	portCount++
	return portCount
}

const binaryPath = "/home/yourmother/moje/projects/p2pnetwork/peercli"

var envs = []string{
	"GOLOG_LOG_LEVEL=connmgr=debug,p2pnetwork=debug",
	"GOLOG_LOG_FMT=json",
}

type instance struct {
	dir string

	port   int
	peerId string

	initConns []string

	// organisations
	trustedOrgs           []string
	instanceOrgSignatures []config.OrgSig
}

func NewInstance(dir string) (*instance, error) {
	i := &instance{
		port:      nextPort(),
		dir:       dir,
		initConns: make([]string, 0),

		trustedOrgs:           make([]string, 0),
		instanceOrgSignatures: make([]config.OrgSig, 0),
	}
	err := i.createDirs()
	if err != nil {
		return nil, err
	}

	key, err := i.createKey()
	if err != nil {
		return nil, err
	}
	peerId, err := peer.IDFromPrivateKey(key)
	if err != nil {
		return nil, err
	}
	i.peerId = peerId.String()

	return i, nil
}

func (i *instance) createDirs() error {
	err := os.Mkdir(i.dir, os.ModePerm)
	if err != nil {
		return err
	}
	err = os.Mkdir(i.downloadsDir(), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (i *instance) downloadsDir() string {
	return fmt.Sprintf("%s/downloads", i.dir)
}

func (i *instance) keyPath() string {
	return fmt.Sprintf("%s/key.priv", i.dir)
}

func (i *instance) createKey() (crypto.PrivKey, error) {
	return cryptotools.GetPrivateKey(&config.IdentityConfig{
		GenerateNewKey:  true,
		LoadKeyFromFile: "",
		SaveKeyToFile:   i.keyPath(),
	})
}

func (i *instance) addInitConn(ins *instance) {
	conn := fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic %s", ins.port, ins.peerId)
	i.initConns = append(i.initConns, conn)
}

func (i *instance) addInitConns(instances []*instance) {
	for _, ins := range instances {
		i.addInitConn(ins)
	}
}

func (i *instance) createCfg() config.Config {
	protoCfg := config.ProtocolSettings{
		FileShare: config.FileShareSettings{
			MetaSpreadSettings: map[string]config.SpreadStrategy{
				"MINOR": {
					NumberOfPeers: 2,
					Every:         time.Minute * 20,
					Until:         time.Hour,
				},
			},
			DownloadDir: i.downloadsDir(),
		},
	}

	c := config.Config{
		Identity: config.IdentityConfig{
			GenerateNewKey:  false,
			LoadKeyFromFile: i.keyPath(),
		},
		PeerDiscovery: config.PeerDiscovery{
			ListOfMultiAddresses: i.initConns,
		},
		Server: config.Server{
			Port: uint32(i.port),
		},
		Redis: config.Redis{
			Host:         "127.0.0.1",
			Tl2NlChannel: fmt.Sprintf("gp2p_tl2nl%d", i.port),
		},
		ProtocolSettings: protoCfg,
		Organisations: config.OrgConfig{
			Trustworthy:  i.trustedOrgs,
			MySignatures: i.instanceOrgSignatures,
		},
		Connections: config.Connections{
			Low:               2,
			Medium:            3,
			High:              4,
			ReconnectInterval: time.Second * 60,
		}, // use defaults
	}
	return c
}

func (i *instance) cfgPath() string {
	return fmt.Sprintf("%s/conf.yaml", i.dir)
}

func (i *instance) writeCfg(cfg config.Config) error {
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(i.cfgPath(), data, os.ModePerm)
}

func (i *instance) removeCfg() error {
	return os.Remove(i.cfgPath())
}

func (i *instance) addTrustedOrg(trustworthy string) {
	i.trustedOrgs = append(i.trustedOrgs, trustworthy)
}

func (i *instance) addOrgSignature(sig config.OrgSig) {
	i.instanceOrgSignatures = append(i.instanceOrgSignatures, sig)
}

func (i *instance) command(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(ctx, binaryPath, "--conf", i.cfgPath())
	cmd.Env = envs
	return cmd
}

func (i *instance) run(ctx context.Context) error {
	// create conf
	cfg := i.createCfg()
	err := i.writeCfg(cfg)
	if err != nil {
		return err
	}

	cmd := i.command(ctx)

	out, err := os.Create(fmt.Sprintf("%s/out", i.dir))
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	cmd.Stdout = out
	cmd.Stderr = out

	return cmd.Run()
}

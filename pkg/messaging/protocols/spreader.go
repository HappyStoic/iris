package protocols

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	"happystoic/p2pnetwork/pkg/config"
	"happystoic/p2pnetwork/pkg/files"
	"happystoic/p2pnetwork/pkg/messaging/utils"
	"happystoic/p2pnetwork/pkg/org"
)

type SpreadStrategy struct {
	numberOfPeers int
	every         time.Duration
	until         time.Duration
}

type SpreadStrategies map[files.Severity]*SpreadStrategy

var defaultStrategies = map[files.Severity]*SpreadStrategy{
	files.MINOR: {
		numberOfPeers: 2,
		every:         time.Minute * 20,
		until:         time.Hour,
	},
	files.MAJOR: {
		numberOfPeers: 5,
		every:         time.Minute * 10,
		until:         time.Hour,
	},
	files.CRITICAL: {
		numberOfPeers: 10,
		every:         time.Minute * 5,
		until:         time.Hour,
	},
}

type Spreader struct {
	*utils.ProtoUtils

	ctx            context.Context
	pushStrategies map[files.Severity]*SpreadStrategy
}

func NewSpreader(ctx context.Context, pu *utils.ProtoUtils, cfg map[string]config.SpreadStrategy) *Spreader {
	strategies := defaultStrategies
	for rawSev, strategy := range cfg {
		// severity format should be already validated in config package
		sev, _ := files.SeverityFromString(rawSev)

		strategies[sev] = &SpreadStrategy{
			numberOfPeers: strategy.NumberOfPeers,
			every:         strategy.Every,
			until:         strategy.Until,
		}
	}
	return &Spreader{pu, ctx, strategies}
}

func (s *Spreader) spread(protocol protocol.ID,
	nPeers int,
	rights []*org.Org,
	visited map[peer.ID]struct{},
	msg proto.Message) {

	peers := s.GetNConnectedPeers(nPeers, rights, visited)
	log.Debugf("spreading file meta to %d peers", len(peers))
	for _, p := range peers {
		err := s.SendProtoMessage(p, protocol, msg)
		if err != nil {
			log.Errorf("error spreading file meta to peer %s: %s", p.String(), err)
		} else {
			log.Debugf("successfully spread file meta to peer %s", p.String())
		}
		visited[p] = struct{}{}
	}
	log.Debugf("spreading finished")
}

// TODO: do not use "file meta" in in logging if alert proto gonna use this too
func (s *Spreader) startSpreading(protocol protocol.ID,
	sev files.Severity,
	rights []*org.Org,
	msg proto.Message,
	author peer.ID) {

	// To keep track who already knows about the file
	visited := make(map[peer.ID]struct{})
	visited[author] = struct{}{}

	go func() {
		strategy := s.pushStrategies[sev]

		nPeers := strategy.numberOfPeers
		if nPeers <= 0 {
			// no peers configured, don't spread
			return
		}
		s.spread(protocol, nPeers, rights, visited, msg)

		if strategy.every <= 0 {
			// periodical spreading disabled, return now
			log.Debugf("spreading of file done")
			return
		}

		ticker := time.NewTicker(strategy.every)
		timeout := time.After(strategy.until)
		for {
			select {
			case <-ticker.C:
				s.spread(protocol, nPeers, rights, visited, msg)
			case <-timeout:
				ticker.Stop()
				log.Debugf("spreading of file done")
				return
			case <-s.ctx.Done():
				log.Debugf("ending file spread: context cancelled.")
				return
			}
		}
	}()
}

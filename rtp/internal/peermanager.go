package internal

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type SSRC uint32

type CName string

type Address string

type Peer struct {
	sender net.Addr
	ssrcs  map[SSRC]time.Time
}

type Source struct {
	peerCount uint32
	bitrate   int
	cname     CName
	pipeline *Pipeline
}

func (s *Source) isConfigured() bool {
	return s.pipeline != nil
}

type PeerManager struct {
	sync.Mutex

	peers map[Address]*Peer

	sources map[SSRC]*Source
}

func NewPeerManager(ctx context.Context, timeout time.Duration, statsInterval time.Duration, debug string) *PeerManager {
	m := &PeerManager{
		peers:     make(map[Address]*Peer),
		sources:   make(map[SSRC]*Source),
	}
	// start a cleanup routine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(timeout):
				m.Lock()
				// remove all ssrcs that have not been updated in the last timeout.
				for sender, peer := range m.peers {
					for ssrc, timestamp := range peer.ssrcs {
						if time.Since(timestamp) > timeout {
							log.Printf("removing ssrc %d from peer %s", ssrc, sender)
							delete(peer.ssrcs, ssrc)
							if source, ok := m.sources[SSRC(ssrc)]; ok {
								source.peerCount--
								if source.peerCount == 0 {
									source.pipeline.Close()
									delete(m.sources, SSRC(ssrc))
								}
							} else {
								log.Printf("source not found, but peer was referencing it?")
							}
						}
					}
					// if the peer is empty, prune it off.
					if len(peer.ssrcs) == 0 {
						log.Printf("removing peer %s", sender)
						delete(m.peers, sender)
					}
				}
				m.Unlock()
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				var builder strings.Builder
				builder.WriteString(fmt.Sprintf("---- %s peers ----\n", debug))
				for ssrc, source := range m.sources {
					builder.WriteString(fmt.Sprintf("%s -> %d\t%d peers, %d bps\n", source.cname, ssrc, source.peerCount, source.bitrate*8/int(statsInterval/time.Second)))
					source.bitrate = 0
				}
				log.Print(builder.String())
			}
		}
	}()
	return m
}

// MarkReceived updates a peer's ssrcs map with the given ssrc.
func (m *PeerManager) MarkReceived(sender net.Addr, ssrc SSRC, bits int) {
	m.Lock()
	defer m.Unlock()

	// create the source if it doesn't exist yet, or update the bitrate.
	if source, ok := m.sources[ssrc]; ok {
		source.bitrate += bits
	} else {
		m.sources[ssrc] = &Source{}
	}
	addr := Address(sender.String())
	// create the peer if it doesn't exist yet.
	if _, ok := m.peers[addr]; !ok {
		log.Printf("adding ssrc %d from peer %s", ssrc, addr)
		m.peers[Address(sender.String())] = &Peer{
			sender: sender,
			ssrcs:  map[SSRC]time.Time{},
		}
	}
	// mark the peer as a source for the given ssrc.
	if _, ok := m.peers[addr].ssrcs[ssrc]; !ok {
		source := m.sources[ssrc]
		source.peerCount++
	}
	m.peers[addr].ssrcs[ssrc] = time.Now()
}

// GetPeersForSSRC returns a list of peers that have sent a packet with the given ssrc.
func (m *PeerManager) GetPeersForSSRC(ssrc SSRC) []net.Addr {
	m.Lock()
	defer m.Unlock()

	var peers []net.Addr

	for _, peer := range m.peers {
		if _, ok := peer.ssrcs[ssrc]; ok {
			peers = append(peers, peer.sender)
		}
	}

	return peers
}

// GetCNAME gets the assigned cname for the given ssrc, if set.
func (m *PeerManager) GetCNAME(ssrc SSRC) (CName, error) {
	m.Lock()
	defer m.Unlock()

	if source, ok := m.sources[ssrc]; ok {
		return source.cname, nil
	}
	return "", fmt.Errorf("no cname for ssrc %d", ssrc)
}

// GetPipeline gets the pipeline for a given cname.
func (m *PeerManager) GetPipeline(ssrc SSRC) (*Pipeline, error) {
	m.Lock()
	defer m.Unlock()

	if source, ok := m.sources[ssrc]; ok {
		return source.pipeline, nil
	}
	return nil, fmt.Errorf("no source for ssrc %d", ssrc)
}

// IsConfigured checks if the given ssrc is configured.
func (m *PeerManager) IsConfigured(ssrc SSRC) bool {
	m.Lock()
	defer m.Unlock()

	source, ok := m.sources[ssrc]
	return ok && source.isConfigured()
}

// Configure sets the cname and pipeline for a given ssrc.
func (m *PeerManager) Configure(ssrc SSRC, cname CName, pipeline *Pipeline) {
	m.Lock()
	defer m.Unlock()

	if source, ok := m.sources[ssrc]; ok {
		source.cname = cname
		source.pipeline = pipeline
	} else {
		m.sources[ssrc] = &Source{
			cname:   cname,
			pipeline: pipeline,
		}
	}
}
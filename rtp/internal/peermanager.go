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

type Peer struct {
	sender net.Addr
	ssrcs  map[uint32]time.Time
}

type Source struct {
	peerCount uint32
	bitrate   int
	cname     string
}

type PeerManager struct {
	sync.Mutex

	peers map[string]*Peer

	sources map[uint32]*Source
}

func NewPeerManager(ctx context.Context, timeout time.Duration, statsInterval time.Duration) *PeerManager {
	m := &PeerManager{
		peers:   make(map[string]*Peer),
		sources: make(map[uint32]*Source),
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
							if source, ok := m.sources[ssrc]; ok {
								source.peerCount--
								if source.peerCount == 0 {
									delete(m.sources, ssrc)
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
				builder.WriteString("---- peers ----\n")
				for ssrc, source := range m.sources {
					builder.WriteString(fmt.Sprintf("%s -> %d\t%d peers, %d bps\n", source.cname, ssrc, source.peerCount, source.bitrate*8/int(statsInterval / time.Second)))
					source.bitrate = 0
				}
				log.Print(builder.String())
			}
		}
	}()
	return m
}

// MarkReceived updates a peer's ssrcs map with the given ssrc.
func (m *PeerManager) MarkReceived(sender net.Addr, ssrc uint32, bits int) {
	m.Lock()
	defer m.Unlock()

	// create the source if it doesn't exist yet, or update the bitrate.
	if source, ok := m.sources[ssrc]; ok {
		source.bitrate += bits
	} else {
		m.sources[ssrc] = &Source{}
	}
	// create the peer if it doesn't exist yet.
	if _, ok := m.peers[sender.String()]; !ok {
		log.Printf("adding ssrc %d from peer %s", ssrc, sender.String())
		m.peers[sender.String()] = &Peer{
			sender: sender,
			ssrcs:  map[uint32]time.Time{},
		}
	}
	// mark the peer as a source for the given ssrc.
	if _, ok := m.peers[sender.String()].ssrcs[ssrc]; !ok {
		source := m.sources[ssrc]
		source.peerCount++
	}
	m.peers[sender.String()].ssrcs[ssrc] = time.Now()
}

// GetPeersForSSRC returns a list of peers that have sent a packet with the given ssrc.
func (m *PeerManager) GetPeersForSSRC(ssrc uint32) []net.Addr {
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
func (m *PeerManager) GetCNAME(ssrc uint32) (string, error) {
	m.Lock()
	defer m.Unlock()

	if source, ok := m.sources[ssrc]; ok {
		return source.cname, nil
	}
	return "", fmt.Errorf("no cname for ssrc %d", ssrc)
}

// SetCNAME sets the cname for the given ssrc or creates the ssrc if it doens't exist.
func (m *PeerManager) SetCNAME(ssrc uint32, cname string) {
	m.Lock()
	defer m.Unlock()

	if source, ok := m.sources[ssrc]; ok {
		if source.cname != cname {
			log.Printf("assigning cname %s to ssrc %d", cname, ssrc)
			source.cname = cname
		}
	} else {
		m.sources[ssrc] = &Source{
			cname: cname,
		}
	}
}

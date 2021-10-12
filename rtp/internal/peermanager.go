package internal

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
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
	cname     *CName
}

type PeerManager struct {
	sync.Mutex

	peers map[Address]*Peer

	sources map[SSRC]*Source
}

func NewPeerManager(ctx context.Context, timeout time.Duration) *PeerManager {
	m := &PeerManager{
		peers:   make(map[Address]*Peer),
		sources: make(map[SSRC]*Source),
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
	return m
}

// MarkRTCPReceived processes an incoming rtcp packet.
func (m *PeerManager) MarkRTCPReceived(cp rtcp.CompoundPacket) {
	m.Lock()
	defer m.Unlock()

	for _, pkt := range cp {
		for _, ssrc := range pkt.DestinationSSRC() {
			// check if this packet has a cname associated with it.
			if sdes, ok := pkt.(*rtcp.SourceDescription); ok {
				for _, chunk := range sdes.Chunks {
					for _, item := range chunk.Items {
						if item.Type == rtcp.SDESCNAME {
							cname := CName(item.Text)
							if source, ok := m.sources[SSRC(ssrc)]; ok {
								source.cname = &cname
							} else {
								m.sources[SSRC(ssrc)] = &Source{
									cname: &cname,
								}
							}
						}
					}
				}
			}
		}
	}
}

// MarkRTPReceived updates a peer's ssrcs map with the given ssrc.
func (m *PeerManager) MarkRTPReceived(sender net.Addr, p *rtp.Packet) {
	m.Lock()
	defer m.Unlock()

	ssrc := SSRC(p.SSRC)
	bits := p.MarshalSize()

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
		// return an error if the cname is empty.
		if source.cname == nil {
			return "", fmt.Errorf("cname not set for ssrc %d", ssrc)
		}
		return *source.cname, nil
	}
	return "", fmt.Errorf("no cname for ssrc %d", ssrc)
}

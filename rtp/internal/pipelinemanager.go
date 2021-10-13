package internal

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3/pkg/media"
)

type PipelineManager struct {
	sync.Mutex

	peerManager   *PeerManager
	clientManager *ClientManager

	vp8Sink  func(SSRC, media.Sample) (int, error)
	opusSink func(SSRC, media.Sample) (int, error)
	rtcpSink func([]byte) (int, error)

	pipelines map[SSRC]*Pipeline
}

func NewPipelineManager(ctx context.Context, peerManager *PeerManager, clientManager *ClientManager, pc net.PacketConn) *PipelineManager {
	vp8Sink := func(ssrc SSRC, sample media.Sample) (int, error) {
		cname, err := peerManager.GetCNAME(ssrc)
		if err != nil {
			return 0, err
		}
		client, err := clientManager.GetClient(cname)
		if err != nil {
			return 0, err
		}
		if err := client.VideoTrack.WriteSample(sample); err != nil {
			return 0, err
		}
		return len(sample.Data), nil
	}
	opusSink := func(ssrc SSRC, sample media.Sample) (int, error) {
		cname, err := peerManager.GetCNAME(ssrc)
		if err != nil {
			return 0, err
		}
		client, err := clientManager.GetClient(cname)
		if err != nil {
			return 0, err
		}
		if err := client.AudioTrack.WriteSample(sample); err != nil {
			return 0, err
		}
		return len(sample.Data), nil
	}
	rtcpSink := func(buffer []byte) (int, error) {
		// choose peers to send the rtcp packet to.
		pkts, err := rtcp.Unmarshal(buffer)
		if err != nil {
			return 0, err
		}
		written := make(map[string]bool)
		for _, pkt := range pkts {
			for _, ssrc := range pkt.DestinationSSRC() {
				for _, peer := range peerManager.GetPeersForSSRC(SSRC(ssrc)) {
					// prevent duplicates.
					if _, ok := written[peer.String()]; ok {
						continue
					}
					written[peer.String()] = true
					if _, err := pc.WriteTo(buffer, peer); err != nil {
						log.Printf("failed to write rtcp packet: %v", err)
					}
				}
			}
		}
		return len(buffer), nil
	}

	return &PipelineManager{
		peerManager:   peerManager,
		clientManager: clientManager,

		vp8Sink:  vp8Sink,
		opusSink: opusSink,
		rtcpSink: rtcpSink,

		pipelines: make(map[SSRC]*Pipeline),
	}
}

func (p *PipelineManager) Write(sender net.Addr, buf []byte) (int, error) {
	p.Lock()
	defer p.Unlock()

	// parse the rtp packet.
	pkt := &rtp.Packet{}
	if err := pkt.Unmarshal(buf); err != nil {
		return 0, err
	}

	// https://datatracker.ietf.org/doc/html/rfc5761
	if isRTCP := pkt.PayloadType >= 64 && pkt.PayloadType <= 95; isRTCP {
		// rtcp packet.
		cp, err := rtcp.Unmarshal(buf)
		if err != nil {
			return 0, err
		}
		p.peerManager.MarkRTCPReceived(cp)
		for _, ssrc := range rtcp.CompoundPacket(cp).DestinationSSRC() {
			// write to the corresponding pipeline if it exists, otherwise create it.
			if p.pipelines[SSRC(ssrc)] == nil {
				p.pipelines[SSRC(ssrc)] = NewPipeline(
					p.rtcpSink,
					func(sample media.Sample) (int, error) {
						return p.vp8Sink(SSRC(ssrc), sample)
					},
					func(sample media.Sample) (int, error) {
						return p.opusSink(SSRC(ssrc), sample)
					})
			}
			if err := p.pipelines[SSRC(ssrc)].WriteRTP(buf); err != nil {
				return 0, err
			}
		}
	} else {
		// send this packet to the pipeline.
		p.peerManager.MarkRTPReceived(sender, pkt)

		// write to the corresponding pipeline if it exists.
		if pipeline, ok := p.pipelines[SSRC(pkt.SSRC)]; ok {
			if err := pipeline.WriteRTP(buf); err != nil {
				return 0, err
			}
		}
	}
	return len(buf), nil
}

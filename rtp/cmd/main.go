package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/muxable/mediatun/rtp/internal"
	ion "github.com/pion/ion-sdk-go"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: receiver <addr> <destination>")
		os.Exit(1)
	}

	pc, err := net.ListenPacket("udp", os.Args[1])
	if err != nil {
		panic(err)
	}

	log.Printf("listening rtp on %s", os.Args[1])

	engine := ion.NewEngine(ion.Config{
		WebRTC: ion.WebRTCTransportConfig{
			Configuration: webrtc.Configuration{
				ICEServers: []webrtc.ICEServer{
					{URLs: []string{"stun:stun.stunprotocol.org:3478", "stun:stun.l.google.com:19302"}},
				},
			},
		},
	})

	clientManager := internal.NewClientManager(context.Background(), 86400*time.Second, engine, internal.Address(os.Args[2]))
	peerManager := internal.NewPeerManager(context.Background(), 5*time.Second)

	internal.StatsManager{
		ClientManager: clientManager,
		PeerManager:   peerManager,
		Interval:      5 * time.Second,
	}.Run(context.Background())

	pipeline := &internal.Pipeline{
		VP8SampleSink: func(ssrc internal.SSRC, sample media.Sample) (int, error) {
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
		},
		OpusSampleSink: func(ssrc internal.SSRC, sample media.Sample) (int, error) {
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
		},
		RTCPSink: func(buffer []byte) (int, error) {
			// choose peers to send the rtcp packet to.
			pkts, err := rtcp.Unmarshal(buffer)
			if err != nil {
				return 0, err
			}
			written := make(map[string]bool)
			for _, pkt := range pkts {
				for _, ssrc := range pkt.DestinationSSRC() {
					for _, peer := range peerManager.GetPeersForSSRC(internal.SSRC(ssrc)) {
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
		},
	}

	pipeline.Start(context.Background())

	for {
		// read in a udp message.
		buf := make([]byte, 1500)
		n, sender, err := pc.ReadFrom(buf)
		if err != nil {
			panic(err)
		}

		// parse the rtp packet.
		p := &rtp.Packet{}
		if err := p.Unmarshal(buf[0:n]); err != nil {
			log.Printf("received invalid rtp packet: %v", err)
			continue
		}

		// https://datatracker.ietf.org/doc/html/rfc5761
		if isRTCP := p.PayloadType >= 64 && p.PayloadType <= 95; isRTCP {
			// rtcp packet.
			cp, err := rtcp.Unmarshal(buf[0:n])
			if err != nil {
				log.Printf("received invalid rtcp packet: %v", err)
				continue
			}
			peerManager.MarkRTCPReceived(cp)
		} else {
			// send this packet to the pipeline.
			peerManager.MarkRTPReceived(sender, p)
		}

		if  err := pipeline.WriteRTP(buf[0:n]); err != nil {
			log.Printf("failed to write rtp packet: %v", err)
		}

		// if err := pipeline.WriteRTP(p); err != nil {
		// 	log.Printf("failed to write rtp packet to pipeline: %v", err)
		// }
	}
}

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
)

type ListenType uint32

const (
	ListenTypeVideo = ListenType(0)
	ListenTypeAudio = ListenType(1)
)

type Server struct {
	peerManager *internal.PeerManager
	OnRTP       func(string, []byte)
}

func (s *Server) Listen(addr string, pipelineType internal.PipelineType) {
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		panic(err)
	}

	log.Printf("listening rtp on %s", addr)

	for {
		// read in a udp message.
		buf := make([]byte, 1500)
		n, sender, err := pc.ReadFrom(buf)
		if err != nil {
			panic(err)
		}

		// parse the rtp packet.
		p := rtp.Packet{}
		if err := p.Unmarshal(buf[0:n]); err != nil {
			log.Printf("received invalid rtp packet: %v", err)
			continue
		}

		// https://datatracker.ietf.org/doc/html/rfc5761
		if isRTCP := p.PayloadType >= 64 && p.PayloadType <= 95; isRTCP {
			cp, err := rtcp.Unmarshal(buf[0:n])
			if err != nil {
				log.Printf("received invalid rtcp packet: %v", err)
				continue
			}
			for _, pkt := range cp {
				for _, ssrc := range pkt.DestinationSSRC() {
					s.peerManager.MarkReceived(sender, internal.SSRC(ssrc), n)
					// check if this packet has a cname associated with it.
					if sdes, ok := pkt.(*rtcp.SourceDescription); ok {
						for _, chunk := range sdes.Chunks {
							for _, item := range chunk.Items {
								if item.Type == rtcp.SDESCNAME && !s.peerManager.IsConfigured(internal.SSRC(ssrc)) {
									cname := item.Text
									ctx, cancel := context.WithCancel(context.Background())

									pipeline := &internal.Pipeline{
										RTPSink: func(buffer []byte) (int, error) {
											s.OnRTP(cname, buffer)
											return len(buffer), nil
										},
										RTCPSink: func(buffer []byte) (int, error) {
											// choose peers to send the rtcp packet to.
											pkts, err := rtcp.Unmarshal(buffer)
											if err != nil {
												return 0, err
											}
											for _, pkt := range pkts {
												for _, ssrc := range pkt.DestinationSSRC() {
													for _, peer := range s.peerManager.GetPeersForSSRC(internal.SSRC(ssrc)) {
														if _, err := pc.WriteTo(buffer, peer); err != nil {
															log.Printf("failed to write rtcp packet: %v", err)
														}
													}
												}
											}
											return len(buffer), nil
										},
									}

									pipeline.Start(ctx, pipelineType)
									
									s.peerManager.Configure(internal.SSRC(ssrc), internal.CName(cname), pipeline, cancel)

									log.Printf("configured peer %s for ssrc %d", cname, ssrc)
								}
							}
						}
					}

					pipeline, err := s.peerManager.GetPipeline(internal.SSRC(ssrc))
					// check if we have a pipeline for this cname.
					if err != nil  || pipeline == nil {
						log.Printf("failed to get pipeline for ssrc: %v", err)
						continue
					} else if err := pipeline.WriteRTP(buf[:n]); err != nil {
						log.Printf("failed to write rtp packet to pipeline: %v", err)
					}
				}
			}
		} else {
			// ensure the peer is bound to the ssrc.
			s.peerManager.MarkReceived(sender, internal.SSRC(p.SSRC), n)

			pipeline, err := s.peerManager.GetPipeline(internal.SSRC(p.SSRC))
			if err != nil || pipeline == nil {
				log.Printf("received rtp packet with SSRC %d before pipeline construction %v", p.SSRC, err)
			} else if err := pipeline.WriteRTP(buf[:n]); err != nil {
				log.Printf("failed to write rtp packet to pipeline: %v", err)
			}
		}
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: receiver <video addr> <audio addr> <destination>")
		os.Exit(1)
	}

	engine := ion.NewEngine(ion.Config{
		WebRTC: ion.WebRTCTransportConfig{
			Configuration: webrtc.Configuration{
				ICEServers: []webrtc.ICEServer{
					{URLs: []string{"stun:stun.stunprotocol.org:3478", "stun:stun.l.google.com:19302"}},
				},
			},
		},
	})

	clientManager := internal.NewClientManager(context.Background(), 5*time.Second, engine, os.Args[3])

	videoServer := &Server{
		peerManager: internal.NewPeerManager(context.Background(), 5*time.Second, 2*time.Second, "video"),
		OnRTP: func(cname string, buf []byte) {
			client, err := clientManager.GetClient(cname)
			if err != nil {
				log.Printf("failed to get client for cname %v: %v", cname, err)
				return
			}
			if _, err := client.VideoTrack.Write(buf); err != nil {
				log.Printf("failed to write video buffer to track: %v", err)
			}
		},
	}

	audioServer := &Server{
		peerManager: internal.NewPeerManager(context.Background(), 5*time.Second, 2*time.Second, "audio"),
		OnRTP: func(cname string, buf []byte) {
			client, err := clientManager.GetClient(cname)
			if err != nil {
				log.Printf("failed to get client for cname %v: %v", cname, err)
				return
			}
			if _, err := client.AudioTrack.Write(buf); err != nil {
				log.Printf("failed to write audio buffer to track: %v", err)
			}
		},
	}

	// listen for incoming rtp connections.
	go videoServer.Listen(os.Args[1], internal.PipelineTypeVideo)
	go audioServer.Listen(os.Args[2], internal.PipelineTypeAudio)

	// block.
	select {}
}

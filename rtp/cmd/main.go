package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/muxable/mediatun/rtp/internal"
	"github.com/pion/interceptor"
	ion "github.com/pion/ion-sdk-go"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type ListenType uint32

const (
	ListenTypeVideo = ListenType(0)
	ListenTypeAudio = ListenType(1)
)

type Sink struct {
	rtp  interceptor.RTPReader
	rtcp interceptor.RTCPReader
}

type Source struct {
	bitsRecv uint64
	cname    string
}

type Server struct {
	peerManager *internal.PeerManager
	sinks       map[uint32]*Sink
	OnBuffer    func(string, []byte, time.Duration)
}

func (s *Server) Listen(addr string) {
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
					s.peerManager.MarkReceived(sender, ssrc, n)
					// check if this packet has a cname associated with it.
					if sdes, ok := pkt.(*rtcp.SourceDescription); ok {
						for _, chunk := range sdes.Chunks {
							for _, item := range chunk.Items {
								if item.Type == rtcp.SDESCNAME {
									s.peerManager.SetCNAME(ssrc, item.Text)
								}
							}
						}
					}
					// write the packet.
					if sink, ok := s.sinks[ssrc]; ok {
						// forward the rtp packet to the cname's appsrc.
						if _, _, err := sink.rtcp.Read(buf[:n], nil); err != nil {
							log.Printf("failed to write rtp packet to pipeline: %v", err)
						}
					} else {
						chain := interceptor.NewChain([]interceptor.Interceptor{})

						// create a new pipeline for this ssrc.
						pipeline := &internal.Pipeline{
							BufferSink: func(buffer []byte, duration time.Duration) {
								cname, err := s.peerManager.GetCNAME(ssrc)
								if err != nil {
									log.Printf("failed to get cname for ssrc %d: %v", ssrc, err)
								}
								s.OnBuffer(cname, buffer, duration)
							},
							RTCPSink: chain.BindRTCPWriter(interceptor.RTCPWriterFunc(func(pkts []rtcp.Packet, _ interceptor.Attributes) (int, error) {
								// choose peers to send the rtcp packet to.
								buf, err := rtcp.Marshal(pkts)
								if err != nil {
									return 0, err
								}
								for _, peer := range s.peerManager.GetPeersForSSRC(ssrc) {
									if _, err := pc.WriteTo(buf, peer); err != nil {
										log.Printf("failed to write rtcp packet: %v", err)
									}
								}
								return len(buf), nil
							})),
						}

						pipeline.Start(internal.PipelineTypeVideo)

						sink := &Sink{
							rtp: chain.BindRemoteStream(&interceptor.StreamInfo{SSRC: ssrc, RTPHeaderExtensions: []interceptor.RTPHeaderExtension{
								{
									URI: "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
									ID:  1,
								},
							}}, interceptor.RTPReaderFunc(func(b []byte, _ interceptor.Attributes) (int, interceptor.Attributes, error) {
								if err := pipeline.WriteRTP(b); err != nil {
									log.Printf("failed to write rtp packet to pipeline: %v", err)
									return 0, nil, err
								}
								return len(b), nil, nil
							})),
							rtcp: chain.BindRTCPReader(interceptor.RTCPReaderFunc(func(b []byte, _ interceptor.Attributes) (int, interceptor.Attributes, error) {
								if err := pipeline.WriteRTCP(b); err != nil {
									log.Printf("failed to write rtcp packet to pipeline: %v", err)
									return 0, nil, err
								}
								return len(b), nil, nil
							})),
						}
						s.sinks[ssrc] = sink
						if _, _, err := sink.rtcp.Read(buf[:n], nil); err != nil {
							log.Printf("failed to write rtp packet to pipeline: %v", err)
						}
					}
				}
			}
			continue
		} else {
			// ensure the peer is bound to the ssrc.
			s.peerManager.MarkReceived(sender, p.SSRC, n)

			// write the packet.
			if sink, ok := s.sinks[p.SSRC]; ok {
				// forward the rtp packet to the cname's appsrc.
				if _, _, err := sink.rtp.Read(buf[:n], nil); err != nil {
					log.Printf("failed to write rtp packet to pipeline: %v", err)
				}
			} else {
				log.Printf("received rtp package for ssrc %d before rtcp cname data", p.SSRC)
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

	clientManager := internal.NewClientManager(5*time.Second, engine, os.Args[3])

	videoServer := &Server{
		peerManager: internal.NewPeerManager(context.Background(), 5*time.Second, 2*time.Second),
		sinks:       make(map[uint32]*Sink),
		OnBuffer: func(cname string, buf []byte, duration time.Duration) {
			client, err := clientManager.GetClient(cname)
			if err != nil {
				log.Printf("failed to get client for cname %v: %v", cname, err)
				return
			}
			log.Printf("writing sample len %d to %s", len(buf), cname)
			if err := client.VideoTrack.WriteSample(media.Sample{Data: buf, Duration: duration}); err != nil {
				log.Printf("failed to write audio buffer to track: %v", err)
			}
		},
	}

	audioServer := &Server{
		peerManager: internal.NewPeerManager(context.Background(), 5*time.Second, 2*time.Second),
		sinks:       make(map[uint32]*Sink),
		OnBuffer: func(cname string, buf []byte, duration time.Duration) {
			client, err := clientManager.GetClient(cname)
			if err != nil {
				log.Printf("failed to get client for cname %v: %v", cname, err)
				return
			}
			if err := client.AudioTrack.WriteSample(media.Sample{Data: buf, Duration: duration}); err != nil {
				log.Printf("failed to write audio buffer to track: %v", err)
			}
		},
	}

	// listen for incoming rtp connections.
	go videoServer.Listen(os.Args[1])
	go audioServer.Listen(os.Args[2])

	// block.
	select {}
}

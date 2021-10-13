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
	"github.com/pion/webrtc/v3"
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
	pipelineManager := internal.NewPipelineManager(context.Background(), peerManager, clientManager, pc)

	internal.StatsManager{
		ClientManager: clientManager,
		PeerManager:   peerManager,
		Interval:      5 * time.Second,
	}.Run(context.Background())

	for {
		// read in a udp message.
		buf := make([]byte, 1500)
		n, sender, err := pc.ReadFrom(buf)
		if err != nil {
			panic(err)
		}

		if _, err := pipelineManager.Write(sender, buf[:n]); err != nil {
			log.Printf("failed to write rtp packet to pipeline: %v", err)
		}
	}
}

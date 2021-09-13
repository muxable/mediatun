package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	gst "github.com/muxable/mediatun/protocols/internal/rtmp"
	ion "github.com/pion/ion-sdk-go"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

func main() {
	source := flag.String("source", "sfu:50051", "the source to egress from")
	id := flag.String("id", "", "the id of the tunnel to egress to")
	destination := flag.String("destination", "", "the rtmp destination url to egress to")

	flag.Parse()

	engine := ion.NewEngine(ion.Config{
		WebRTC: ion.WebRTCTransportConfig{
			Configuration: webrtc.Configuration{
				ICEServers: []webrtc.ICEServer{
					{URLs: []string{"stun:stun.l.google.com:19302"}},
				},
			},
		},
	})
	client, err := ion.NewClient(engine, *source, uuid.New().String())
	if err != nil {
		panic(err)
	}
	if err = engine.AddClient(client); err != nil {
		panic(err)
	}

	publisher := client.GetPubTransport().GetPeerConnection()

	publisher.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("Connection state changed: %s", state)
	})

	audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "rtmp-audio")
	if err != nil {
		panic(err)
	}
	if _, err = publisher.AddTrack(audioTrack); err != nil {
		panic(err)
	}

	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "video/vp8"}, "video", "rtmp-video")
	if err != nil {
		panic(err)
	}
	if _, err = publisher.AddTrack(videoTrack); err != nil {
		panic(err)
	}

	pipeline := gst.CreatePipeline("rtmpsink location=" + *destination)
	pipeline.Start()

	client.OnTrack = func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			for range ticker.C {
				rtcpSendErr := client.GetSubTransport().GetPeerConnection().WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
				if rtcpSendErr != nil {
					fmt.Println(rtcpSendErr)
				}
			}
		}()

		codecName := strings.Split(track.Codec().RTPCodecCapability.MimeType, "/")[1]
		fmt.Printf("Track has started, of type %d: %s \n", track.PayloadType(), codecName)
		buf := make([]byte, 1400)
		for {
			n, _, err := track.Read(buf)
			if err != nil {
				log.Printf("read error: %v", err)
				break
			}

			switch track.Kind() {
			case webrtc.RTPCodecTypeAudio:
				pipeline.PushAudio(buf[:n])
			case webrtc.RTPCodecTypeVideo:
				pipeline.PushVideo(buf[:n])
			}
		}
	}

	if err = client.Join(*id, ion.NewJoinConfig().SetNoPublish()); err != nil {
		panic(err)
	}

	select {}
}

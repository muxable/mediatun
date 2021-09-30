package main

import (
	"context"
	"log"
	"net/url"
	"time"

	"code.videolan.org/rist/ristgo"
	"code.videolan.org/rist/ristgo/libristwrapper"
	"github.com/muxable/mediatun/rist/internal"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

func main() {
	options := make(map[string]string)
	options["blocking"] = "0"
	options["transtype"] = "live"

	receiver, err := ristgo.ReceiverCreate(context.Background(), &ristgo.ReceiverConfig{
		RistProfile:             libristwrapper.RistProfileSimple,
		LoggingCallbackFunction: ristgo.CreateDefaultLoggingCallback(libristwrapper.LogLevelInfo),
		StatsCallbackFunction:   statsCallback,
		StatsInterval:           1 * time.Second,
		RecoveryBufferSize:      gRistrecoverysize,
	})
	if err != nil {
		log.Fatalf("Error creating receiver: %v", err)
		return
	}
	receiver.Start()
	inputurl, err := url.Parse("rist://@:4000")
	if err != nil {
		log.Fatalf("Error parsing input url: %v", err)
		return
	}
	peerConfig, err := ristgo.ParseRistURL(inputurl)
	if err != nil {
		log.Fatalf("Error parsing input url: %v", err)
		return
	}
	receiver.AddPeer(peerConfig)

	receiver.ConfigureFlow(0)

	engine := ion.NewEngine(ion.Config{
		WebRTC: ion.WebRTCTransportConfig{
			Configuration: webrtc.Configuration{
				ICEServers: []webrtc.ICEServer{
					{URLs: []string{"stun:stun.l.google.com:19302"}},
				},
			},
		},
	})

	if err := socket.Listen(1); err != nil {
		log.Printf("failed to listen on srt socket: %v", err)
		return
	}

	for {
		socket, _, err := socket.Accept()
		if err != nil {
			log.Printf("failed to accept srt socket: %v", err)
			return
		}
		go func() {
			cname, err := socket.GetSockOptString(srtgo.SRTO_STREAMID)
			if err != nil || cname == "" {
				log.Printf("failed to get cname: %v", err)
				return
			}
			client, err := ion.NewClient(engine, "mtun.io:50051", cname)
			if err != nil {
				log.Printf("failed to create client: %v", err)
				return
			}
			defer client.Close()
			if err := client.Join(cname, ion.NewJoinConfig().SetNoSubscribe()); err != nil {
				log.Printf("failed to join: %v", err)
				return
			}

			videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "video/vp8"}, "video", "video")
			if err != nil {
				log.Printf("failed to create video track: %v", err)
				return
			}
			if _, err := client.Publish(videoTrack); err != nil {
				log.Printf("failed to publish: %v", err)
				return
			}
			audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "video")
			if err != nil {
				log.Printf("failed to create audio track: %v", err)
				return
			}
			if _, err := client.Publish(audioTrack); err != nil {
				log.Printf("failed to publish: %v", err)
				return
			}
			pipeline := internal.Pipeline{
				OnVP8Buffer: func(buffer []byte, duration time.Duration) {
					videoTrack.WriteSample(media.Sample{Data: buffer, Duration: duration})
				},
				OnOpusBuffer: func(buffer []byte, duration time.Duration) {
					audioTrack.WriteSample(media.Sample{Data: buffer, Duration: duration})
				},
			}
			pipeline.Start()
			defer pipeline.Close()

			buf := make([]byte, 1500)
			for {
				len, err := socket.Read(buf)
				if err != nil {
					log.Printf("failed to read from srt socket: %v", err)
					return
				}
				if err := pipeline.Write(buf[:len]); err != nil {
					log.Printf("failed to write to pipeline: %v", err)
					return
				}
			}
		}()
	}
}

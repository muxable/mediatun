package main

import (
	"flag"
	"log"

	ion "github.com/pion/ion-sdk-go"
	gst "github.com/pion/ion-sdk-go/pkg/gstreamer-src"
	"github.com/pion/webrtc/v3"
)

func main() {
	source := flag.String("source", "", "the rtmp source url to push from")
	id := flag.String("id", "", "the id of the tunnel to push to")
	destination := flag.String("destination", "sfu:50051", "the destination to push to")

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
	client, err := ion.NewClient(engine, *destination, *id)
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

	if err = client.Join(*id, ion.NewJoinConfig().SetNoSubscribe()); err != nil {
		panic(err)
	}

	gst.CreatePipeline("opus", []*webrtc.TrackLocalStaticSample{audioTrack}, "rtmpsrc location=" + *source + " ! decodebin ! audioconvert ! audioresample").Start()
	gst.CreatePipeline("vp8", []*webrtc.TrackLocalStaticSample{videoTrack}, "rtmpsrc location=" + *source + " ! decodebin").Start()

	select {}
}

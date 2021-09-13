package main

import (
	"context"
	"flag"

	"github.com/muxable/mediatun/ingest/internal/gst"
	ion "github.com/muxable/mediatun/ingest/internal/ion-sdk-go"
	"github.com/pion/ion-sfu/cmd/signal/grpc/proto"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"google.golang.org/grpc"
)

func main() {
	source := flag.String("source", "", "the rtmp source url to push from")
	id := flag.String("id", "", "the id of the tunnel to push to")
	destination := flag.String("destination", "sfu:50051", "the destination to push to")

	flag.Parse()

	videoPipeline := gst.CreatePipeline("rtmpsrc location=" + *source + " ! decodebin ! vp8enc error-resilient=partitions keyframe-max-dist=10 auto-alt-ref=true cpu-used=5 deadline=1 ! appsink name=appsink")
	audioPipeline := gst.CreatePipeline("rtmpsrc location=" + *source + " ! decodebin ! audioconvert ! audioresample ! opusenc ! appsink name=appsink")

	conn, err := grpc.Dial(*destination, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	rpc := proto.NewSFUClient(conn)
	client, err := ion.NewClient(context.Background(), rpc, webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	client.Join(*id, *id)
	
	opusTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "pion1")
	if err != nil {
		panic(err)
	} else if err = client.AddTrack(opusTrack); err != nil {
		panic(err)
	}

	vp8Track, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "video/vp8"}, "video", "pion2")
	if err != nil {
		panic(err)
	} else if err = client.AddTrack(vp8Track); err != nil {
		panic(err)
	}

	videoPipeline.SetOnSample(func(sample media.Sample) {
		vp8Track.WriteSample(sample)
	})
	audioPipeline.SetOnSample(func(sample media.Sample) {
		opusTrack.WriteSample(sample)
	})

	videoPipeline.Start()
	audioPipeline.Start()

	select {}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/muxable/mediatun/ingest/internal/gst"
	ion "github.com/muxable/mediatun/ingest/internal/ion-sdk-go"
	"github.com/pion/ion-sfu/cmd/signal/grpc/proto"
	"github.com/pion/webrtc/v3"
	"google.golang.org/grpc"
)

func main() {
	id := flag.String("id", "", "the tunnel id to pull")
	destination := flag.String("destination", "", "the rtmp url of the destination to pull to")

	flag.Parse()

	conn, err := grpc.Dial("sfu:50051", grpc.WithInsecure())
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

	client.Join(*id, uuid.New().String())

	videoPipeline := "appsrc format=time is-live=true do-timestamp=true name=videosrc ! application/x-rtp,encoding-name=VP8-DRAFT-IETF-01 ! rtpvp8depay ! queue ! videoconvert ! x264enc ! h264parse ! mux."
	audioPipeline := "appsrc format=time is-live=true do-timestamp=true name=videosrc ! application/x-rtp,encoding-name=OPUS ! rtpopusdepay  ! queue ! audioconvert ! avenc_aac ! aacparse ! mux."
	pipeline := gst.CreatePipeline(fmt.Sprintf("%s %s flvmux name=mux ! rtmpsink location=%s", videoPipeline, audioPipeline, *destination))
	pipeline.Start()

	client.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		// go func() {
		// 	ticker := time.NewTicker(time.Second * 3)
		// 	for range ticker.C {
		// 		rtcpSendErr := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
		// 		if rtcpSendErr != nil {
		// 			fmt.Println(rtcpSendErr)
		// 		}
		// 	}
		// }()

		codecName := strings.Split(track.Codec().RTPCodecCapability.MimeType, "/")[1]
		fmt.Printf("Track has started, of type %d: %s \n", track.PayloadType(), codecName)
		buf := make([]byte, 1400)
		for {
			n, _, err := track.Read(buf)
			if err != nil {
				break
			}

			fmt.Printf("Read %d bytes", n)
			// switch track.Kind() {
			// case webrtc.RTPCodecTypeVideo:
			// 	pipeline.PushVideo(buf[:i])
			// case webrtc.RTPCodecTypeAudio:
			// 	pipeline.PushAudio(buf[:i])
			// }
		}
	})

	select {}
}

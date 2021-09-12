package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"

	"github.com/muxable/mediatun/ingest/rtmp/internal/gst"
	"github.com/pion/ion-sfu/cmd/signal/grpc/proto"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"google.golang.org/grpc"
)

func main() {
	source := flag.String("source", "", "the rtmp source url to push from")
	id := flag.String("id", "", "the id of the tunnel to push to")

	flag.Parse()

	// read the source with gstreamer and convert to vp8/opus.

	videoPipeline := gst.CreatePipeline("rtmpsrc location=" + *source + " ! flvdemux ! h264parse ! decodebin ! vp8enc error-resilient=partitions keyframe-max-dist=10 auto-alt-ref=true cpu-used=5 deadline=1 ! appsink name=appsink")
	audioPipeline := gst.CreatePipeline("rtmpsrc location=" + *source + " ! flvdemux ! aacparse ! avdec_aac ! audioresample ! audioconvert ! opusenc ! appsink name=appsink")

	conn, err := grpc.Dial("sfu:50051", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	rpc := proto.NewSFUClient(conn)
	client, err := rpc.Signal(context.Background())
	if err != nil {
		panic(err)
	}

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("Connection State has changed %s \n", connectionState.String())
	})

	go func() {
		for {
			message, err := client.Recv()
			if err != nil {
				panic(err)
			}
			switch payload := message.Payload.(type) {
			case *proto.SignalReply_Join:
				// set the remote description
				sdp := webrtc.SessionDescription{}
				if err := json.Unmarshal(payload.Join.Description, &sdp); err != nil {
					panic(err)
				}
				if err := peerConnection.SetRemoteDescription(sdp); err != nil {
					panic(err)
				}
			case *proto.SignalReply_Description:
				// set the remote description and send an answer.
				sdp := webrtc.SessionDescription{}
				if err := json.Unmarshal(payload.Description, &sdp); err != nil {
					panic(err)
				}
				if err = peerConnection.SetRemoteDescription(sdp); err != nil {
					panic(err)
				}
				if sdp.Type == webrtc.SDPTypeAnswer {
					break
				}
				answer, err := peerConnection.CreateAnswer(nil)
				if err != nil {
					panic(err)
				}
				description, err := json.Marshal(answer)
				if err != nil {
					panic(err)
				}
				client.Send(&proto.SignalRequest{Payload: &proto.SignalRequest_Description{Description: description}})
			case *proto.SignalReply_Trickle:
				// add the ice candidate.
				if payload.Trickle.Target == proto.Trickle_PUBLISHER {
					panic("unexpected trickle for publisher")
				}
				candidate := webrtc.ICECandidateInit{}
				if err := json.Unmarshal([]byte(payload.Trickle.Init), &candidate); err != nil {
					panic(err)
				}
				peerConnection.AddICECandidate(candidate)
			}
		}
	}()

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		init, err := json.Marshal(candidate)
		if err != nil {
			panic(err)
		}
		client.Send(&proto.SignalRequest{Payload: &proto.SignalRequest_Trickle{Trickle: &proto.Trickle{Target: proto.Trickle_PUBLISHER, Init: string(init)}}})
	})
	
	peerConnection.OnNegotiationNeeded(func() {
		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			panic(err)
		}

		if err = peerConnection.SetLocalDescription(offer); err != nil {
			panic(err)
		}

		// Issue a grpc command to the sfu.
		description, err := json.Marshal(*peerConnection.LocalDescription())
		if err != nil {
			panic(err)
		}

		client.Send(&proto.SignalRequest{
			Payload: &proto.SignalRequest_Join{
				Join: &proto.JoinRequest{
					Sid:         *id,
					Uid:         *id,
					Description: description,
				},
			},
		})
	})

	opusTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "pion1")
	if err != nil {
		panic(err)
	} else if _, err = peerConnection.AddTrack(opusTrack); err != nil {
		panic(err)
	}

	vp8Track, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "video/vp8"}, "video", "pion2")
	if err != nil {
		panic(err)
	} else if _, err = peerConnection.AddTrack(vp8Track); err != nil {
		panic(err)
	}

	// Start pushing buffers on these tracks
	videoPipeline.SetOnSample(func(sample media.Sample) {
		vp8Track.WriteSample(sample)
	})
	audioPipeline.SetOnSample(func(sample media.Sample) {
		opusTrack.WriteSample(sample)
	})

	videoPipeline.Start()
	audioPipeline.Start()

	// Block forever
	select {}
}

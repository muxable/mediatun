package ion_sdk_go

import (
	"encoding/json"
	"log"

	"github.com/pion/ion-sfu/cmd/signal/grpc/proto"
	"github.com/pion/webrtc/v3"
)

const apiChannel = "ion-sfu"

type Transport struct {
	api        *webrtc.DataChannel
	signal     proto.SFU_SignalClient
	pc         *webrtc.PeerConnection
	candidates []webrtc.ICECandidateInit
}

func NewTransport(role proto.Trickle_Target, signal proto.SFU_SignalClient, config webrtc.Configuration) (*Transport, error) {
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	t := &Transport{
		signal: signal,
		pc:     pc,
	}

	if role == proto.Trickle_PUBLISHER {
		pc.CreateDataChannel(apiChannel, nil)
	}

	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		init, err := json.Marshal(candidate)
		if err != nil {
			log.Fatalf("failed to marshal candidate %v", err)
			return
		}

		signal.Send(&proto.SignalRequest{
			Payload: &proto.SignalRequest_Trickle{
				Trickle: &proto.Trickle{
					Target: role,
					Init:   string(init),
				},
			},
		})
	})

	return t, nil
}

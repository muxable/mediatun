package ion_sdk_go

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/pion/ion-sfu/cmd/signal/grpc/proto"
	"github.com/pion/webrtc/v3"
)

type Client struct {
	transports map[proto.Trickle_Target]*Transport
	config     webrtc.Configuration
	signal     proto.SFU_SignalClient

	joinReplyCh     chan webrtc.SessionDescription
	onDescriptionCh chan webrtc.SessionDescription

	onTrack        func(*webrtc.TrackRemote, *webrtc.RTPReceiver)
	onErrNegotiate func(proto.Trickle_Target, error)
}

func NewClient(ctx context.Context, sfu proto.SFUClient, config webrtc.Configuration) (*Client, error) {
	signal, err := sfu.Signal(ctx)
	if err != nil {
		return nil, err
	}

	client := &Client{
		transports: make(map[proto.Trickle_Target]*Transport),
		config:     config,
		signal:     signal,
	}

	go func() {
		for {
			reply, err := signal.Recv()
			if err != nil {
				break
			}
			switch message := reply.Payload.(type) {
			case *proto.SignalReply_Join:
				answer := webrtc.SessionDescription{}
				if err := json.Unmarshal(message.Join.Description, &answer); err != nil {
					log.Fatalf("invalid json from sfu: %v", err)
					break
				}
				if client.joinReplyCh == nil {
					log.Fatalf("not expecting a join reply")
					break
				}
				client.joinReplyCh <- answer
			case *proto.SignalReply_Description:
				desc := webrtc.SessionDescription{}
				if err := json.Unmarshal(message.Description, &desc); err != nil {
					log.Fatalf("invalid json from sfu: %v", err)
					break
				}
				switch desc.Type {
				case webrtc.SDPTypeOffer:
					if err := client.negotiate(desc); err != nil && client.onErrNegotiate != nil {
						client.onErrNegotiate(proto.Trickle_PUBLISHER, err)
					}
				case webrtc.SDPTypeAnswer:
					if client.onDescriptionCh == nil {
						log.Fatalf("not expecting an answer")
						break
					}
					client.onDescriptionCh <- desc
				}
			case *proto.SignalReply_Trickle:
				trickle := message.Trickle
				if trickle.Init != "" {
					candidate := webrtc.ICECandidateInit{}
					if err := json.Unmarshal([]byte(trickle.Init), &candidate); err != nil {
						log.Fatalf("invalid json from sfu: %v", err)
						break
					}
					client.trickle(trickle.Target, candidate)
				}
			}
		}
	}()

	return client, nil
}

func (c *Client) Join(sid, uid string) error {
	publisher, err := NewTransport(proto.Trickle_PUBLISHER, c.signal, c.config)
	if err != nil {
		return err
	}
	subscriber, err := NewTransport(proto.Trickle_SUBSCRIBER, c.signal, c.config)
	if err != nil {
		return err
	}
	c.transports = map[proto.Trickle_Target]*Transport{
		proto.Trickle_PUBLISHER:  publisher,
		proto.Trickle_SUBSCRIBER: subscriber,
	}

	subscriber.pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		if c.onTrack != nil {
			c.onTrack(track, receiver)
		}
	})

	subscriber.pc.OnDataChannel(func(channel *webrtc.DataChannel) {
		if channel.Label() == apiChannel {
			subscriber.api = channel
			publisher.api = channel
			channel.OnMessage(func(msg webrtc.DataChannelMessage) {
				log.Printf("got message: %s", string(msg.Data))
			})
		}
	})

	offer, err := publisher.pc.CreateOffer(nil)
	if err != nil {
		return err
	}
	if err := publisher.pc.SetLocalDescription(offer); err != nil {
		return err
	}
	description, err := json.Marshal(offer)
	if err != nil {
		return err
	}
	c.joinReplyCh = make(chan webrtc.SessionDescription)
	c.signal.Send(&proto.SignalRequest{
		Payload: &proto.SignalRequest_Join{
			Join: &proto.JoinRequest{
				Sid:         sid,
				Uid:         uid,
				Description: description,
			},
		},
	})
	answer := <-c.joinReplyCh
	close(c.joinReplyCh)
	if err := publisher.pc.SetRemoteDescription(answer); err != nil {
		return err
	}
	for _, candidate := range publisher.candidates {
		if err := publisher.pc.AddICECandidate(candidate); err != nil {
			return err
		}
	}
	publisher.pc.OnNegotiationNeeded(func() {
		if err := c.onNegotiationNeeded(); err != nil && c.onErrNegotiate != nil {
			// TODO: onErrorNegotiate
			c.onErrNegotiate(proto.Trickle_PUBLISHER, err)
		}
	})
	return nil
}

func (c *Client) AddTrack(track webrtc.TrackLocal) error {
	if c.transports == nil {
		return fmt.Errorf("no session started")
	}
	if _, err := c.transports[proto.Trickle_PUBLISHER].pc.AddTrack(track); err != nil {
		return err
	}
	return nil
}

func (c *Client) OnTrack(callback func(*webrtc.TrackRemote, *webrtc.RTPReceiver)) {
	c.onTrack = callback
}

func (c *Client) OnErrNegotiate(callback func(proto.Trickle_Target, error)) {
	c.onErrNegotiate = callback
}

func (c *Client) trickle(target proto.Trickle_Target, candidate webrtc.ICECandidateInit) error {
	if c.transports == nil {
		return fmt.Errorf("no session started")
	}
	transport := c.transports[target]
	if transport.pc.RemoteDescription() != nil {
		transport.pc.AddICECandidate(candidate)
	} else {
		transport.candidates = append(transport.candidates, candidate)
	}
	return nil
}

func (c *Client) negotiate(desc webrtc.SessionDescription) error {
	if c.transports == nil {
		return fmt.Errorf("no session started")
	}

	sub := c.transports[proto.Trickle_SUBSCRIBER]
	sub.pc.SetRemoteDescription(desc)
	for _, candidate := range sub.candidates {
		sub.pc.AddICECandidate(candidate)
	}
	sub.candidates = make([]webrtc.ICECandidateInit, 0)
	answer, err := sub.pc.CreateAnswer(nil)
	if err != nil {
		return err
	}
	if err := sub.pc.SetLocalDescription(answer); err != nil {
		return err
	}
	description, err := json.Marshal(answer)
	if err != nil {
		return err
	}
	c.signal.Send(&proto.SignalRequest{
		Payload: &proto.SignalRequest_Description{
			Description: description,
		},
	})
	return nil
}

func (c *Client) onNegotiationNeeded() error {
	if c.transports == nil {
		return fmt.Errorf("no session started")
	}

	pub := c.transports[proto.Trickle_PUBLISHER]
	offer, err := pub.pc.CreateOffer(nil)
	if err != nil {
		return err
	}
	if err := pub.pc.SetLocalDescription(offer); err != nil {
		return err
	}
	description, err := json.Marshal(offer)
	if err != nil {
		return err
	}
	id := uuid.New()
	c.onDescriptionCh = make(chan webrtc.SessionDescription)
	c.signal.Send(&proto.SignalRequest{
		Id:      id.String(),
		Payload: &proto.SignalRequest_Description{Description: description},
	})
	answer := <-c.onDescriptionCh
	close(c.onDescriptionCh)
	return pub.pc.SetRemoteDescription(answer)
}

func (c *Client) Leave() error {
	for _, transport := range c.transports {
		if err := transport.pc.Close(); err != nil {
			return err
		}
	}
	c.transports = nil
	return nil
}

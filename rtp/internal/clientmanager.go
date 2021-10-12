package internal

import (
	"context"
	"log"
	"sync"
	"time"

	ion "github.com/pion/ion-sdk-go"
	"github.com/pion/webrtc/v3"
)

type Client struct {
	sync.Mutex

	lastUpdated time.Time
	sdk         *ion.Client

	VideoTrack *webrtc.TrackLocalStaticSample
	AudioTrack *webrtc.TrackLocalStaticSample
}

type ClientManager struct {
	sync.Mutex

	engine *ion.Engine
	addr   Address

	clients map[CName]*Client
}

func NewClientManager(ctx context.Context, timeout time.Duration, engine *ion.Engine, addr Address) *ClientManager {
	m := &ClientManager{
		clients: make(map[CName]*Client),
		engine:  engine,
		addr:    addr,
	}
	// start a cleanup routine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(timeout):
				m.Lock()
				// remove all ssrcs that have not been updated in the last timeout.
				for cname, client := range m.clients {
					if time.Since(client.lastUpdated) > timeout {
						delete(m.clients, cname)
					}
				}
				m.Unlock()
			}
		}
	}()
	return m
}

func (m *ClientManager) GetClient(cname CName) (*Client, error) {
	m.Lock()
	defer m.Unlock()

	if client, ok := m.clients[cname]; ok {
		client.lastUpdated = time.Now()
	} else {
		sdk, err := ion.NewClient(m.engine, string(m.addr), string(cname))
		if err != nil {
			return nil, err
		}

		peerConnection := sdk.GetPubTransport().GetPeerConnection()

		peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
			log.Printf("Connection state changed: %s", state)
		})

		videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "video")
		if err != nil {
			return nil, err
		}
		if _, err = peerConnection.AddTrack(videoTrack); err != nil {
			return nil, err
		}

		audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "audio")
		if err != nil {
			return nil, err
		}
		if _, err = peerConnection.AddTrack(audioTrack); err != nil {
			return nil, err
		}

		if err := sdk.Join(string(cname), ion.NewJoinConfig().SetNoSubscribe()); err != nil {
			return nil, err
		}

		m.clients[cname] = &Client{
			lastUpdated: time.Now(),
			sdk:         sdk,
			VideoTrack:  videoTrack,
			AudioTrack:  audioTrack,
		}
	}
	return m.clients[cname], nil
}

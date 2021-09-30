package internal

import (
	"log"
	"sync"
	"time"

	ion "github.com/pion/ion-sdk-go"
	"github.com/pion/webrtc/v3"
)

type Client struct {
	sync.Mutex

	sdk         *ion.Client
	lastUpdated time.Time

	VideoTrack *webrtc.TrackLocalStaticSample
	AudioTrack *webrtc.TrackLocalStaticSample
}

type ClientManager struct {
	sync.Mutex

	engine *ion.Engine
	addr   string

	clients map[string]*Client

	cancel chan bool
}

func NewClientManager(timeout time.Duration, engine *ion.Engine, addr string) *ClientManager {
	cancel := make(chan bool)
	m := &ClientManager{
		clients: make(map[string]*Client),
		engine:  engine,
		addr:    addr,
		cancel:  cancel,
	}
	// start a cleanup routine
	go func() {
		for {
			select {
			case <-cancel:
				close(cancel)
				return
			case <-time.After(timeout):
				m.Lock()
				// remove all ssrcs that have not been updated in the last timeout.
				for cname, client := range m.clients {
					if time.Since(client.lastUpdated) > timeout {
						log.Printf("removing client %s", cname)
						delete(m.clients, cname)
					}
				}
				m.Unlock()
			}
		}
	}()
	return m
}

// Close closes the Client manager.
func (m *ClientManager) Close() {
	m.cancel <- true
}

func (m *ClientManager) GetClient(cname string) (*Client, error) {
	m.Lock()
	defer m.Unlock()

	if client, ok := m.clients[cname]; ok {
		client.lastUpdated = time.Now()
	} else {
		sdk, err := ion.NewClient(m.engine, m.addr, cname)
		if err != nil {
			return nil, err
		}

		peerConnection := sdk.GetPubTransport().GetPeerConnection()

		peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
			log.Printf("Connection state changed: %s", state)
		})

		if err = m.engine.AddClient(sdk); err != nil {
			return nil, err
		}

		videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "video/vp8"}, "video", "video")
		if err != nil {
			return nil, err
		}
		if _, err = peerConnection.AddTrack(videoTrack); err != nil {
			return nil, err
		}

		audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "audio")
		if err != nil {
			return nil, err
		}
		if _, err = peerConnection.AddTrack(videoTrack); err != nil {
			return nil, err
		}
		
		m.clients[cname] = &Client{
			sdk:         sdk,
			lastUpdated: time.Now(),
			VideoTrack:  videoTrack,
			AudioTrack:  audioTrack,
		}
		
		sdk.Join(cname, ion.NewJoinConfig().SetNoSubscribe())
	}
	return m.clients[cname], nil
}

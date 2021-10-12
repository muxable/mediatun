package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	rist "github.com/muxable/mediatun/protocols/internal/librist-go"
	ion "github.com/pion/ion-sdk-go"
	"github.com/pion/webrtc/v3"
)

func main() {
	source := flag.String("source", "rist://@:5000", "the rtmp source url to push from")
	destination := flag.String("destination", "sfu:50051", "the destination to push to")

	flag.Parse()

	// Set up RIST.
	profile := rist.PROFILE_MAIN
	r := rist.NewReceiver(profile, nil)
	if r == nil {
		panic("Could not create rist receiver context")
	}

	if !r.SetAuthHandler(&AuthHandler{}) {
		panic("Could not init rist auth handler")
	}

	if !r.SetConnectionStatusHandler(&ConnectionStatusHandler{}) {
		panic("Could not initialize rist connection status callback\n")
	}

	if !r.SetOobHandler(&OobHandler{}) {
		panic("Could not add enable out-of-band data\n")
	}

	if !r.SetStatsHandler(1*time.Second, &StatsHandler{}) {
		panic("Could not enable stats callback\n")
	}

	peerConfig := rist.ParseAddress(*source)
	defer peerConfig.Close()

	if peerConfig == nil {
		panic("Could not parse peer options for receiver")
	}

	peerConfig.SetRecoveryLengthMin(3500)
	peerConfig.SetRecoveryLengthMax(5500)
	peerConfig.SetRecoveryReorderBuffer(200)

	fmt.Printf("Link configured with maxrate=%d bufmin=%d bufmax=%d reorder=%d rttmin=%d rttmax=%d congestion_control=%d min_retries=%d max_retries=%d\n",
		peerConfig.GetRecoveryMaxbitrate(), peerConfig.GetRecoveryLengthMin(), peerConfig.GetRecoveryLengthMax(),
		peerConfig.GetRecoveryReorderBuffer(), peerConfig.GetRecoveryRttMin(), peerConfig.GetRecoveryRttMax(),
		peerConfig.GetCongestionControlMode(), peerConfig.GetMinRetries(), peerConfig.GetMaxRetries())

	peer := r.PeerCreate(peerConfig)
	if peer == nil {
		panic("Could not add peer connector to receiver")
	}

	// Set up the WebRTC connection.
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

	if !r.SetReceiverDataHandler(&ReceiverDataHandler{}) {
		panic("Could not set data_callback pointer\n")
	}

	if !r.Start() {
		panic("Could not start rist receiver")
	}

	select {}
}

type AuthHandler struct {
}

func (h *AuthHandler) OnConnect(connecting_ip string, connecting_port uint16, local_ip string, local_port uint16, peer *rist.Peer) int {
	fmt.Printf("on connect")
	return 0
}

func (h *AuthHandler) OnDisconnect(peer *rist.Peer) int {
	fmt.Printf("on disconnect")
	return 0
}

type ConnectionStatusHandler struct {
}

func (h *ConnectionStatusHandler) OnConnectionStatus(peer *rist.Peer, status rist.ConnectionStatus) {
}

type SessionMap struct {
	sync.RWMutex
	data map[uint16]string
}

type OobHandler struct {
	sessionMap *SessionMap
}

func (h *OobHandler) OnReceiveOob(oobBlock *rist.OobBlock) int {
	b := oobBlock.GetPayload()
	s := struct {
		virtDstPort uint16 `json:"virt_dst_port"`
		cname       string `json:"cname"`
	}{}
	if err := json.Unmarshal(b, &s); err != nil {
		return 0
	}
	h.sessionMap.Lock()
	defer h.sessionMap.Unlock()
	h.sessionMap.data[s.virtDstPort] = s.cname
	return 0
}

type StatsHandler struct {
}

func (h *StatsHandler) OnReceiveStats(stats *rist.Stats) int {
	log.Printf(stats.GetJsonString())
	return 0
}

type ReceiverDataHandler struct {
	sessionMap *SessionMap
}

func (h *ReceiverDataHandler) OnData(data *rist.DataBlock) int {
	// get the cname corresponding to the data payload.
	h.sessionMap.RLock()
	defer h.sessionMap.RUnlock()
	if cname, ok := h.sessionMap.data[data.GetVirtDstPort()]; ok {

	}
		if key.StreamId == data.GetVirtDstPort() {
			pushed = true
			transcoder.PushRaw(data.GetRawPayload())
		}
	return 0
}

// AddTrack adds a track to the Demuxer and starts the pipeline if necessary.
func (d *Demuxer) AddTrack(key Key, track *webrtc.TrackLocalStaticSample) {
	// if there's already an existing transcoder, add the track to the transcoder.
	d.Lock()
	defer d.Unlock()
	if transcoder, ok := d.transcoders[key]; ok {
		transcoder.AddTrack(track)
		return
	}

	// create a new transcoder.
	transcoder := NewTranscoder(key.Codec)
	transcoder.AddTrack(track)
	d.transcoders[key] = transcoder
}

// RemoveTrack removes a track from the Demuxer and stops the pipeline if necessary.
func (d *Demuxer) RemoveTrack(key Key, track *webrtc.TrackLocalStaticSample) {
	d.Lock()
	defer d.Unlock()
	// find the transcoder
	if t, ok := d.transcoders[key]; ok {
		t.Lock()
		defer t.Unlock()
		delete(t.tracks, track)
		if len(t.tracks) == 0 {
			t.pipeline.Stop()
			delete(d.transcoders, key)
		}
	}
}

// Close the demuxer.
func (d *Demuxer) Close() {
	d.rist.Close()
}

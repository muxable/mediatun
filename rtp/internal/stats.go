package internal

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

type StatsManager struct {
	PeerManager   *PeerManager
	ClientManager *ClientManager
	Interval      time.Duration
}

// Run starts the stats manager with a given context to keep it alive.
func (sm StatsManager) Run(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(sm.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sm.printPeerManagerStats()
				sm.printClientManagerStats()
			}
		}
	}()
}

// printPeerManagerStats prints the stats of the peer manager.
func (sm StatsManager) printPeerManagerStats() {
	var builder strings.Builder
	builder.WriteString("---- peers ----\n")
	sm.PeerManager.Lock()
	for ssrc, source := range sm.PeerManager.sources {
		cname := CName("<unset>")
		if source.cname != nil {
			cname = *source.cname
		}
		builder.WriteString(fmt.Sprintf("%d\t-> %s\t%d peers, %d bps\n", ssrc, cname, source.peerCount, source.bitrate*8/int(sm.Interval/time.Second)))
		source.bitrate = 0
	}
	sm.PeerManager.Unlock()
	log.Print(builder.String())
}

// printClientManager prints the stats of the client manager.
func (sm StatsManager) printClientManagerStats() {
	var builder strings.Builder
	builder.WriteString("---- clients ----\n")
	sm.ClientManager.Lock()
	for cname, client := range sm.ClientManager.clients {
		builder.WriteString(fmt.Sprintf("%s (ICE: %s) -> audio: %d bps, video: %d pps\n", cname, client.sdk.GetPubTransport().GetPeerConnection().ConnectionState(), 0, 0))
	}
	sm.ClientManager.Unlock()
	log.Print(builder.String())
}
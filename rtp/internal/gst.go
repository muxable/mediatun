package internal

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0
#include "gst.h"
*/
import "C"
import (
	"context"
	"log"
	"time"
	"unsafe"

	"github.com/mattn/go-pointer"
	"github.com/pion/webrtc/v3/pkg/media"
)

func init() {
	go C.gstreamer_main_loop()
	C.gstreamer_init()
}

type Pipeline struct {
	gstElement *C.GstElement

	RTCPSink       func([]byte) (int, error)
	VP8SampleSink  func(SSRC, media.Sample) (int, error)
	OpusSampleSink func(SSRC, media.Sample) (int, error)
}

func (p *Pipeline) Start(ctx context.Context) error {
	pipelineStr := C.CString(`
		rtpbin name=rtpbin rtp-profile=avpf do-retransmission=true latency=2000 sdes="application/x-rtp-source-sdes,cname=(string)\"mtun.io\""
			appsrc name=rtpappsrc is-live=true format=time caps="application/x-rtp" ! rtpbin.recv_rtp_sink_0
			rtpbin.send_rtcp_src_0 ! appsink name=rtcpappsink`)
	defer C.free(unsafe.Pointer(pipelineStr))

	p.gstElement = C.gstreamer_start(pipelineStr, pointer.Save(p))

	go func() {
		<-ctx.Done()
		C.gstreamer_stop(p.gstElement)
	}()

	return nil
}

// WriteRTP sends the given RTP packet to the pipeline for processing.
func (p *Pipeline) WriteRTP(buf []byte) error {
	b := C.CBytes(buf)
	defer C.free(b)

	codec := C.CString("rtpappsrc")
	defer C.free(unsafe.Pointer(codec))
	C.gstreamer_push_rtp(p.gstElement, codec, b, C.int(len(buf)))

	return nil
}

var vp8Duration = 33333333
var opusDuration = 20000000

//export goHandleVP8Buffer
func goHandleVP8Buffer(buffer unsafe.Pointer, bufferLen C.int, timestamp C.ulong, ssrc C.ulong, data unsafe.Pointer) {
	if _, err := pointer.Restore(data).(*Pipeline).VP8SampleSink(SSRC(ssrc), media.Sample{Data: C.GoBytes(buffer, bufferLen), Duration: time.Duration(vp8Duration)}); err != nil {
		log.Printf("failed to write rtp packet: %v", err)
	}
}

//export goHandleOpusBuffer
func goHandleOpusBuffer(buffer unsafe.Pointer, bufferLen C.int, timestamp C.ulong, ssrc C.ulong, data unsafe.Pointer) {
	if _, err := pointer.Restore(data).(*Pipeline).OpusSampleSink(SSRC(ssrc), media.Sample{Data: C.GoBytes(buffer, bufferLen), Duration: time.Duration(opusDuration)}); err != nil {
		log.Printf("failed to write rtp packet: %v", err)
	}
}

//export goHandleRtcpAppSinkBuffer
func goHandleRtcpAppSinkBuffer(buffer unsafe.Pointer, bufferLen C.int, data unsafe.Pointer) {
	if _, err := pointer.Restore(data).(*Pipeline).RTCPSink(C.GoBytes(buffer, bufferLen)); err != nil {
		log.Printf("failed to write rtcp packet: %v", err)
	}
}
